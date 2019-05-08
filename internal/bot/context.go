package bot

import (
	"strconv"

	"github.com/jakebailey/irc"
)

type Context struct {
	M *irc.Message

	// ID is the UUID specified by twitch in the "id" tag. It may be empty.
	ID string

	RoomID int64
	UserID int64
}

func NewContext(m *irc.Message) *Context {
	c := &Context{
		M: m,
	}

	c.ID, _ = c.Tag("id")

	roomID, _ := c.Tag("room-id")
	userID, _ := c.Tag("user-id")

	c.RoomID, _ = strconv.ParseInt(roomID, 10, 64)
	c.UserID, _ = strconv.ParseInt(userID, 10, 64)

	return c
}

func (c *Context) Tag(key string) (value string, ok bool) {
	if c.M.Tags == nil {
		return "", false
	}

	value, ok = c.M.Tags[key]
	return value, ok
}
