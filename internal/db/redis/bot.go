package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/opentracing/opentracing-go"
)

const (
	linkPermit       = "link_permit"
	confirm          = "confirm"
	repeatedCommand  = "repeated_command"
	scheduledCommand = "scheduled_command"
	messageCount     = "message_count"
	autoreply        = "autoreply"
	filterWarned     = "filter_warning"
	raffle           = "raffle"
	cooldown         = "cooldown"
	userState        = "user_state"
)

func (db *DB) LinkPermit(ctx context.Context, channel, user string, expiry time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "LinkPermit")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(channel, linkPermit, user)
	return mark(client, key, expiry)
}

func (db *DB) HasLinkPermit(ctx context.Context, channel, user string) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "HasLinkPermit")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(channel, linkPermit, user)
	return checkAndDelete(client, key)
}

func (db *DB) Confirm(ctx context.Context, channel, user, key string, expiry time.Duration) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Confirm")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	rkey := buildKey(channel, confirm, user, key)
	return markOrDelete(client, rkey, expiry)
}

func (db *DB) RepeatAllowed(ctx context.Context, channel string, id int64, expiry time.Duration) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RepeatAllowed")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(channel, repeatedCommand, strconv.FormatInt(id, 10))
	seen, err := checkAndMark(client, key, expiry)
	return !seen, err
}

func (db *DB) ScheduledAllowed(ctx context.Context, channel string, id int64, expiry time.Duration) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ScheduledAllowed")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(channel, scheduledCommand, strconv.FormatInt(id, 10))
	seen, err := checkAndMark(client, key, expiry)
	return !seen, err
}

func (db *DB) MessageCount(ctx context.Context, channel string) (int64, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "MessageCount")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(channel, messageCount)
	v, err := client.Get(key).Int64()
	if err == redis.Nil {
		err = nil
	}
	return v, err
}

func (db *DB) IncrementMessageCount(ctx context.Context, channel string) (int64, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "IncrementMessageCount")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(channel, messageCount)
	return client.Incr(key).Result()
}

func (db *DB) AutoreplyAllowed(ctx context.Context, channel string, id int64, expiry time.Duration) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "AutoreplyAllowed")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(channel, autoreply, strconv.FormatInt(id, 10))
	seen, err := checkAndMark(client, key, expiry)
	return !seen, err
}

func (db *DB) FilterWarned(ctx context.Context, channel, user, filter string, expiry time.Duration) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "FilterWarned")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(channel, filterWarned, user, filter)
	return checkAndRefresh(client, key, expiry)
}

func (db *DB) RaffleAdd(ctx context.Context, channel, user string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RaffleAdd")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(channel, raffle)
	return setAdd(client, key, user)
}

func (db *DB) RaffleReset(ctx context.Context, channel string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RaffleReset")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(channel, raffle)
	return setClear(client, key)
}

func (db *DB) RaffleWinner(ctx context.Context, channel string) (string, bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RaffleWinner")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(channel, raffle)
	return setPop(client, key)
}

func (db *DB) RaffleCount(ctx context.Context, channel string) (int64, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RaffleCount")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(channel, raffle)
	return setLen(client, key)
}

func (db *DB) MarkCooldown(ctx context.Context, channel, key string, expiry time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "MarkCooldown")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	rkey := buildKey(channel, cooldown, key)
	return mark(client, rkey, expiry)
}

func (db *DB) CheckAndMarkCooldown(ctx context.Context, channel, key string, expiry time.Duration) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CheckAndMarkCooldown")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	rkey := buildKey(channel, cooldown, key)
	return checkAndMark(client, rkey, expiry)
}

func (db *DB) SetUserState(ctx context.Context, botName, ircChannel string, fast bool, expiry time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SetUserState")
	defer span.Finish()

	v := "0"
	if fast {
		v = "1"
	}

	client := db.client.WithContext(ctx)
	key := buildKey(botName, userState, ircChannel)
	return client.Set(key, v, expiry).Err()
}

func (db *DB) GetUserState(ctx context.Context, botName, ircChannel string) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetUserState")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(botName, userState, ircChannel)
	r, err := client.Get(key).Result()
	if err == redis.Nil {
		err = nil
	}
	return r == "1", err
}
