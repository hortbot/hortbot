package bot

import (
	"errors"
	"strconv"
)

func (s *session) LinkPermit(user string, seconds int) error {
	return s.Deps.RDB.Mark(seconds, s.RoomIDStr, "link_permit", user)
}

func (s *session) HasLinkPermit(user string) (permitted bool, err error) {
	return s.Deps.RDB.CheckAndDelete(s.RoomIDStr, "link_permit", user)
}

func (s *session) Confirm(user string, key string, seconds int) (confirmed bool, err error) {
	return s.Deps.RDB.MarkOrDelete(seconds, s.RoomIDStr, "confirm", user, key)
}

func (s *session) RepeatAllowed(id int64, seconds int) (bool, error) {
	seen, err := s.Deps.RDB.CheckAndMark(seconds, s.RoomIDStr, "repeated_command", strconv.FormatInt(id, 10))
	return !seen, err
}

func (s *session) ScheduledAllowed(id int64, seconds int) (bool, error) {
	seen, err := s.Deps.RDB.CheckAndMark(seconds, s.RoomIDStr, "scheduled_command", strconv.FormatInt(id, 10))
	return !seen, err
}

func (s *session) MessageCount() (int64, error) {
	return s.Deps.RDB.GetInt64(s.RoomIDStr, "message_count")
}

func (s *session) IncrementMessageCount() (int64, error) {
	return s.Deps.RDB.Increment(s.RoomIDStr, "message_count")
}

func (s *session) AutoreplyAllowed(id int64, seconds int) (bool, error) {
	seen, err := s.Deps.RDB.CheckAndMark(seconds, s.RoomIDStr, "autoreply", strconv.FormatInt(id, 10))
	return !seen, err
}

func (s *session) FilterWarned(user string, filter string) (bool, error) {
	return s.Deps.RDB.CheckAndRefresh(3600, s.RoomIDStr, "filter_warning", filter, user)
}

func (s *session) RaffleAdd(user string) error {
	return s.Deps.RDB.SetAdd(user, s.RoomIDStr, "raffle")
}

func (s *session) RaffleReset() error {
	return s.Deps.RDB.SetClear(s.RoomIDStr, "raffle")
}

func (s *session) RaffleWinner() (string, bool, error) {
	return s.Deps.RDB.SetPop(s.RoomIDStr, "raffle")
}

func (s *session) RaffleCount() (int64, error) {
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

func (s *session) TryCooldown() error {
	cooldown := s.Deps.DefaultCooldown

	if s.Channel.Cooldown.Valid {
		cooldown = s.Channel.Cooldown.Int
	}

	return s.tryCooldown("command_cooldown", cooldown)
}

func (s *session) TryRollCooldown() error {
	return s.tryCooldown("roll_cooldown", s.Channel.RollCooldown)
}
