package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis/v7"
	"go.opencensus.io/trace"
)

const (
	keyKey              = keyStr("key")
	keyChannel          = keyStr("channel")
	keyLinkPermit       = keyStr("link_permit")
	keyConfirm          = keyStr("confirm")
	keyRepeatedCommand  = keyStr("repeated_command")
	keyScheduledCommand = keyStr("scheduled_command")
	keyAutoreply        = keyStr("autoreply")
	keyFilterWarned     = keyStr("filter_warning")
	keyRaffle           = keyStr("raffle")
	keyCooldown         = keyStr("cooldown")
	keyUserState        = keyStr("user_state")
	keyFilter           = keyStr("filter")
	keyBotName          = keyStr("bot_name")
)

func linkPermitKey(channel string, user string) string {
	return buildKey(
		keyChannel.is(channel),
		keyLinkPermit.is(user),
	)
}

func (db *DB) LinkPermit(ctx context.Context, channel, user string, expiry time.Duration) error {
	ctx, span := trace.StartSpan(ctx, "LinkPermit")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := linkPermitKey(channel, user)
	return mark(client, key, expiry)
}

func (db *DB) HasLinkPermit(ctx context.Context, channel, user string) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "HasLinkPermit")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := linkPermitKey(channel, user)
	return checkAndDelete(client, key)
}

func (db *DB) Confirm(ctx context.Context, channel, user, key string, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "Confirm")
	defer span.End()

	client := db.client.WithContext(ctx)
	rkey := buildKey(
		keyChannel.is(channel),
		keyConfirm.is(user),
		keyKey.is(key),
	)

	return markOrDelete(client, rkey, expiry)
}

func (db *DB) RepeatAllowed(ctx context.Context, channel string, id int64, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "RepeatAllowed")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(
		keyChannel.is(channel),
		keyRepeatedCommand.is(strconv.FormatInt(id, 10)),
	)
	seen, err := checkAndMark(client, key, expiry)
	return !seen, err
}

func (db *DB) ScheduledAllowed(ctx context.Context, channel string, id int64, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "ScheduledAllowed")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(
		keyChannel.is(channel),
		keyScheduledCommand.is(strconv.FormatInt(id, 10)),
	)
	seen, err := checkAndMark(client, key, expiry)
	return !seen, err
}

func (db *DB) AutoreplyAllowed(ctx context.Context, channel string, id int64, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "AutoreplyAllowed")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(
		keyChannel.is(channel),
		keyAutoreply.is(strconv.FormatInt(id, 10)),
	)
	seen, err := checkAndMark(client, key, expiry)
	return !seen, err
}

func (db *DB) FilterWarned(ctx context.Context, channel, user, filter string, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "FilterWarned")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(
		keyChannel.is(channel),
		keyFilterWarned.is(user),
		keyFilter.is(filter),
	)
	return checkAndRefresh(client, key, expiry)
}

func raffleKey(channel string) string {
	return buildKey(
		keyChannel.is(channel),
		keyRaffle.is(""),
	)
}

func (db *DB) RaffleAdd(ctx context.Context, channel, user string) error {
	ctx, span := trace.StartSpan(ctx, "RaffleAdd")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := raffleKey(channel)
	return setAdd(client, key, user)
}

func (db *DB) RaffleReset(ctx context.Context, channel string) error {
	ctx, span := trace.StartSpan(ctx, "RaffleReset")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := raffleKey(channel)
	return setClear(client, key)
}

func (db *DB) RaffleWinner(ctx context.Context, channel string) (string, bool, error) {
	ctx, span := trace.StartSpan(ctx, "RaffleWinner")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := raffleKey(channel)
	return setPop(client, key)
}

func (db *DB) RaffleCount(ctx context.Context, channel string) (int64, error) {
	ctx, span := trace.StartSpan(ctx, "RaffleCount")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := raffleKey(channel)
	return setLen(client, key)
}

func cooldownKey(channel string, key string) string {
	return buildKey(
		keyChannel.is(channel),
		keyCooldown.is(key),
	)
}

func (db *DB) MarkCooldown(ctx context.Context, channel, key string, expiry time.Duration) error {
	ctx, span := trace.StartSpan(ctx, "MarkCooldown")
	defer span.End()

	client := db.client.WithContext(ctx)
	rkey := cooldownKey(channel, key)
	return mark(client, rkey, expiry)
}

func (db *DB) CheckAndMarkCooldown(ctx context.Context, channel, key string, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "CheckAndMarkCooldown")
	defer span.End()

	client := db.client.WithContext(ctx)
	rkey := cooldownKey(channel, key)
	return checkAndMark(client, rkey, expiry)
}

func userStateKey(botName, target string) string {
	return buildKey(
		keyBotName.is(botName),
		keyUserState.is(target),
	)
}

func (db *DB) SetUserState(ctx context.Context, botName, ircChannel string, fast bool, expiry time.Duration) error {
	ctx, span := trace.StartSpan(ctx, "SetUserState")
	defer span.End()

	v := "0"
	if fast {
		v = "1"
	}

	client := db.client.WithContext(ctx)
	key := userStateKey(botName, ircChannel)
	return client.Set(key, v, expiry).Err()
}

func (db *DB) GetUserState(ctx context.Context, botName, ircChannel string) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "GetUserState")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := userStateKey(botName, ircChannel)
	r, err := client.Get(key).Result()
	if err == redis.Nil {
		err = nil
	}
	return r == "1", err
}
