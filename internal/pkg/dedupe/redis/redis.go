package redis

import (
	"context"
	"errors"
	"time"

	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/dedupe"
)

var ErrExpiryTooShort = errors.New("expiry is too short")

type Dedupe struct {
	db     *redis.DB
	expiry time.Duration
}

func New(rdb *redis.DB, expiry time.Duration) (*Dedupe, error) {
	if expiry < time.Second {
		return nil, ErrExpiryTooShort
	}

	return &Dedupe{
		db:     rdb,
		expiry: expiry,
	}, nil
}

var _ dedupe.Deduplicator = (*Dedupe)(nil)

func (d *Dedupe) Mark(id string) error {
	return d.db.DedupeMark(context.TODO(), id, d.expiry)
}

func (d *Dedupe) Check(id string) (seen bool, err error) {
	return d.db.DedupeCheck(context.TODO(), id, d.expiry)
}

func (d *Dedupe) CheckAndMark(id string) (seen bool, err error) {
	return d.db.DedupeCheckAndMark(context.TODO(), id, d.expiry)
}
