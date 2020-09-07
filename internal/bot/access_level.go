package bot

import (
	"fmt"

	"github.com/hortbot/hortbot/internal/db/models"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=accessLevel -trimprefix=level

type accessLevel int

const (
	levelUnknown accessLevel = iota
	levelEveryone
	levelSubscriber
	levelModerator
	levelBroadcaster
	levelAdmin
	levelSuperAdmin
)

const (
	levelMinValid = levelEveryone
	levelMaxValid = levelSuperAdmin
)

func newAccessLevel(s string) accessLevel {
	switch s {
	case models.AccessLevelEveryone:
		return levelEveryone
	case models.AccessLevelSubscriber:
		return levelSubscriber
	case models.AccessLevelModerator:
		return levelModerator
	case models.AccessLevelBroadcaster:
		return levelBroadcaster
	case models.AccessLevelAdmin:
		return levelAdmin
	default:
		return levelUnknown
	}
}

func (a accessLevel) Valid() bool {
	return a >= levelMinValid && a <= levelMaxValid
}

func (a accessLevel) CanAccess(resource accessLevel) bool {
	if a == levelSuperAdmin {
		return true
	}

	if !a.Valid() || !resource.Valid() {
		return false
	}

	return a >= resource
}

func (a accessLevel) CanAccessPG(s string) bool {
	return a.CanAccess(newAccessLevel(s))
}

func (a accessLevel) PGEnum() string {
	switch a {
	case levelEveryone:
		return models.AccessLevelEveryone
	case levelSubscriber:
		return models.AccessLevelSubscriber
	case levelModerator:
		return models.AccessLevelModerator
	case levelBroadcaster:
		return models.AccessLevelBroadcaster
	case levelAdmin:
		return models.AccessLevelAdmin
	default:
		panic(fmt.Sprintf("cannot convert %v to enum value", a))
	}
}

func parseLevel(s string) accessLevel {
	switch s {
	case "everyone", "all", "everybody", "normal":
		return levelEveryone
	case "sub", "subs", "subscriber", "subscribers", "regular", "regulars", "reg", "regs":
		return levelSubscriber
	case "mod", "mods", "moderator", "moderators":
		return levelModerator
	case "broadcaster", "broadcasters", "owner", "owners", "streamer", "streamers":
		return levelBroadcaster
	case "admin", "admins":
		return levelAdmin
	default:
		return levelUnknown
	}
}

func parseLevelPG(s string) string {
	l := parseLevel(s)
	if l == levelUnknown {
		return ""
	}
	return l.PGEnum()
}
