package bot

import (
	"fmt"

	"github.com/hortbot/hortbot/internal/db/models"
)

//go:generate gobin -run -m golang.org/x/tools/cmd/stringer -type=AccessLevel

type AccessLevel int

const (
	LevelUnknown     AccessLevel = 0
	LevelEveryone    AccessLevel = 1
	LevelSubscriber  AccessLevel = 2
	LevelModerator   AccessLevel = 3
	LevelBroadcaster AccessLevel = 4
	LevelAdmin       AccessLevel = 99
)

func NewAccessLevel(s string) AccessLevel {
	switch s {
	case models.AccessLevelEveryone:
		return LevelEveryone
	case models.AccessLevelSubscriber:
		return LevelSubscriber
	case models.AccessLevelModerator:
		return LevelModerator
	case models.AccessLevelBroadcaster:
		return LevelBroadcaster
	case models.AccessLevelAdmin:
		return LevelAdmin
	default:
		return LevelUnknown
	}
}

func (a AccessLevel) CanAccess(resource AccessLevel) bool {
	if a == LevelAdmin {
		return true
	}

	if a == 0 || resource == 0 {
		return false
	}

	return a >= resource
}

func (a AccessLevel) PGEnum() string {
	switch a {
	case LevelEveryone:
		return models.AccessLevelEveryone
	case LevelSubscriber:
		return models.AccessLevelSubscriber
	case LevelModerator:
		return models.AccessLevelModerator
	case LevelBroadcaster:
		return models.AccessLevelBroadcaster
	case LevelAdmin:
		return models.AccessLevelAdmin
	default:
		panic(fmt.Sprintf("cannot convert %v to enum value", a))
	}
}
