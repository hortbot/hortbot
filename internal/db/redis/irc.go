package redis

import (
	"context"
	"time"

	"go.opencensus.io/trace"
)

const keyRateLimit = keyStr("rate_limit")

// SendMessageAllowed limits the rate of sent message, based on the user state
// of the bot in the targeted IRC channel.
func (db *DB) SendMessageAllowed(ctx context.Context, botName, target string, limitSlow, limitFast int, window time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "SendMessageAllowed")
	defer span.End()

	onlyFastKey := userStateKey(botName, target)

	key := buildKey(keyRateLimit.is(botName))
	return rateLimit(ctx, db.client, key, window, limitSlow, limitFast, onlyFastKey)
}
