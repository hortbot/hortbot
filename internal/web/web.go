package web

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gofrs/uuid"
	"github.com/gorilla/sessions"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/web/mid"
	"github.com/hortbot/hortbot/internal/web/templates"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/queries"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"go.uber.org/zap"
)

var botScopes = []string{
	"user_follows_edit",
	"channel:moderate",
	"chat:edit",
	"chat:read",
	"whispers:read",
	"whispers:edit",
}

type App struct {
	Addr       string
	RealIP     bool
	SessionKey []byte

	Redis  *redis.DB
	DB     *sql.DB
	Twitch *twitch.Twitch

	store *sessions.CookieStore
}

func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if len(a.SessionKey) == 0 {
		panic("empty session key")
	}

	a.store = sessions.NewCookieStore(a.SessionKey)

	r := chi.NewRouter()

	logger := ctxlog.FromContext(ctx)
	r.Use(mid.Logger(logger))
	r.Use(mid.RequestID)

	if a.RealIP {
		r.Use(middleware.RealIP)
	}

	r.Use(mid.RequestLogger)
	r.Use(mid.Tracer)
	r.Use(mid.Recoverer)

	r.Get("/", a.index)
	r.Get("/channels", a.channels)

	r.Route("/c/{channel}", func(r chi.Router) {
		r.Use(a.channelMiddleware("channel"))
		r.Get("/", a.channel)
		r.Get("/commands", a.channelCommands)
		r.Get("/quotes", a.channelQuotes)
		r.Get("/autoreplies", a.channelAutoreplies)
		r.Get("/lists", a.channelLists)
		r.Get("/regulars", a.channelRegulars)
		r.Get("/chatrules", a.channelChatRules)
		r.Get("/scheduled", a.channelScheduled)
	})

	r.Get("/login", a.login)
	r.Get("/auth/twitch", a.authTwitch)
	r.Get("/auth/twitch/bot/{botName}", a.authTwitch)
	r.Get("/auth/twitch/callback", a.authTwitchCallback)

	srv := http.Server{
		Addr:    a.Addr,
		Handler: r,
	}

	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			logger.Error("error shutting down server", zap.Error(err))
		}
	}()

	logger.Info("web server listening", zap.String("addr", srv.Addr))

	return srv.ListenAndServe()
}

func (a *App) authTwitch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctxlog.FromContext(ctx)

	state := uuid.Must(uuid.NewV4()).String()

	botName := chi.URLParam(r, "botName")
	if botName != "" {
		state = strings.ToLower(botName) + ":" + state
	}

	if err := a.Redis.SetAuthState(r.Context(), state, time.Minute); err != nil {
		logger.Error("error setting auth state", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}

	var extraScopes []string
	if botName != "" {
		extraScopes = botScopes
	}

	url := a.Twitch.AuthCodeURL(state, extraScopes...)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (a *App) authTwitchCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctxlog.FromContext(ctx)

	state := r.FormValue("state")
	if state == "" {
		httpError(w, http.StatusBadRequest)
		return
	}

	ok, err := a.Redis.CheckAuthState(ctx, state)
	if err != nil {
		logger.Error("error checking auth state", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}

	if !ok {
		httpError(w, http.StatusBadRequest)
		return
	}

	var botName string
	if i := strings.IndexByte(state, ':'); i > 0 {
		botName = strings.ToLower(state[:i])
	}
	isBot := botName != ""

	code := r.FormValue("code")

	tok, err := a.Twitch.Exchange(ctx, code)
	if err != nil {
		logger.Error("error exchanging code", zap.Error(err))
		httpError(w, http.StatusBadRequest)
		return
	}

	user, newToken, err := a.Twitch.GetUserForToken(ctx, tok)
	if err != nil {
		logger.Error("error getting user for token", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}
	if newToken != nil {
		tok = newToken
	}

	tt := modelsx.TokenToModel(user.ID, tok)

	if botName != "" {
		if botName != user.Name {
			httpError(w, http.StatusBadRequest)
			return
		}
		tt.BotName = null.StringFrom(botName)
	}

	if err := modelsx.UpsertToken(ctx, a.DB, tt); err != nil {
		logger.Error("error upserting token", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}

	if !isBot {
		session := a.getSession(r)
		session.setTwitchID(user.ID)
		session.setUsername(user.Name)

		if err := session.save(w, r); err != nil {
			logger.Error("error saving session", zap.Error(err))
			httpError(w, http.StatusInternalServerError)
			return
		}
	}

	templates.WritePageTemplate(w, &templates.LoginSuccessPage{
		Name: user.Name,
		ID:   user.ID,
		Bot:  isBot,
	})
}

func (a *App) index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctxlog.FromContext(ctx)

	session := a.getSession(r)
	logger.Debug("session", zap.Int64("twitch_id", session.getTwitchID()), zap.String("username", session.getUsername()))

	channels, err := models.Channels(models.ChannelWhere.Active.EQ(true)).Count(ctx, a.DB)
	if err != nil {
		logger.Error("error querying channels", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}

	var row struct {
		BotCount int64
	}

	if err := queries.Raw("SELECT COUNT(DISTINCT bot_name) AS bot_count FROM channels WHERE active").Bind(ctx, a.DB, &row); err != nil {
		logger.Error("error querying bot names", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}

	templates.WritePageTemplate(w, &templates.IndexPage{
		ChannelCount: channels,
		BotCount:     row.BotCount,
	})
}

func (a *App) channels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctxlog.FromContext(ctx)

	channels, err := models.Channels(
		models.ChannelWhere.Active.EQ(true),
		qm.OrderBy(models.ChannelColumns.Name),
	).All(ctx, a.DB)
	if err != nil {
		logger.Error("error querying channels", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}

	templates.WritePageTemplate(w, &templates.ChannelsPage{
		Channels: channels,
	})
}

func (a *App) channel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	templates.WritePageTemplate(w, &templates.ChannelPage{
		Channel: channel,
	})
}

func (a *App) channelCommands(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctxlog.FromContext(ctx)
	channel := getChannel(ctx)

	commands, err := channel.CustomCommands(qm.Load(models.CustomCommandRels.CommandInfo)).All(ctx, a.DB)
	if err != nil {
		logger.Error("error querying custom commands", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}

	sort.Slice(commands, func(i, j int) bool {
		return commands[i].R.CommandInfo.Name < commands[j].R.CommandInfo.Name
	})

	templates.WritePageTemplate(w, &templates.ChannelCommandsPage{
		ChannelPage: templates.ChannelPage{
			Channel: channel,
		},
		Commands: commands,
	})
}

func (a *App) channelQuotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctxlog.FromContext(ctx)
	channel := getChannel(ctx)

	quotes, err := channel.Quotes(qm.OrderBy(models.QuoteColumns.Num)).All(ctx, a.DB)
	if err != nil {
		logger.Error("error querying quotes", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}

	templates.WritePageTemplate(w, &templates.ChannelQuotesPage{
		ChannelPage: templates.ChannelPage{
			Channel: channel,
		},
		Quotes: quotes,
	})
}

func (a *App) channelAutoreplies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctxlog.FromContext(ctx)
	channel := getChannel(ctx)

	autoreplies, err := channel.Autoreplies(qm.OrderBy(models.AutoreplyColumns.Num)).All(ctx, a.DB)
	if err != nil {
		logger.Error("error querying autoreplies", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}

	templates.WritePageTemplate(w, &templates.ChannelAutorepliesPage{
		ChannelPage: templates.ChannelPage{
			Channel: channel,
		},
		Autoreplies: autoreplies,
	})
}

func (a *App) channelLists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctxlog.FromContext(ctx)
	channel := getChannel(ctx)

	lists, err := channel.CommandLists(qm.Load(models.CommandListRels.CommandInfo)).All(ctx, a.DB)
	if err != nil {
		logger.Error("error querying command lists", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}

	sort.Slice(lists, func(i, j int) bool {
		return lists[i].R.CommandInfo.Name < lists[j].R.CommandInfo.Name
	})

	templates.WritePageTemplate(w, &templates.ChannelListsPage{
		ChannelPage: templates.ChannelPage{
			Channel: channel,
		},
		Lists: lists,
	})
}

func (a *App) channelRegulars(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	templates.WritePageTemplate(w, &templates.ChannelRegularsPage{
		ChannelPage: templates.ChannelPage{
			Channel: channel,
		},
	})
}

func (a *App) channelChatRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	templates.WritePageTemplate(w, &templates.ChannelRulesPage{
		ChannelPage: templates.ChannelPage{
			Channel: channel,
		},
	})
}

func (a *App) channelScheduled(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctxlog.FromContext(ctx)
	channel := getChannel(ctx)

	repeated, err := channel.RepeatedCommands(qm.Load(models.RepeatedCommandRels.CommandInfo)).All(ctx, a.DB)
	if err != nil {
		logger.Error("error querying repeated commands", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}

	scheduled, err := channel.ScheduledCommands(qm.Load(models.ScheduledCommandRels.CommandInfo)).All(ctx, a.DB)
	if err != nil {
		logger.Error("error querying scheduled commands", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}

	sort.Slice(repeated, func(i, j int) bool {
		return repeated[i].R.CommandInfo.Name < repeated[j].R.CommandInfo.Name
	})

	sort.SliceStable(repeated, func(i, j int) bool {
		return repeated[i].Enabled && !repeated[j].Enabled
	})

	sort.Slice(scheduled, func(i, j int) bool {
		return scheduled[i].R.CommandInfo.Name < scheduled[j].R.CommandInfo.Name
	})

	sort.SliceStable(scheduled, func(i, j int) bool {
		return scheduled[i].Enabled && !scheduled[j].Enabled
	})

	templates.WritePageTemplate(w, &templates.ChannelScheduledPage{
		ChannelPage: templates.ChannelPage{
			Channel: channel,
		},
		Repeated:  repeated,
		Scheduled: scheduled,
	})
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	templates.WritePageTemplate(w, &templates.LoginPage{})
}

func httpError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
