package bot

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/rs/xid"
	"go.uber.org/zap"
)

type correlationKey struct{}

func withCorrelation(ctx context.Context) context.Context {
	cid := xid.New().String()
	ctx = context.WithValue(ctx, correlationKey{}, cid)
	ctx = ctxlog.With(ctx, zap.String("cid", cid))
	return ctx
}
