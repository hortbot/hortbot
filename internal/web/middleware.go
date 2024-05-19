package web

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

type contextKey int

const (
	channelKey contextKey = iota
)

func (a *App) channelMiddleware(urlParam string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			name := chi.URLParamFromCtx(ctx, urlParam)

			channel, err := models.Channels(models.ChannelWhere.Name.EQ(strings.ToLower(name))).One(ctx, a.DB)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					a.httpError(w, r, http.StatusNotFound)
				} else {
					ctxlog.Error(ctx, "error querying channel", zap.Error(err))
					a.httpError(w, r, http.StatusInternalServerError)
				}
				return
			}

			ctx = context.WithValue(ctx, channelKey, channel)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

func getChannel(ctx context.Context) *models.Channel {
	return ctx.Value(channelKey).(*models.Channel)
}
