package bot

import (
	"time"

	"github.com/hortbot/hortbot/internal/pkg/dedupe"
	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/leononame/clock"
)

type sharedDeps struct {
	RDB          *rdb.DB
	Dedupe       dedupe.Deduplicator
	Sender       Sender
	Notifier     Notifier
	Clock        clock.Clock
	Rand         Rand
	UpdateRepeat func(id int64, add bool, interval, wait time.Duration)

	DefaultPrefix   string
	DefaultBullet   string
	DefaultCooldown int

	Admins    map[string]bool
	Whitelist map[string]bool
}

func (s *sharedDeps) IsAllowed(name string) bool {
	if s.Whitelist == nil {
		return true
	}

	if s.Admins[name] {
		return true
	}

	if s.Whitelist[name] {
		return true
	}

	return false
}
