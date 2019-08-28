package bot

import (
	"context"
	"errors"
	"strconv"

	"github.com/opentracing/opentracing-go"
)

func (s *session) LinkPermit(ctx context.Context, user string, seconds int) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "LinkPermit")
	defer span.Finish()
	return s.Deps.RDB.Mark(seconds, s.RoomIDStr, "link_permit", user)
}

func (s *session) HasLinkPermit(ctx context.Context, user string) (permitted bool, err error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "HasLinkPermit")
	defer span.Finish()
	return s.Deps.RDB.CheckAndDelete(s.RoomIDStr, "link_permit", user)
}

func (s *session) Confirm(ctx context.Context, user string, key string, seconds int) (confirmed bool, err error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "Confirm")
	defer span.Finish()
	return s.Deps.RDB.MarkOrDelete(seconds, s.RoomIDStr, "confirm", user, key)
}

func (s *session) RepeatAllowed(ctx context.Context, id int64, seconds int) (bool, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "RepeatAllowed")
	defer span.Finish()
	seen, err := s.Deps.RDB.CheckAndMark(seconds, s.RoomIDStr, "repeated_command", strconv.FormatInt(id, 10))
	return !seen, err
}

func (s *session) ScheduledAllowed(ctx context.Context, id int64, seconds int) (bool, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "ScheduledAllowed")
	defer span.Finish()
	seen, err := s.Deps.RDB.CheckAndMark(seconds, s.RoomIDStr, "scheduled_command", strconv.FormatInt(id, 10))
	return !seen, err
}

func (s *session) MessageCount(ctx context.Context) (int64, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "MessageCount")
	defer span.Finish()
	return s.Deps.RDB.GetInt64(s.RoomIDStr, "message_count")
}

func (s *session) IncrementMessageCount(ctx context.Context) (int64, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "IncrementMessageCount")
	defer span.Finish()
	return s.Deps.RDB.Increment(s.RoomIDStr, "message_count")
}

func (s *session) AutoreplyAllowed(ctx context.Context, id int64, seconds int) (bool, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "AutoreplyAllowed")
	defer span.Finish()
	seen, err := s.Deps.RDB.CheckAndMark(seconds, s.RoomIDStr, "autoreply", strconv.FormatInt(id, 10))
	return !seen, err
}

func (s *session) FilterWarned(ctx context.Context, user string, filter string) (bool, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "FilterWarned")
	defer span.Finish()
	return s.Deps.RDB.CheckAndRefresh(3600, s.RoomIDStr, "filter_warning", filter, user)
}

func (s *session) RaffleAdd(ctx context.Context, user string) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "RaffleAdd")
	defer span.Finish()
	return s.Deps.RDB.SetAdd(user, s.RoomIDStr, "raffle")
}

func (s *session) RaffleReset(ctx context.Context) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "RaffleReset")
	defer span.Finish()
	return s.Deps.RDB.SetClear(s.RoomIDStr, "raffle")
}

func (s *session) RaffleWinner(ctx context.Context) (string, bool, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "RaffleWinner")
	defer span.Finish()
	return s.Deps.RDB.SetPop(s.RoomIDStr, "raffle")
}

func (s *session) RaffleCount(ctx context.Context) (int64, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "RaffleCount")
	defer span.Finish()
	return s.Deps.RDB.SetLen(s.RoomIDStr, "raffle")
}

var errInCooldown = errors.New("bot: in cooldown")

func (s *session) tryCooldown(key string, seconds int) error {
	if seconds == 0 {
		return nil
	}

	if s.UserLevel.CanAccess(levelModerator) {
		return s.Deps.RDB.Mark(seconds, s.RoomIDStr, key)
	}

	seen, err := s.Deps.RDB.CheckAndMark(seconds, s.RoomIDStr, key)

	switch {
	case err != nil:
		return err
	case seen:
		return errInCooldown
	default:
		return nil
	}
}

func (s *session) TryCooldown(ctx context.Context) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "TryCooldown")
	defer span.Finish()

	cooldown := s.Deps.DefaultCooldown

	if s.Channel.Cooldown.Valid {
		cooldown = s.Channel.Cooldown.Int
	}

	return s.tryCooldown("command_cooldown", cooldown)
}

func (s *session) TryRollCooldown(ctx context.Context) error {
	span, _ := opentracing.StartSpanFromContext(ctx, "TryRollCooldown")
	defer span.Finish()
	return s.tryCooldown("roll_cooldown", s.Channel.RollCooldown)
}
