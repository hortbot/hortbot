package bot

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/findlinks"
	"github.com/jakebailey/irc"
)

type session struct {
	Origin string
	M      *irc.Message

	Deps *sharedDeps
	Tx   *sql.Tx
	RDB  *RDB

	Start   time.Time
	TMISent time.Time

	ID         string
	RoomID     int64
	IRCChannel string
	Message    string
	Me         bool

	User        string
	UserDisplay string
	UserID      int64
	UserLevel   AccessLevel
	Ignored     bool

	links    []*url.URL
	linksSet bool

	Channel *models.Channel

	CommandParams     string
	OrigCommandParams string

	usageContext string
}

func (s *session) formatResponse(response string) string {
	response = strings.TrimSpace(response)

	if len(response) >= 4 {
		switch response[:4] {
		case "/me ", ".me ":
			return response
		}
	}

	bullet := s.Deps.DefaultBullet

	if s.Channel != nil && s.Channel.Bullet.Valid {
		bullet = s.Channel.Bullet.String
	}

	return bullet + " " + response
}

func (s *session) Reply(response string) error {
	return s.Deps.Sender.SendMessage(s.Origin, "#"+s.IRCChannel, s.formatResponse(response))
}

func (s *session) Replyf(format string, args ...interface{}) error {
	response := fmt.Sprintf(format, args...)
	return s.Reply(response)
}

func (s *session) ReplyUsage(usage string) error {
	var builder strings.Builder
	builder.WriteString("usage: ")
	builder.WriteString(s.Channel.Prefix)
	builder.WriteString(s.usageContext)
	builder.WriteString(usage)

	return s.Reply(builder.String())
}

func (s *session) UsageContext(command string) func() {
	if command == "" {
		return func() {}
	}

	do := true

	old := s.usageContext
	s.usageContext += command + " "
	return func() {
		if do {
			s.usageContext = old
			do = false
		}
	}
}

func (s *session) SetUserLevel() {
	s.UserLevel = s.parseUserLevel()
}

func (s *session) parseUserLevel() AccessLevel {
	if s.Deps.Admins[s.User] {
		return LevelAdmin
	}

	// Tags are present, safe to not check for nil

	tags := s.M.Tags

	if isTesting && tags["testing-admin"] != "" {
		return LevelAdmin
	}

	if s.User == s.IRCChannel {
		return LevelBroadcaster
	}

	if tags["mod"] == "1" {
		return LevelModerator
	}

	badges := parseBadges(tags["badges"])

	switch {
	case badges["broadcaster"] != "":
		return LevelBroadcaster
	case badges["moderator"] != "":
		return LevelModerator
	case badges["subscriber"] != "", badges["vip"] != "", tags["subscriber"] == "1":
		return LevelSubscriber
	}

	if tags["user-type"] == "mod" {
		return LevelModerator
	}

	if s.Channel != nil {
		for _, owner := range s.Channel.CustomOwners {
			if s.User == owner {
				return LevelBroadcaster
			}
		}

		for _, mod := range s.Channel.CustomMods {
			if s.User == mod {
				return LevelModerator
			}
		}

		for _, reg := range s.Channel.CustomRegulars {
			if s.User == reg {
				return LevelSubscriber
			}
		}
	}

	return LevelEveryone
}

func (s *session) IsAdmin() bool {
	return s.UserLevel.CanAccess(LevelAdmin)
}

func (s *session) IsInCooldown() bool {
	seconds := s.Deps.Clock.Since(s.Channel.LastCommandAt).Seconds()
	cooldown := s.Deps.DefaultCooldown

	if s.Channel.Cooldown.Valid {
		cooldown = s.Channel.Cooldown.Int
	}

	return seconds < float64(cooldown)
}

func (s *session) SendCommand(command string, args ...string) error {
	switch command {
	case "slow":
	case "slowoff":
	case "subscribers":
	case "subscribersoff":
	case "r9kbeta":
	case "r9kbetaoff":
	case "ban":
	case "unban":
	case "timeout":
	case "untimeout":
	case "me":
	case "delete":
	case "clear":
	default:
		panic("attempt to use IRC command " + command)
	}

	var builder strings.Builder
	builder.WriteByte('/')
	builder.WriteString(command)

	for _, arg := range args {
		builder.WriteByte(' ')
		builder.WriteString(arg)
	}

	return s.Deps.Sender.SendMessage(s.Origin, "#"+s.IRCChannel, builder.String())
}

func (s *session) DeleteMessage() error {
	return s.SendCommand("delete", s.ID)
}

func (s *session) Links() []*url.URL {
	if !s.linksSet {
		s.links = findlinks.Find(s.Message, "http", "https", "ftp")
	}

	return s.links
}
