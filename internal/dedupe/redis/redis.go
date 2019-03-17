package redis

import (
	"errors"
	"time"

	"github.com/go-redis/redis"
	"github.com/hortbot/hortbot/internal/dedupe"
)

var check = redis.NewScript(`
local exists = redis.pcall('EXISTS', KEYS[1])
if exists == 1 then
	redis.pcall('EXPIRE', KEYS[1], ARGV[1])
	return true
end
return false
`)

var checkAndMark = redis.NewScript(`
local v = redis.pcall('GETSET', KEYS[1], '1')
redis.call('EXPIRE', KEYS[1], ARGV[1])
return v ~= false
`)

var ErrExpiryTooShort = errors.New("expiry is too short")

type Dedupe struct {
	r      redis.Cmdable
	expiry time.Duration
}

func New(r redis.Cmdable, expiry time.Duration) (*Dedupe, error) {
	if expiry < time.Second {
		return nil, ErrExpiryTooShort
	}

	if err := check.Load(r).Err(); err != nil {
		return nil, err
	}

	if err := checkAndMark.Load(r).Err(); err != nil {
		return nil, err
	}

	return &Dedupe{
		r:      r,
		expiry: expiry,
	}, nil
}

var _ dedupe.Deduplicator = (*Dedupe)(nil)

func (d *Dedupe) Mark(id string) error {
	err := d.r.Set(id, "1", d.expiry).Err()
	if err == redis.Nil {
		return nil
	}
	return err
}

func (d *Dedupe) Check(id string) (seen bool, err error) {
	return d.runScript(check, id)
}

func (d *Dedupe) CheckAndMark(id string) (seen bool, err error) {
	return d.runScript(checkAndMark, id)
}

func (d *Dedupe) runScript(s *redis.Script, id string) (bool, error) {
	b, err := s.Run(d.r, []string{id}, int(d.expiry.Seconds())).Bool()
	if err == redis.Nil {
		err = nil
	}

	if err != nil {
		return false, err
	}

	return b, nil
}
