package bot

import (
	"database/sql"
	"strings"

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
	response = strings.TrimSpace(response)

	if len(response) >= 4 {
		switch response[:4] {
		case "/me ":
		case ".me ":
			return response
		}
	}

	bullet := c.Channel.Bullet.String
	if bullet == "" {
		bullet = c.Bot.bullet
	}

	return bullet + " " + response
}

func (c *Context) Reply(response string) error {
	return c.Sender.SendMessage("#"+c.ChannelName, c.formatResponse(response))
}
