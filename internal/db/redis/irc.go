package redis

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
)

func (db *DB) SendMessageAllowed(ctx context.Context, botName, target string, limitSlow, limitFast int, window time.Duration) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SendMessageAllowed")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	onlyFastKey := userStateKey(botName, target)

	key := buildKey("rate_limit", botName)
	return rateLimit(client, key, window, limitSlow, limitFast, onlyFastKey)
}
