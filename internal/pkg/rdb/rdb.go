package rdb

import (
	"strings"
	"time"

	"github.com/go-redis/redis"
)

var checkAndMark = redis.NewScript(`
local exists = redis.pcall('GETSET', KEYS[1], '1')
redis.call('EXPIRE', KEYS[1], ARGV[1])
return exists ~= false
`)

var checkAndRefresh = redis.NewScript(`
local exists = redis.pcall('EXISTS', KEYS[1])
if exists == 1 then
	redis.pcall('EXPIRE', KEYS[1], ARGV[1])
	return true
end
return false
`)

var markOrDelete = redis.NewScript(`
local exists = redis.pcall('GETSET', KEYS[1], '1')
if exists ~= false then
	redis.pcall('DEL', KEYS[1])
	return true
end
redis.pcall('EXPIRE', KEYS[1], ARGV[1])
return false
`)

const keySep = ":"

var keyEscaper = strings.NewReplacer(keySep, keySep+keySep)

type DB struct {
	client redis.Cmdable
	prefix string
}

func New(client redis.Cmdable, options ...func(*DB)) (*DB, error) {
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

func KeyPrefix(prefix string) func(*DB) {
	if prefix != "" {
		prefix = keyEscaper.Replace(prefix)
	}

	return func(d *DB) {
		d.prefix = prefix
	}
}

func (d *DB) Mark(seconds int, key string, more ...string) error {
	k := d.buildKey(key, more...)
	dur := time.Duration(seconds) * time.Second
	return d.client.Set(k, "1", dur).Err()
}

func (d *DB) Check(key string, more ...string) (exists bool, err error) {
	k := d.buildKey(key, more...)

	v, err := d.client.Exists(k).Result()
	return v == 1, err
}

func (d *DB) CheckAndMark(seconds int, key string, more ...string) (exists bool, err error) {
	k := d.buildKey(key, more...)
	return d.runScript(checkAndMark, k, seconds)
}

func (d *DB) CheckAndRefresh(seconds int, key string, more ...string) (exists bool, err error) {
	k := d.buildKey(key, more...)
	return d.runScript(checkAndRefresh, k, seconds)
}

func (d *DB) CheckAndDelete(key string, more ...string) (exists bool, err error) {
	k := d.buildKey(key, more...)
	v, err := d.client.Del(k).Result()
	return v == 1, err
}

func (d *DB) MarkOrDelete(seconds int, key string, more ...string) (exists bool, err error) {
	k := d.buildKey(key, more...)
	return d.runScript(markOrDelete, k, seconds)
}

func (d *DB) GetInt64(key string, more ...string) (int64, error) {
	k := d.buildKey(key, more...)
	return d.client.Get(k).Int64()
}

func (d *DB) Increment(key string, more ...string) (int64, error) {
	k := d.buildKey(key, more...)
	return d.client.Incr(k).Result()
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
