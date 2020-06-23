package redis

import (
	"context"
	"strconv"
	"time"

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

// LinkPermit checks marks a user as having a single link permit.
func (db *DB) LinkPermit(ctx context.Context, channel, user string, expiry time.Duration) error {
	ctx, span := trace.StartSpan(ctx, "LinkPermit")
	defer span.End()

	key := linkPermitKey(channel, user)
	return mark(ctx, db.client, key, expiry)
}

// HasLinkPermit returns true if a user has a link permit, invalidating the permit.
func (db *DB) HasLinkPermit(ctx context.Context, channel, user string) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "HasLinkPermit")
	defer span.End()

	key := linkPermitKey(channel, user)
	return checkAndDelete(ctx, db.client, key)
}

// Confirm checks that a user has confirmed something (keyed on "key").
func (db *DB) Confirm(ctx context.Context, channel, user, key string, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "Confirm")
	defer span.End()

	rkey := buildKey(
		keyChannel.is(channel),
		keyConfirm.is(user),
		keyKey.is(key),
	)

	return markOrDelete(ctx, db.client, rkey, expiry)
}

// RepeatAllowed returns true if a certain repeat is allowed to run,
// preventing future repeats of this ID from running until expiring.
func (db *DB) RepeatAllowed(ctx context.Context, channel string, id int64, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "RepeatAllowed")
	defer span.End()

	key := buildKey(
		keyChannel.is(channel),
		keyRepeatedCommand.is(strconv.FormatInt(id, 10)),
	)
	seen, err := checkAndMark(ctx, db.client, key, expiry)
	return !seen, err
}

// ScheduledAllowed returns true if a certain schedule is allowed to run,
// preventing future schedules of this ID from running until expiring.
func (db *DB) ScheduledAllowed(ctx context.Context, channel string, id int64, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "ScheduledAllowed")
	defer span.End()

	key := buildKey(
		keyChannel.is(channel),
		keyScheduledCommand.is(strconv.FormatInt(id, 10)),
	)
	seen, err := checkAndMark(ctx, db.client, key, expiry)
	return !seen, err
}

// AutoreplyAllowed returns true if a certain autoreply is allowed to run,
// preventing future runs of this ID from running until expiring.
func (db *DB) AutoreplyAllowed(ctx context.Context, channel string, id int64, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "AutoreplyAllowed")
	defer span.End()

	key := buildKey(
		keyChannel.is(channel),
		keyAutoreply.is(strconv.FormatInt(id, 10)),
	)
	seen, err := checkAndMark(ctx, db.client, key, expiry)
	return !seen, err
}

// FilterWarned chceks if a user has already been warned for a given filter.
func (db *DB) FilterWarned(ctx context.Context, channel, user, filter string, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "FilterWarned")
	defer span.End()

	key := buildKey(
		keyChannel.is(channel),
		keyFilterWarned.is(user),
		keyFilter.is(filter),
	)
	return checkAndRefresh(ctx, db.client, key, expiry)
}

func raffleKey(channel string) string {
	return buildKey(
		keyChannel.is(channel),
		keyRaffle.is(""),
	)
}

// RaffleAdd adds a user to the current raffle. Duplicate entries are not allowed.
func (db *DB) RaffleAdd(ctx context.Context, channel, user string) error {
	ctx, span := trace.StartSpan(ctx, "RaffleAdd")
	defer span.End()

	key := raffleKey(channel)
	return setAdd(ctx, db.client, key, user)
}

// RaffleReset removes all entires from the current raffle.
func (db *DB) RaffleReset(ctx context.Context, channel string) error {
	ctx, span := trace.StartSpan(ctx, "RaffleReset")
	defer span.End()

	key := raffleKey(channel)
	return setClear(ctx, db.client, key)
}

// RaffleWinner removes a user from the raffle entries and returns them.
func (db *DB) RaffleWinner(ctx context.Context, channel string) (string, bool, error) {
	ctx, span := trace.StartSpan(ctx, "RaffleWinner")
	defer span.End()

	key := raffleKey(channel)
	return setPop(ctx, db.client, key)
}

// RaffleCount returns the number of entries in the current raffle.
func (db *DB) RaffleCount(ctx context.Context, channel string) (int64, error) {
	ctx, span := trace.StartSpan(ctx, "RaffleCount")
	defer span.End()

	key := raffleKey(channel)
	return setLen(ctx, db.client, key)
}

func cooldownKey(channel string, key string) string {
	return buildKey(
		keyChannel.is(channel),
		keyCooldown.is(key),
	)
}

// MarkCooldown marks that a command is on cooldown.
func (db *DB) MarkCooldown(ctx context.Context, channel, key string, expiry time.Duration) error {
	ctx, span := trace.StartSpan(ctx, "MarkCooldown")
	defer span.End()

	rkey := cooldownKey(channel, key)
	return mark(ctx, db.client, rkey, expiry)
}

// CheckAndMarkCooldown checks that a command is on cooldown, and marks it.
func (db *DB) CheckAndMarkCooldown(ctx context.Context, channel, key string, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "CheckAndMarkCooldown")
	defer span.End()

	rkey := cooldownKey(channel, key)
	return checkAndMark(ctx, db.client, rkey, expiry)
}

func userStateKey(botName, target string) string {
	return buildKey(
		keyBotName.is(botName),
		keyUserState.is(target),
	)
}

// SetUserState sets the current bot user state as either fast or slow for a
// given IRC channel.
func (db *DB) SetUserState(ctx context.Context, botName, ircChannel string, fast bool, expiry time.Duration) error {
	ctx, span := trace.StartSpan(ctx, "SetUserState")
	defer span.End()

	v := "0"
	if fast {
		v = "1"
	}

	key := userStateKey(botName, ircChannel)
	return db.client.Set(ctx, key, v, expiry).Err()
}

// GetUserState returns the current user state of the bot in the given IRC channel.
func (db *DB) GetUserState(ctx context.Context, botName, ircChannel string) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "GetUserState")
	defer span.End()

	key := userStateKey(botName, ircChannel)
	r, err := db.client.Get(ctx, key).Result()
	return r == "1", ignoreRedisNil(err)
}
