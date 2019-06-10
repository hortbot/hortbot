package redis

import (
	"errors"
	"time"

	"github.com/go-redis/redis"
	"github.com/hortbot/hortbot/internal/pkg/dedupe"
	"github.com/hortbot/hortbot/internal/pkg/rdb"
)

var ErrExpiryTooShort = errors.New("expiry is too short")

type Dedupe struct {
	d      *rdb.DB
	expiry int
}

func New(r redis.Cmdable, expiry time.Duration) (*Dedupe, error) {
	if expiry < time.Second {
		return nil, ErrExpiryTooShort
	}

	d, err := rdb.New(r, rdb.KeyPrefix("dedupe"))
	if err != nil {
		return nil, err
	}

	return &Dedupe{
		d:      d,
		expiry: int(expiry.Seconds()),
	}, nil
}

var _ dedupe.Deduplicator = (*Dedupe)(nil)

func (d *Dedupe) Mark(id string) error {
	return d.d.Mark(d.expiry, id)
}

func (d *Dedupe) Check(id string) (seen bool, err error) {
	return d.d.CheckAndRefresh(d.expiry, id)
}

func (d *Dedupe) CheckAndMark(id string) (seen bool, err error) {
	return d.d.CheckAndMark(d.expiry, id)
}
