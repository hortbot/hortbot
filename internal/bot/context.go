package bot

import (
	"database/sql"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/jakebailey/irc"
)

type Context struct {
	M *irc.Message

	ID          string
	RoomID      int64
	ChannelName string
	Message     string

	Bot    *Bot
	Tx     *sql.Tx
	Sender MessageSender

	Channel *models.Channel
}

func (c *Context) formatResponse(response string) string {
	// TODO: /me

	bullet := c.Channel.Bullet.String
	if bullet == "" {
		bullet = c.Bot.bullet
	}

	return bullet + " " + response
}
