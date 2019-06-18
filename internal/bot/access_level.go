package bot

import (
	"fmt"

	"github.com/hortbot/hortbot/internal/db/models"
)

//go:generate gobin -run -m golang.org/x/tools/cmd/stringer -type=accessLevel

type accessLevel int

const (
	levelUnknown     accessLevel = 0
	levelEveryone    accessLevel = 1
	levelSubscriber  accessLevel = 2
	levelModerator   accessLevel = 3
	levelBroadcaster accessLevel = 4
	levelAdmin       accessLevel = 99
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

func (a accessLevel) CanAccess(resource accessLevel) bool {
	if a == levelAdmin {
		return true
	}

	if a <= levelUnknown || resource <= levelUnknown {
		return false
	}

	return a >= resource
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
