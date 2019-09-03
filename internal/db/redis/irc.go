package redis

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
)

func (db *DB) SendMessageAllowed(ctx context.Context, botName string, limit int, window time.Duration) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SendMessageAllowed")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey("rate_limit", botName)
	return rateLimit(client, key, limit, window)
}
