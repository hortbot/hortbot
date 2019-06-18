package bot

func (s *session) LinkPermit(user string, seconds int) error {
	return s.Deps.RDB.Mark(seconds, "link_permit", s.RoomIDStr, user)
}

func (s *session) HasLinkPermit(user string) (permitted bool, err error) {
	return s.Deps.RDB.CheckAndDelete("link_permit", s.RoomIDStr, user)
}

func (s *session) Confirm(user string, key string, seconds int) (confirmed bool, err error) {
	return s.Deps.RDB.MarkOrDelete(seconds, "confirm", s.RoomIDStr, user, key)
}
