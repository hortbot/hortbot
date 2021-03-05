package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hortbot/hortbot/internal/confimport"
	"github.com/hortbot/hortbot/internal/pkg/dbx"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"github.com/hortbot/hortbot/internal/web/templates"
	"github.com/tomwright/queryparam/v4"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

func (a *App) routeAdmin(r chi.Router) {
	r.Use(middleware.NoCache)
	r.Use(a.adminAuth)

	r.Route("/debug", a.routeDebug)

	r.Get("/import", a.adminImport)
	r.Post("/import", a.adminImportPost)
	r.Get("/export/{channel}", a.adminExport)
	r.Get("/stats", a.adminStats)
}

func (a *App) adminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(a.AdminAuth) == 0 {
			a.notAuthorized(w, r, false)
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			a.notAuthorized(w, r, true)
			return
		}

		expected := a.AdminAuth[user]
		if expected == "" || pass != expected {
			a.notAuthorized(w, r, true)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *App) adminExport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	channelName := chi.URLParam(r, "channel")

	config, err := confimport.ExportByName(ctx, a.DB, strings.ToLower(channelName))
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			ctxlog.Error(ctx, "error exporting channel", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	enc := json.NewEncoder(w)

	query := struct {
		Pretty bool `queryparam:"pretty"`
	}{}

	if err := queryparam.Parse(r.URL.Query(), &query); err != nil {
		a.httpError(w, r, http.StatusBadRequest)
		return
	}

	if query.Pretty {
		enc.SetIndent("", "    ")
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := enc.Encode(config); err != nil {
		ctxlog.Error(ctx, "error encoding exported config", zap.Error(err))
	}
}

func (a *App) adminImport(w http.ResponseWriter, r *http.Request) {
	page := &templates.AdminImportPage{
		BasePage: a.basePage(r),
	}
	templates.WritePageTemplate(w, page)
}

func (a *App) adminImportPost(w http.ResponseWriter, r *http.Request) {
	config := &confimport.Config{}

	if err := jsonx.DecodeSingle(r.Body, config); err != nil {
		http.Error(w, "decoding body: "+err.Error(), http.StatusBadRequest)
		return
	}

	err := dbx.Transact(r.Context(), a.DB,
		dbx.SetLocalLockTimeout(5*time.Second),
		func(ctx context.Context, tx *sql.Tx) error {
			return config.Insert(ctx, tx)
		},
	)
	if err != nil {
		http.Error(w, "inserting config: "+err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Fprintln(w, "Successfully inserted channel", config.Channel.ID)
}

func (a *App) adminStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := a.Redis.GetBuiltinUsageStats(ctx)
	if err != nil {
		ctxlog.Error(ctx, "error fetching builtin usage statistics", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type pair struct {
		key   string
		value int
	}

	pairs := make([]pair, 0, len(stats))
	for k, v := range stats {
		value, _ := strconv.Atoi(v)
		pairs = append(pairs, pair{key: k, value: value})
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].value == pairs[j].value {
			return pairs[i].key < pairs[j].key
		}
		return pairs[i].value > pairs[j].value
	})

	fmt.Fprintln(w, "Builtin command usage:")

	for _, p := range pairs {
		fmt.Fprintf(w, "%s = %d\n", p.key, p.value)
	}
}
