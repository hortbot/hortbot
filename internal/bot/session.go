package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/findlinks"
	"github.com/jakebailey/irc"
)

//go:generate gobin -run -m golang.org/x/tools/cmd/stringer -type=sessionType

type sessionType int

const (
	sessionUnknown sessionType = iota
	sessionNormal
	sessionRepeat
	sessionAutoreply
	sessionSubNotification
)

type session struct {
	Type sessionType

	Origin string
	M      *irc.Message

	Deps *sharedDeps
	Tx   *sql.Tx

	Start   time.Time
	TMISent time.Time

	N          int64
	ID         string
	RoomID     int64
	RoomIDStr  string
	IRCChannel string
	Message    string
	Me         bool

	User        string
	UserDisplay string
	UserID      int64
	UserLevel   accessLevel
	Ignored     bool

	links    []*url.URL
	linksSet bool

	tracks    []lastfm.Track
	tracksSet bool

	Channel *models.Channel

	CommandParams     string
	OrigCommandParams string

	usageContext string

	Silent bool
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
	if s.Silent {
		return nil
	}

	response = strings.TrimSpace(response)

	if response == "" {
		return nil
	}

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

func (s *session) parseUserLevel() accessLevel {
	if s.Deps.Admins[s.User] {
		return levelAdmin
	}

	// Tags are present, safe to not check for nil

	tags := s.M.Tags

	if isTesting && tags["testing-admin"] != "" {
		return levelAdmin
	}

	if s.User == s.IRCChannel {
		return levelBroadcaster
	}

	if tags["mod"] == "1" {
		return levelModerator
	}

	badges := parseBadges(tags["badges"])

	switch {
	case badges["broadcaster"] != "":
		return levelBroadcaster
	case badges["moderator"] != "":
		return levelModerator
	case badges["subscriber"] != "", badges["vip"] != "", tags["subscriber"] == "1":
		return levelSubscriber
	}

	if tags["user-type"] == "mod" {
		return levelModerator
	}

	if s.Channel != nil {
		for _, owner := range s.Channel.CustomOwners {
			if s.User == owner {
				return levelBroadcaster
			}
		}

		for _, mod := range s.Channel.CustomMods {
			if s.User == mod {
				return levelModerator
			}
		}

		for _, reg := range s.Channel.CustomRegulars {
			if s.User == reg {
				return levelSubscriber
			}
		}
	}

	return levelEveryone
}

func (s *session) IsAdmin() bool {
	return s.UserLevel.CanAccess(levelAdmin)
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
	case "host":
	case "unhost":
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
		s.linksSet = true
	}

	return s.links
}

var errLastFMDisabled = errors.New("bot: LastFM disabled")

func (s *session) Tracks() ([]lastfm.Track, error) {
	if s.Deps.LastFM == nil {
		return nil, errLastFMDisabled
	}

	if s.Channel.LastFM == "" {
		return nil, errLastFMDisabled
	}

	if !s.tracksSet {
		var err error
		s.tracks, err = s.Deps.LastFM.RecentTracks(s.Channel.LastFM, 2)
		if err != nil {
			return nil, err
		}

		s.tracksSet = true
	}

	return s.tracks, nil
}
