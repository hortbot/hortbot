package redis

import (
	"context"
	"time"

	"github.com/bsm/redislock"
	"go.opencensus.io/trace"
)

const keyChannelLock = keyStr("channel_lock")

type Lock interface {
	Unlock() error
	TTL() (time.Duration, error)
	Refresh(ttl time.Duration) (ok bool, err error)
}

type lock struct {
	l *redislock.Lock
}

func (l lock) Unlock() error {
	return l.l.Release()
}

func (l lock) TTL() (time.Duration, error) {
	return l.l.TTL()
}

func (l lock) Refresh(ttl time.Duration) (ok bool, err error) {
	err = l.l.Refresh(ttl, nil)
	return err != redislock.ErrNotObtained, err
}

func (db *DB) LockChannel(ctx context.Context, channel string, ttl time.Duration, maxWait time.Duration) (Lock, bool, error) {
	ctx, span := trace.StartSpan(ctx, "LockChannel")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(keyChannelLock.is(channel))

	opt := &redislock.Options{
		Context:       ctx,
		RetryStrategy: redislock.ExponentialBackoff(16*time.Millisecond, maxWait),
	}

	l, err := redislock.Obtain(client, key, ttl, opt)
	switch err {
	case nil:
		return lock{l: l}, true, nil
	case redislock.ErrNotObtained:
		return nil, false, nil
	default:
		return nil, false, err
	}
}
