package bot

import "strconv"

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
	seen, err := s.Deps.RDB.CheckAndMark(seconds, s.RoomIDStr, "repeat", strconv.FormatInt(id, 10))
	return !seen, err
}

func (s *session) messageCount() (int64, error) {
	return s.Deps.RDB.GetInt64(s.RoomIDStr, "message_count")
}

func (s *session) incrementMessageCount() (int64, error) {
	return s.Deps.RDB.Increment(s.RoomIDStr, "message_count")
}
