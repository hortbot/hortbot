package bot

import (
	"database/sql"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/jakebailey/irc"
)

type Session struct {
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

func (s *Session) formatResponse(response string) string {
	response = strings.TrimSpace(response)

	if len(response) >= 4 {
		switch response[:4] {
		case "/me ":
		case ".me ":
			return response
		}
	}

	bullet := s.Channel.Bullet.String
	if bullet == "" {
		bullet = s.Bot.bullet
	}

	return bullet + " " + response
}

func (s *Session) Reply(response string) error {
	return s.Sender.SendMessage("#"+s.ChannelName, s.formatResponse(response))
}
