package bot_test

import (
	"os"
	"testing"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/pkg/docker/dpostgres/pgpool"
)

func init() {
	bot.Testing()
}

var pool pgpool.Pool

func TestMain(m *testing.M) {
	status := 1
	defer func() {
		if r := recover(); r != nil {
			panic(r)
		}
		os.Exit(status)
	}()

	defer pool.Cleanup()

	status = m.Run()
}
