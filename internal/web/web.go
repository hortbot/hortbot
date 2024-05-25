// Package web implements the HortBot web server.
package web

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gofrs/uuid"
	"github.com/gorilla/sessions"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/must"
	"github.com/hortbot/hortbot/internal/web/mid"
	"github.com/hortbot/hortbot/internal/web/templates"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tomwright/queryparam/v4"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

//go:embed static
var static embed.FS

var staticDir = must.Must(fs.Sub(static, "static"))

// App is the HortBot webapp.
type App struct {
	Addr       string
	RealIP     bool
	SessionKey []byte
	AdminAuth  map[string]string

	Brand    string
	BrandMap map[string]string

	Debug bool

	Redis  *redis.DB
	DB     *sql.DB
	Twitch twitch.API

	store *sessions.CookieStore
}

// Run runs the webapp until the context is canceled.
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

	r.Use(func(next http.Handler) http.Handler {
		return promhttp.InstrumentHandlerCounter(metricRequest, next)
	})

	r.Use(mid.RequestLogger)
	r.Use(mid.Tracer)
	r.Use(mid.Recoverer)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		a.httpError(w, r, http.StatusNotFound)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.RedirectSlashes)

		r.Get("/", a.index)
		r.Get("/about", a.about)
		r.Get("/help", a.help)
		r.Get("/docs", a.docs)
		r.Get("/channels", a.channels)

		const paramChannel = "channel"
		r.Route("/c/{"+paramChannel+"}", func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					p := r.URL.Path
					lp := strings.ToLower(p)

					if p == lp {
						next.ServeHTTP(w, r)
						return
					}

					if r.URL.RawQuery != "" {
						lp += "?" + r.URL.RawQuery
					}

					http.Redirect(w, r, lp, http.StatusMovedPermanently)
				})
			})

			r.Use(a.channelMiddleware(paramChannel))
			r.Get("/", a.channel)
			r.Get("/commands", a.channelCommands)
			r.Get("/quotes", a.channelQuotes)
			r.Get("/autoreplies", a.channelAutoreplies)
			r.Get("/lists", a.channelLists)
			r.Get("/regulars", a.channelRegulars)
			r.Get("/chatrules", a.channelChatRules)
			r.Get("/scheduled", a.channelScheduled)
			r.Get("/variables", a.channelVariables)
			r.Get("/highlights", a.channelHighlights)
		})

		r.Route("/api/v1", a.routeAPIv1)
		r.Get("/showvar.php", a.showVar)

		r.Get("/login", a.login)
		r.Get("/logout", a.logout)
		r.Get("/auth/twitch", a.authTwitchNormal)
		r.Get("/auth/twitch/bot", a.authTwitchBot)
		r.Get("/auth/twitch/callback", a.authTwitchCallback)

		if a.Debug {
			r.Route("/debug", a.routeDebug)
		}

		r.Route("/admin", a.routeAdmin)
	})

	r.Handle("/static/*", http.StripPrefix("/static", http.FileServer(http.FS(staticDir))))
	r.Handle("/favicon.ico", http.RedirectHandler("/static/icons/favicon.ico", http.StatusFound))

	srv := http.Server{
		Addr:              a.Addr,
		Handler:           r,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			ctxlog.Error(ctx, "error shutting down server", zap.Error(err))
		}
	}()

	ctxlog.Info(ctx, "web server listening", zap.String("addr", srv.Addr))

	return srv.ListenAndServe()
}

func (a *App) getBrand(r *http.Request) string {
	if a.BrandMap == nil {
		return a.Brand
	}

	host := r.Host
	if addrHost, _, err := net.SplitHostPort(host); err == nil {
		host = addrHost
	}
	host = normalizeHost(host)

	if host != "" {
		if brand := a.BrandMap[host]; brand != "" {
			return brand
		}
	}

	return a.Brand
}

func (a *App) basePage(r *http.Request) templates.BasePage {
	return templates.BasePage{
		Brand: a.getBrand(r),
		User:  a.getSession(r).getUsername(),
	}
}

type authState struct {
	Host     string
	Bot      bool
	Redirect string
}

const authTimeout = time.Hour

func (a *App) authTwitch(w http.ResponseWriter, r *http.Request, bot bool) {
	ctx := r.Context()

	state := uuid.Must(uuid.NewV4()).String()

	stateVal := &authState{
		Host: r.Host, // Not normalized; needed for redirects.
		Bot:  bot,
	}

	query := struct {
		Redirect string `queryparam:"redirect"`
	}{}

	if err := queryparam.Parse(r.URL.Query(), &query); err != nil {
		a.httpError(w, r, http.StatusBadRequest)
		return
	}

	stateVal.Redirect = query.Redirect

	if err := a.Redis.SetAuthState(ctx, state, stateVal, authTimeout); err != nil {
		ctxlog.Error(ctx, "error setting auth state", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	scopes := twitch.UserScopes
	if bot {
		scopes = twitch.BotScopes
	}

	url := a.Twitch.AuthCodeURL(state, scopes)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (a *App) authTwitchNormal(w http.ResponseWriter, r *http.Request) {
	a.authTwitch(w, r, false)
}

func (a *App) authTwitchBot(w http.ResponseWriter, r *http.Request) {
	a.authTwitch(w, r, true)
}

func (a *App) authTwitchCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	state := r.FormValue("state")
	if state == "" {
		a.httpError(w, r, http.StatusBadRequest)
		return
	}

	var stateVal authState

	ok, err := a.Redis.GetAuthState(ctx, state, &stateVal)
	if err != nil {
		ctxlog.Error(ctx, "error checking auth state", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	if !ok {
		a.httpError(w, r, http.StatusBadRequest)
		return
	}

	if normalizeHost(stateVal.Host) != normalizeHost(r.Host) {
		// This came to the wrong host. Put the state back and redirect.
		if err := a.Redis.SetAuthState(ctx, state, &stateVal, authTimeout); err != nil {
			ctxlog.Error(ctx, "error setting auth state", zap.Error(err))
			a.httpError(w, r, http.StatusInternalServerError)
			return
		}

		u := *r.URL
		u.Host = stateVal.Host
		templates.WriteMetaRedirect(w, u.String())
		return
	}

	tok, err := a.Twitch.Exchange(ctx, r.FormValue("code"))
	if err != nil {
		ctxlog.Error(ctx, "error exchanging code", zap.Error(err))
		a.httpError(w, r, http.StatusBadRequest)
		return
	}

	user, newToken, err := a.Twitch.GetUserByToken(ctx, tok)
	if err != nil {
		ctxlog.Error(ctx, "error getting user for token", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}
	if newToken != nil {
		tok = newToken
	}

	var botName null.String
	if stateVal.Bot {
		botName = null.StringFrom(user.Name)
	}

	tt := modelsx.TokenToModel(tok, int64(user.ID), botName, strings.Fields(r.FormValue("scope")))

	if err := modelsx.UpsertToken(ctx, a.DB, tt); err != nil {
		ctxlog.Error(ctx, "error upserting token", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	session := a.getSession(r)
	session.clearValues()
	session.setTwitchID(int64(user.ID))
	session.setUsername(user.Name)

	if err := session.save(w, r); err != nil {
		ctxlog.Error(ctx, "error saving session", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	if stateVal.Redirect != "" {
		http.Redirect(w, r, stateVal.Redirect, http.StatusSeeOther)
		return
	}

	page := &templates.LoginSuccessPage{
		BasePage: a.basePage(r),
		Name:     user.Name,
		ID:       int64(user.ID),
		Bot:      stateVal.Bot,
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	channelCount, botCount, err := modelsx.CountActiveChannels(ctx, a.DB)
	if err != nil {
		ctxlog.Error(ctx, "error counting channels", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	page := &templates.IndexPage{
		BasePage:     a.basePage(r),
		ChannelCount: int64(channelCount),
		BotCount:     int64(botCount),
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) about(w http.ResponseWriter, r *http.Request) {
	page := &templates.AboutPage{
		BasePage: a.basePage(r),
	}
	templates.WritePageTemplate(w, page)
}

func (a *App) help(w http.ResponseWriter, r *http.Request) {
	page := &templates.HelpPage{
		BasePage: a.basePage(r),
	}
	templates.WritePageTemplate(w, page)
}

func (a *App) docs(w http.ResponseWriter, r *http.Request) {
	page := &templates.DocsPage{
		BasePage: a.basePage(r),
	}
	templates.WritePageTemplate(w, page)
}

func (a *App) channels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	channels, err := models.Channels(
		qm.Select(models.ChannelColumns.Name, models.ChannelColumns.DisplayName),
		models.ChannelWhere.Active.EQ(true),
		qm.OrderBy(models.ChannelColumns.Name),
	).All(ctx, a.DB)
	if err != nil {
		ctxlog.Error(ctx, "error querying channels", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	page := &templates.ChannelsPage{
		BasePage: a.basePage(r),
		Channels: channels,
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) channelPage(r *http.Request, channel *models.Channel) templates.ChannelPage {
	return templates.ChannelPage{
		BasePage: a.basePage(r),
		Channel:  channel,
	}
}

func (a *App) channel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	page := &templates.ChannelPage{
		BasePage: a.basePage(r),
		Channel:  channel,
	}
	templates.WritePageTemplate(w, page)
}

func (a *App) channelCommands(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	commands, err := channel.CustomCommands(qm.Load(models.CustomCommandRels.CommandInfo)).All(ctx, a.DB)
	if err != nil {
		ctxlog.Error(ctx, "error querying custom commands", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	sort.Slice(commands, func(i, j int) bool {
		return commands[i].R.CommandInfo.Name < commands[j].R.CommandInfo.Name
	})

	page := &templates.ChannelCommandsPage{
		ChannelPage: a.channelPage(r, channel),
		Commands:    commands,
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) channelQuotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	quotes, err := channel.Quotes(qm.OrderBy(models.QuoteColumns.Num)).All(ctx, a.DB)
	if err != nil {
		ctxlog.Error(ctx, "error querying quotes", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	page := &templates.ChannelQuotesPage{
		ChannelPage: a.channelPage(r, channel),
		Quotes:      quotes,
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) channelAutoreplies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	autoreplies, err := channel.Autoreplies(qm.OrderBy(models.AutoreplyColumns.Num)).All(ctx, a.DB)
	if err != nil {
		ctxlog.Error(ctx, "error querying autoreplies", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	page := &templates.ChannelAutorepliesPage{
		ChannelPage: a.channelPage(r, channel),
		Autoreplies: autoreplies,
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) channelLists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	lists, err := channel.CommandLists(qm.Load(models.CommandListRels.CommandInfo)).All(ctx, a.DB)
	if err != nil {
		ctxlog.Error(ctx, "error querying command lists", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	sort.Slice(lists, func(i, j int) bool {
		return lists[i].R.CommandInfo.Name < lists[j].R.CommandInfo.Name
	})

	page := &templates.ChannelListsPage{
		ChannelPage: a.channelPage(r, channel),
		Lists:       lists,
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) channelRegulars(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	page := &templates.ChannelRegularsPage{
		ChannelPage: a.channelPage(r, channel),
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) channelChatRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	page := &templates.ChannelRulesPage{
		ChannelPage: a.channelPage(r, channel),
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) channelScheduled(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	repeated, err := channel.RepeatedCommands(qm.Load(models.RepeatedCommandRels.CommandInfo)).All(ctx, a.DB)
	if err != nil {
		ctxlog.Error(ctx, "error querying repeated commands", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	scheduled, err := channel.ScheduledCommands(qm.Load(models.ScheduledCommandRels.CommandInfo)).All(ctx, a.DB)
	if err != nil {
		ctxlog.Error(ctx, "error querying scheduled commands", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	sort.Slice(repeated, func(i, j int) bool {
		if repeated[i].Enabled != repeated[j].Enabled {
			return repeated[i].Enabled
		}

		return repeated[i].R.CommandInfo.Name < repeated[j].R.CommandInfo.Name
	})

	sort.Slice(scheduled, func(i, j int) bool {
		if scheduled[i].Enabled != scheduled[j].Enabled {
			return scheduled[i].Enabled
		}

		return scheduled[i].R.CommandInfo.Name < scheduled[j].R.CommandInfo.Name
	})

	page := &templates.ChannelScheduledPage{
		ChannelPage: a.channelPage(r, channel),
		Repeated:    repeated,
		Scheduled:   scheduled,
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) channelVariables(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	variables, err := channel.Variables(qm.OrderBy(models.VariableColumns.Name)).All(ctx, a.DB)
	if err != nil {
		ctxlog.Error(ctx, "error querying variables", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	page := &templates.ChannelVariablesPage{
		ChannelPage: a.channelPage(r, channel),
		Variables:   variables,
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) channelHighlights(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channel := getChannel(ctx)

	limit := time.Now().Add(-30 * 24 * time.Hour)

	highlights, err := channel.Highlights(
		models.HighlightWhere.HighlightedAt.GT(limit),
		qm.OrderBy(models.HighlightColumns.HighlightedAt),
	).All(ctx, a.DB)
	if err != nil {
		ctxlog.Error(ctx, "error querying highlights", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	page := &templates.ChannelHighlightsPage{
		ChannelPage: a.channelPage(r, channel),
		Highlights:  highlights,
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	page := &templates.LoginPage{
		BasePage: a.basePage(r),
	}

	templates.WritePageTemplate(w, page)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := a.clearSession(w, r); err != nil {
		ctxlog.Error(ctx, "error clearing session", zap.Error(err))
		a.httpError(w, r, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) showVar(w http.ResponseWriter, r *http.Request) {
	query := struct {
		Channel    string `queryparam:"channel"`
		Var        string `queryparam:"var"`
		Refresh    int    `queryparam:"refresh"`
		Themes     string `queryparam:"themes"`
		ValueFont  string `queryparam:"valueFont"`
		ValueColor string `queryparam:"valueColor"`
		LabelFont  string `queryparam:"labelFont"`
		LabelColor string `queryparam:"labelColor"`
		Label      string `queryparam:"label"`
	}{}

	if err := queryparam.Parse(r.URL.Query(), &query); err != nil {
		a.httpError(w, r, http.StatusBadRequest)
		return
	}

	if query.Refresh <= 0 {
		query.Refresh = 5000
	}

	themes := make(map[string]bool)
	for _, theme := range strings.Fields(query.Themes) {
		themes[theme] = true
	}

	p := &templates.ShowVarPage{
		Channel:    query.Channel,
		Var:        query.Var,
		Refresh:    query.Refresh,
		ThemesStr:  query.Themes,
		Themes:     themes,
		ValueFont:  query.ValueFont,
		ValueColor: query.ValueColor,
		LabelFont:  query.LabelFont,
		LabelColor: query.LabelColor,
		Label:      query.Label,
	}

	p.WriteRender(w)
}
