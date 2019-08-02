package web

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/hortbot/hortbot/internal/web/mid"
	"go.uber.org/zap"
)

const rdbKey = "auth_state"

type App struct {
	Addr   string
	RealIP bool

	RDB    *rdb.DB
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
	r.Use(mid.Recoverer)

	r.Get("/auth/twitch", a.authTwitch)
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
	state := uuid.Must(uuid.NewV4()).String()

	if err := a.RDB.Mark(60*60, rdbKey, state); err != nil {
		httpError(w, http.StatusInternalServerError)
		return
	}

	url := a.Twitch.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (a *App) authTwitchCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	state := r.FormValue("state")
	if state == "" {
		httpError(w, http.StatusBadRequest)
		return
	}

	ok, err := a.RDB.CheckAndDelete(rdbKey, state)
	if err != nil {
		httpError(w, http.StatusInternalServerError)
		return
	}

	if !ok {
		httpError(w, http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")

	tok, err := a.Twitch.Exchange(ctx, code)
	if err != nil {
		httpError(w, http.StatusBadRequest)
		return
	}

	id, newToken, err := a.Twitch.GetIDForToken(ctx, tok)
	if err != nil {
		httpError(w, http.StatusInternalServerError)
		return
	}
	if newToken != nil {
		tok = newToken
	}

	tt := modelsx.TokenToModel(id, tok)

	if err := modelsx.UpsertToken(ctx, a.DB, tt); err != nil {
		httpError(w, http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Success for user %d.\n", id)
}

func httpError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}
