package bot

import (
	"fmt"

	"github.com/hortbot/hortbot/internal/db/models"
)

//go:generate go tool golang.org/x/tools/cmd/stringer -type=AccessLevel -trimprefix=AccessLevel

type AccessLevel int

const (
	AccessLevelUnknown AccessLevel = iota
	AccessLevelEveryone
	AccessLevelSubscriber
	AccessLevelVIP
	AccessLevelModerator
	AccessLevelBroadcaster
	AccessLevelAdmin
	AccessLevelSuperAdmin
)

const (
	levelMinValid = AccessLevelEveryone
	levelMaxValid = AccessLevelSuperAdmin
)

func newAccessLevel(s string) AccessLevel {
	switch s {
	case models.AccessLevelEveryone:
		return AccessLevelEveryone
	case models.AccessLevelSubscriber:
		return AccessLevelSubscriber
	case models.AccessLevelVip:
		return AccessLevelVIP
	case models.AccessLevelModerator:
		return AccessLevelModerator
	case models.AccessLevelBroadcaster:
		return AccessLevelBroadcaster
	case models.AccessLevelAdmin:
		return AccessLevelAdmin
	default:
		return AccessLevelUnknown
	}
}

func (a AccessLevel) Valid() bool {
	return a >= levelMinValid && a <= levelMaxValid
}

func (a AccessLevel) CanAccess(resource AccessLevel) bool {
	if a == AccessLevelSuperAdmin {
		return true
	}

	if !a.Valid() || !resource.Valid() {
		return false
	}

	return a >= resource
}

func (a AccessLevel) CanAccessPG(s string) bool {
	return a.CanAccess(newAccessLevel(s))
}

func (a AccessLevel) PGEnum() string {
	switch a { //nolint:exhaustive
	case AccessLevelEveryone:
		return models.AccessLevelEveryone
	case AccessLevelSubscriber:
		return models.AccessLevelSubscriber
	case AccessLevelVIP:
		return models.AccessLevelVip
	case AccessLevelModerator:
		return models.AccessLevelModerator
	case AccessLevelBroadcaster:
		return models.AccessLevelBroadcaster
	case AccessLevelAdmin:
		return models.AccessLevelAdmin
	default:
		panic(fmt.Sprintf("cannot convert %v to enum value", a))
	}
}

func parseLevel(s string) AccessLevel {
	switch s {
	case "everyone", "all", "everybody", "normal":
		return AccessLevelEveryone
	case "sub", "subs", "subscriber", "subscribers", "regular", "regulars", "reg", "regs":
		return AccessLevelSubscriber
	case "vip", "vips":
		return AccessLevelVIP
	case "mod", "mods", "moderator", "moderators":
		return AccessLevelModerator
	case "broadcaster", "broadcasters", "owner", "owners", "streamer", "streamers":
		return AccessLevelBroadcaster
	case "admin", "admins":
		return AccessLevelAdmin
	default:
		return AccessLevelUnknown
	}
}

func parseLevelPG(s string) string {
	l := parseLevel(s)
	if l == AccessLevelUnknown {
		return ""
	}
	return l.PGEnum()
}
