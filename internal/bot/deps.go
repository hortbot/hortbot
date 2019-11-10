package bot

import (
	"context"
	"time"

	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/extralife"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apis/steam"
	"github.com/hortbot/hortbot/internal/pkg/apis/tinyurl"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apis/urban"
	"github.com/hortbot/hortbot/internal/pkg/apis/xkcd"
	"github.com/hortbot/hortbot/internal/pkg/apis/youtube"
	"github.com/hortbot/hortbot/internal/pkg/recache"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/leononame/clock"
)

type sharedDeps struct {
	Redis    *redis.DB
	Sender   Sender
	Notifier Notifier
	Clock    clock.Clock
	Rand     Rand

	LastFM    lastfm.API
	YouTube   youtube.API
	XKCD      xkcd.API
	ExtraLife extralife.API
	Twitch    twitch.API
	Steam     steam.API
	TinyURL   tinyurl.API
	Urban     urban.API

	ReCache *recache.RegexpCache

	// TODO: split these into an interface.

	UpdateRepeat   func(id int64, add bool, interval, wait time.Duration)
	UpdateSchedule func(id int64, add bool, expr *repeat.Cron)
	ReloadRepeats  func(ctx context.Context) error
	CountRepeats   func() (repeats, schedules int)

	BulletMap map[string]string

	DefaultCooldown int

	Admins    map[string]bool
	Whitelist map[string]bool

	WebAddr    string
	WebAddrMap map[string]string
}

func (s *sharedDeps) IsAllowed(name string) bool {
	if s.Whitelist == nil {
		return true
	}
	return s.Admins[name] || s.Whitelist[name]
}
