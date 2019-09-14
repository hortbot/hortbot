package web

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/web/mid"
	"github.com/hortbot/hortbot/internal/web/templates"
	"github.com/volatiletech/null"
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
	Addr   string
	RealIP bool

	Redis  *redis.DB
	DB     *sql.DB
	Twitch *twitch.Twitch
}

func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

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
		r.Get("/commands", a.channelCommands)
		r.Get("/quotes", a.channelQuotes)
	})

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

	if botName == "" {
		fmt.Fprintf(w, "Success for user %s (%d).\n", user.Name, user.ID)
	} else {
		fmt.Fprintf(w, "Success for user %s (%d) as bot.\n", user.Name, user.ID)
	}
}

func (a *App) index(w http.ResponseWriter, r *http.Request) {
	templates.WritePageTemplate(w, &templates.IndexPage{})
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

func httpError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
