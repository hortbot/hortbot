package redis_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dhui/dktest"
	redislib "github.com/go-redis/redis"
	"github.com/hortbot/hortbot/internal/dedupe/redis"
	"gotest.tools/assert"
)

func redisReady(ctx context.Context, c dktest.ContainerInfo) bool {
	ip, port, err := c.FirstPort()
	if err != nil {
		return false
	}

	client := redislib.NewClient(&redislib.Options{
		Addr: fmt.Sprintf("%v:%v", ip, port),
	})

	_, err = client.Ping().Result()
	return err == nil
}

func Test(t *testing.T) {
	if testing.Short() {
		t.Skip("requires starting a docker container")
	}

	dktest.Run(t, "redis:latest", dktest.Options{PortRequired: true, ReadyFunc: redisReady},
		func(t *testing.T, c dktest.ContainerInfo) {
			ip, port, err := c.FirstPort()
			assert.NilError(t, err)

			client := redislib.NewClient(&redislib.Options{
				Addr: fmt.Sprintf("%v:%v", ip, port),
			})
			defer client.Close()

			idNum := 0

			t.Run("CheckNotFound", func(t *testing.T) {
				id := fmt.Sprintf("test-%d", idNum)
				idNum++

				d, err := redis.New(client, time.Second)
				assert.NilError(t, err)

				seen, err := d.Check(id)
				assert.Assert(t, !seen)
				assert.NilError(t, err)
			})

			t.Run("MarkThenCheck", func(t *testing.T) {
				id := fmt.Sprintf("test-%d", idNum)
				idNum++

				d, err := redis.New(client, time.Second)
				assert.NilError(t, err)

				assert.NilError(t, d.Mark(id))
				seen, err := d.Check(id)
				assert.Assert(t, seen)
				assert.NilError(t, err)
			})

			t.Run("CheckAndMark", func(t *testing.T) {
				id := fmt.Sprintf("test-%d", idNum)
				idNum++

				d, err := redis.New(client, time.Second)
				assert.NilError(t, err)

				seen, err := d.CheckAndMark(id)
				assert.Assert(t, !seen)
				assert.NilError(t, err)

				seen, err = d.Check(id)
				assert.Assert(t, seen)
				assert.NilError(t, err)
			})

			t.Run("CheckAndMarkTwice", func(t *testing.T) {
				id := fmt.Sprintf("test-%d", idNum)
				idNum++

				d, err := redis.New(client, time.Second)
				assert.NilError(t, err)

				seen, err := d.CheckAndMark(id)
				assert.Assert(t, !seen)
				assert.NilError(t, err)

				seen, err = d.CheckAndMark(id)
				assert.Assert(t, seen)
				assert.NilError(t, err)
			})

			t.Run("Expire", func(t *testing.T) {
				id := fmt.Sprintf("test-%d", idNum)
				idNum++

				d, err := redis.New(client, time.Second)
				assert.NilError(t, err)

				seen, err := d.CheckAndMark(id)
				assert.Assert(t, !seen)
				assert.NilError(t, err)

				time.Sleep((3 * time.Second) / 2)

				seen, err = d.Check(id)
				assert.Assert(t, !seen)
				assert.NilError(t, err)
			})
		})
}

func TestShortExpiry(t *testing.T) {
	d, err := redis.New(nil, time.Millisecond)
	assert.Assert(t, d == nil)
	assert.Assert(t, err == redis.ErrExpiryTooShort)
}
