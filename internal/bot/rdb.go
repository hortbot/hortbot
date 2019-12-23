package bot

import (
	"context"
	"errors"
	"time"

	"go.opencensus.io/trace"
)

func (s *session) LinkPermit(ctx context.Context, user string, expiry time.Duration) error {
	return s.Deps.Redis.LinkPermit(ctx, s.RoomIDStr(), user, expiry)
}

func (s *session) HasLinkPermit(ctx context.Context, user string) (permitted bool, err error) {
	return s.Deps.Redis.HasLinkPermit(ctx, s.RoomIDStr(), user)
}

func (s *session) Confirm(ctx context.Context, user string, key string, expiry time.Duration) (confirmed bool, err error) {
	return s.Deps.Redis.Confirm(ctx, s.RoomIDStr(), user, key, expiry)
}

func (s *session) AutoreplyAllowed(ctx context.Context, id int64, expiry time.Duration) (bool, error) {
	return s.Deps.Redis.AutoreplyAllowed(ctx, s.RoomIDStr(), id, expiry)
}

func (s *session) FilterWarned(ctx context.Context, user string, filter string) (bool, error) {
	return s.Deps.Redis.FilterWarned(ctx, s.RoomIDStr(), user, filter, time.Hour)
}

func (s *session) RaffleAdd(ctx context.Context, user string) error {
	return s.Deps.Redis.RaffleAdd(ctx, s.RoomIDStr(), user)
}

func (s *session) RaffleReset(ctx context.Context) error {
	return s.Deps.Redis.RaffleReset(ctx, s.RoomIDStr())
}

func (s *session) RaffleWinner(ctx context.Context) (string, bool, error) {
	return s.Deps.Redis.RaffleWinner(ctx, s.RoomIDStr())
}

func (s *session) RaffleCount(ctx context.Context) (int64, error) {
	return s.Deps.Redis.RaffleCount(ctx, s.RoomIDStr())
}

var errInCooldown = errors.New("bot: in cooldown")

func (s *session) tryCooldown(ctx context.Context, key string, seconds int) error {
	if seconds == 0 {
		return nil
	}

	dur := time.Duration(seconds) * time.Second

	if s.UserLevel.CanAccess(levelModerator) {
		return s.Deps.Redis.MarkCooldown(ctx, s.RoomIDStr(), key, dur)
	}

	seen, err := s.Deps.Redis.CheckAndMarkCooldown(ctx, s.RoomIDStr(), key, dur)

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
	ctx, span := trace.StartSpan(ctx, "TryCooldown")
	defer span.End()

	cooldown := s.Deps.DefaultCooldown

	if s.Channel.Cooldown.Valid {
		cooldown = s.Channel.Cooldown.Int
	}

	return s.tryCooldown(ctx, "command_cooldown", cooldown)
}

func (s *session) TryRollCooldown(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "TryRollCooldown")
	defer span.End()
	return s.tryCooldown(ctx, "roll_cooldown", s.Channel.RollCooldown)
}
