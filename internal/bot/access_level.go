package bot

import (
	"fmt"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/db/dbsql"
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

func newAccessLevel(s dbsql.AccessLevel) AccessLevel {
	switch s {
	case dbsql.AccessLevelEveryone:
		return AccessLevelEveryone
	case dbsql.AccessLevelSubscriber:
		return AccessLevelSubscriber
	case dbsql.AccessLevelVip:
		return AccessLevelVIP
	case dbsql.AccessLevelModerator:
		return AccessLevelModerator
	case dbsql.AccessLevelBroadcaster:
		return AccessLevelBroadcaster
	case dbsql.AccessLevelAdmin:
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

func (a AccessLevel) CanAccessPG(s dbsql.AccessLevel) bool {
	return a.CanAccess(newAccessLevel(s))
}

func (a AccessLevel) PGEnum() dbsql.AccessLevel {
	switch a { //nolint:exhaustive
	case AccessLevelEveryone:
		return dbsql.AccessLevelEveryone
	case AccessLevelSubscriber:
		return dbsql.AccessLevelSubscriber
	case AccessLevelVIP:
		return dbsql.AccessLevelVip
	case AccessLevelModerator:
		return dbsql.AccessLevelModerator
	case AccessLevelBroadcaster:
		return dbsql.AccessLevelBroadcaster
	case AccessLevelAdmin:
		return dbsql.AccessLevelAdmin
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

func parseLevelPG(s string) dbsql.AccessLevel {
	l := parseLevel(s)
	if l == AccessLevelUnknown {
		return ""
	}
	return l.PGEnum()
}

func pluralAccessLevel(level dbsql.AccessLevel) string {
	return flect.Pluralize(string(level))
}
