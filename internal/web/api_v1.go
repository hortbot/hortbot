package web

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/v4/queries"
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

	variable := &models.Variable{}

	err := queries.Raw("SELECT variables.* FROM variables JOIN channels ON variables.channel_id = channels.id WHERE variables.name = $1 AND channels.name = $2", varName, channelName).Bind(ctx, a.DB, variable)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
		LastModified: variable.UpdatedAt,
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
