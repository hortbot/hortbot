package bot

import (
	"github.com/efritz/glock"
	"github.com/hortbot/hortbot/internal/pkg/dedupe"
	"github.com/hortbot/hortbot/internal/pkg/rdb"
)

type sharedDeps struct {
	RDB      *rdb.DB
	Dedupe   dedupe.Deduplicator
	Sender   Sender
	Notifier Notifier
	Clock    glock.Clock
	Rand     Rand

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
