// Package rdb implements a type-safe wrapper for redis with custom
// commands for atomic operations.
package rdb

import (
	"strings"

	"github.com/go-redis/redis"
)

const keySep = ":"

var keyEscaper = strings.NewReplacer(keySep, keySep+keySep)

// DB is a redis wrapper.
type DB struct {
	client redis.Cmdable
	prefix string
}

// Option configures the DB created by New.
type Option func(*DB)

// New creates a new redis wrapper from the specified Cmdable.
func New(client redis.Cmdable, options ...Option) (*DB, error) {
	if err := checkAndMark.Load(client).Err(); err != nil {
		return nil, err
	}

	if err := checkAndRefresh.Load(client).Err(); err != nil {
		return nil, err
	}

	if err := markOrDelete.Load(client).Err(); err != nil {
		return nil, err
	}

	d := &DB{
		client: client,
	}

	for _, o := range options {
		o(d)
	}

	return d, nil
}

// KeyPrefix sets the DB's key prefix, which allows multiple calls to RDB's
// methods to operate in different namespaces. No prefix can conflict with
// another, including the empty string (the default).
func KeyPrefix(prefix string) Option {
	if prefix != "" {
		prefix = keyEscaper.Replace(prefix)
	}

	return func(d *DB) {
		d.prefix = prefix
	}
}

func (d *DB) buildKey(key string, more ...string) string {
	var builder strings.Builder

	size := len(d.prefix) + len(key) + 2*len(keySep)

	for _, k := range more {
		size += len(k) + 1
	}

	builder.Grow(size)

	builder.WriteString(d.prefix)
	builder.WriteString(keySep)

	keyEscaper.WriteString(&builder, key) //nolint:errcheck
	for _, k := range more {
		builder.WriteString(keySep)
		keyEscaper.WriteString(&builder, k) //nolint:errcheck
	}

	return builder.String()
}

func (d *DB) runScript(s *redis.Script, key string, args ...interface{}) (exists bool, err error) {
	b, err := s.Run(d.client, []string{key}, args...).Bool()
	if err == redis.Nil {
		err = nil
	}

	return b, err
}
