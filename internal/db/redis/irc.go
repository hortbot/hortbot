package redis

import (
	"context"
	"time"

	"go.opencensus.io/trace"
)

const keyRateLimit = keyStr("rate_limit")

func (db *DB) SendMessageAllowed(ctx context.Context, botName, target string, limitSlow, limitFast int, window time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "SendMessageAllowed")
	defer span.End()

	client := db.client.WithContext(ctx)
	onlyFastKey := userStateKey(botName, target)

	key := buildKey(keyRateLimit.is(botName))
	return rateLimit(client, key, window, limitSlow, limitFast, onlyFastKey)
}
