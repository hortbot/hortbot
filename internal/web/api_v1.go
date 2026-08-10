package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/jackc/pgx/v5"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

func (a *App) routeAPIv1(r chi.Router) {
	r.Get("/vars/get/{varName}/{channel}", a.apiV1VarsGet)
}

func (a *App) apiV1VarsGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	varName := chi.URLParam(r, "varName")
	channelName := strings.ToLower(chi.URLParam(r, "channel"))

	variable, err := a.Queries.GetVariableByChannelName(ctx, dbsql.GetVariableByChannelNameParams{
		ChannelName: channelName,
		Name:        varName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			v1Error(w, http.StatusNotFound)
			return
		}
		ctxlog.Error(ctx, "error querying for variable", zap.Error(err))
		v1Error(w, http.StatusInternalServerError)
		return
	}

	v := &struct {
		Channel      string    `json:"channel"`
		Var          string    `json:"var"`
		Value        string    `json:"value"`
		LastModified time.Time `json:"lastModified"`
	}{
		Channel:      channelName,
		Var:          variable.Name,
		Value:        variable.Value,
		LastModified: variable.UpdatedAt.Time,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		ctxlog.Error(ctx, "failed to write response", zap.Error(err))
		return
	}
}

func v1Error(w http.ResponseWriter, code int) {
	v := &struct {
		Status string `json:"status"`
	}{
		Status: http.StatusText(code),
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
