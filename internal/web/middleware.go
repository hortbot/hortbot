package web

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
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
				if err == sql.ErrNoRows {
					httpError(w, http.StatusNotFound)
				} else {
					ctxlog.Error(ctx, "error querying channel", zap.Error(err))
					httpError(w, http.StatusInternalServerError)
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
