package redis_test

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"testing"
	"time"

	redislib "github.com/go-redis/redis"
	"github.com/hortbot/hortbot/internal/pkg/dedupe/redis"
	"github.com/hortbot/hortbot/internal/pkg/testutil/redistest"
	"gotest.tools/assert"
)

var nextID int64

func getNextID() string {
	id := atomic.AddInt64(&nextID, 1)
	return fmt.Sprintf("test-%d", id)
}

var client *redislib.Client

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func TestMain(m *testing.M) {
	var status int
	defer func() {
		os.Exit(status)
	}()

	var cleanup func()
	var err error

	client, cleanup, err = redistest.New()
	must(err)
	defer cleanup()

	status = m.Run()
}

func TestCheckNotFound(t *testing.T) {
	t.Parallel()

	id := getNextID()

	d, err := redis.New(client, time.Second)
	assert.NilError(t, err)

	seen, err := d.Check(id)
	assert.Assert(t, !seen)
	assert.NilError(t, err)
}

func TestMarkThenCheck(t *testing.T) {
	t.Parallel()

	id := getNextID()

	d, err := redis.New(client, time.Second)
	assert.NilError(t, err)

	assert.NilError(t, d.Mark(id))
	seen, err := d.Check(id)
	assert.Assert(t, seen)
	assert.NilError(t, err)
}

func TestMarkMarkThenCheck(t *testing.T) {
	t.Parallel()

	id := getNextID()

	d, err := redis.New(client, time.Second)
	assert.NilError(t, err)

	assert.NilError(t, d.Mark(id))
	assert.NilError(t, d.Mark(id))
	seen, err := d.Check(id)
	assert.Assert(t, seen)
	assert.NilError(t, err)
}

func TestCheckAndMark(t *testing.T) {
	t.Parallel()

	id := getNextID()

	d, err := redis.New(client, time.Second)
	assert.NilError(t, err)

	seen, err := d.CheckAndMark(id)
	assert.Assert(t, !seen)
	assert.NilError(t, err)

	seen, err = d.Check(id)
	assert.Assert(t, seen)
	assert.NilError(t, err)
}

func TestCheckAndMarkTwice(t *testing.T) {
	t.Parallel()

	id := getNextID()

	d, err := redis.New(client, time.Second)
	assert.NilError(t, err)

	seen, err := d.CheckAndMark(id)
	assert.Assert(t, !seen)
	assert.NilError(t, err)

	seen, err = d.CheckAndMark(id)
	assert.Assert(t, seen)
	assert.NilError(t, err)
}

func TestExpire(t *testing.T) {
	t.Parallel()

	id := getNextID()

	d, err := redis.New(client, time.Second)
	assert.NilError(t, err)

	seen, err := d.CheckAndMark(id)
	assert.Assert(t, !seen)
	assert.NilError(t, err)

	time.Sleep((3 * time.Second) / 2)

	seen, err = d.Check(id)
	assert.Assert(t, !seen)
	assert.NilError(t, err)
}

func TestShortExpiry(t *testing.T) {
	t.Parallel()

	d, err := redis.New(nil, time.Millisecond)
	assert.Assert(t, d == nil)
	assert.Assert(t, err == redis.ErrExpiryTooShort)
}

func TestBadCheckScript(t *testing.T) {
	defer redis.ReplaceCheck("local")()

	_, err := redis.New(client, time.Second)
	assert.ErrorContains(t, err, "Error compiling script")
}

func TestBadCheckAndMarkScript(t *testing.T) {
	defer redis.ReplaceCheckAndMark("local")()

	_, err := redis.New(client, time.Second)
	assert.ErrorContains(t, err, "Error compiling script")
}
