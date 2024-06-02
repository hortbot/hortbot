package bot

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func (s *session) LinkPermit(ctx context.Context, user string, expiry time.Duration) error {
	if err := s.Deps.Redis.LinkPermit(ctx, s.RoomIDStr(), user, expiry); err != nil {
		return fmt.Errorf("link permit: %w", err)
	}
	return nil
}

func (s *session) HasLinkPermit(ctx context.Context, user string) (permitted bool, err error) {
	permitted, err = s.Deps.Redis.HasLinkPermit(ctx, s.RoomIDStr(), user)
	if err != nil {
		return false, fmt.Errorf("has link permit: %w", err)
	}
	return permitted, nil
}

func (s *session) Confirm(ctx context.Context, user string, key string, expiry time.Duration) (confirmed bool, err error) {
	confirmed, err = s.Deps.Redis.Confirm(ctx, s.RoomIDStr(), user, key, expiry)
	if err != nil {
		return false, fmt.Errorf("confirm: %w", err)
	}
	return confirmed, nil
}

func (s *session) AutoreplyAllowed(ctx context.Context, id int64, expiry time.Duration) (allowed bool, err error) {
	allowed, err = s.Deps.Redis.AutoreplyAllowed(ctx, s.RoomIDStr(), id, expiry)
	if err != nil {
		return false, fmt.Errorf("autoreply allowed: %w", err)
	}
	return allowed, nil
}

func (s *session) FilterWarned(ctx context.Context, user string, filter string) (warned bool, err error) {
	warned, err = s.Deps.Redis.FilterWarned(ctx, s.RoomIDStr(), user, filter, time.Hour)
	if err != nil {
		return false, fmt.Errorf("filter warned: %w", err)
	}
	return warned, nil
}

func (s *session) RaffleAdd(ctx context.Context, user string) error {
	if err := s.Deps.Redis.RaffleAdd(ctx, s.RoomIDStr(), user); err != nil {
		return fmt.Errorf("raffle add: %w", err)
	}
	return nil
}

func (s *session) RaffleReset(ctx context.Context) error {
	if err := s.Deps.Redis.RaffleReset(ctx, s.RoomIDStr()); err != nil {
		return fmt.Errorf("raffle reset: %w", err)
	}
	return nil
}

func (s *session) RaffleWinner(ctx context.Context) (winner string, ok bool, err error) {
	winner, ok, err = s.Deps.Redis.RaffleWinner(ctx, s.RoomIDStr())
	if err != nil {
		return "", false, fmt.Errorf("raffle winner: %w", err)
	}
	return winner, ok, nil
}

func (s *session) RaffleWinners(ctx context.Context, n int64) ([]string, error) {
	winners, err := s.Deps.Redis.RaffleWinners(ctx, s.RoomIDStr(), n)
	if err != nil {
		return nil, fmt.Errorf("raffle winners: %w", err)
	}
	return winners, nil
}

func (s *session) RaffleCount(ctx context.Context) (int64, error) {
	count, err := s.Deps.Redis.RaffleCount(ctx, s.RoomIDStr())
	if err != nil {
		return 0, fmt.Errorf("raffle count: %w", err)
	}
	return count, nil
}

var errInCooldown = errors.New("bot: in cooldown")

func (s *session) tryCooldown(ctx context.Context, key string, seconds int, allowMods bool) error {
	if seconds == 0 {
		return nil
	}

	dur := time.Duration(seconds) * time.Second

	if allowMods && s.UserLevel.CanAccess(AccessLevelModerator) {
		if err := s.Deps.Redis.MarkCooldown(ctx, s.RoomIDStr(), key, dur); err != nil {
			return fmt.Errorf("marking cooldown: %w", err)
		}
		return nil
	}

	seen, err := s.Deps.Redis.CheckAndMarkCooldown(ctx, s.RoomIDStr(), key, dur)

	switch {
	case err != nil:
		return fmt.Errorf("checking cooldown: %w", err)
	case seen:
		return errInCooldown
	default:
		return nil
	}
}

func (s *session) TryCooldown(ctx context.Context) error {
	cooldown := s.Deps.DefaultCooldown

	if s.Channel.Cooldown.Valid {
		cooldown = s.Channel.Cooldown.Int
	}

	return s.tryCooldown(ctx, "command_cooldown", cooldown, true)
}

func (s *session) TryRollCooldown(ctx context.Context) error {
	return s.tryCooldown(ctx, "roll_cooldown", s.Channel.RollCooldown, true)
}

func (s *session) TryHighlightCooldown(ctx context.Context) error {
	return s.tryCooldown(ctx, "ht_cooldown", 60, false)
}
