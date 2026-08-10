// Package bot implements the main command for the bot service.
package bot

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/botflags"
	"github.com/hortbot/hortbot/internal/cli/flags/httpflags"
	"github.com/hortbot/hortbot/internal/cli/flags/promflags"
	"github.com/hortbot/hortbot/internal/cli/flags/sqlflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/db/botstate"
	"github.com/hortbot/hortbot/internal/db/chatqueue"
	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/contextx"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/eventsubsync"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

type cmd struct {
	cli.Common
	SQL           sqlflags.SQL
	Twitch        twitchflags.Twitch
	Bot           botflags.Bot
	Prometheus    promflags.Prometheus
	HTTP          httpflags.HTTP
	MessageMaxAge time.Duration `long:"message-max-age" env:"HB_MESSAGE_MAX_AGE" description:"Maximum age of a chat message before it is dropped"`
}

// Command returns a fresh bot command.
func Command() cli.Command {
	return &cmd{
		Common:        cli.Default,
		SQL:           sqlflags.Default,
		Twitch:        twitchflags.Default,
		Bot:           botflags.Default,
		Prometheus:    promflags.Default,
		HTTP:          httpflags.Default,
		MessageMaxAge: 15 * time.Second,
	}
}

func (*cmd) Name() string {
	return "bot"
}

func (c *cmd) Main(ctx context.Context, _ []string) {
	c.Prometheus.Run(ctx)

	httpClient := c.HTTP.Client()
	untrustedClient := c.HTTP.UntrustedClient(ctx)
	db := c.SQL.Open(ctx)
	defer db.Close() //nolint:errcheck

	workCtx, cancelWork := contextx.WithGracePeriod(ctx, 30*time.Second)
	defer cancelWork()
	state := botstate.New()
	twitchAPI := c.Twitch.Client(httpClient)
	eventsubNotifier := eventsubsync.Requests{}

	b := c.Bot.New(workCtx, db, state, eventsubNotifier, twitchAPI, httpClient, untrustedClient)
	defer b.Stop()

	g := errgroupx.FromContext(ctx)

	workers := c.Bot.Workers
	if workers < 1 {
		workers = runtime.GOMAXPROCS(0)
	}

	queue := chatqueue.New(db, workers)
	pollTicker := time.NewTicker(messageQueuePollInterval)
	defer pollTicker.Stop()

	// EventSub identifies the receiving bot by user ID, while bot configuration
	// and message sending use its login.
	var mu sync.Mutex
	var botLoginMap map[int64]string
	var botLoginMapTimestamp time.Time
	getBotLoginMap := func(ctx context.Context) (map[int64]string, error) {
		mu.Lock()
		defer mu.Unlock()

		if botLoginMap != nil && time.Since(botLoginMapTimestamp) < 5*time.Minute {
			return botLoginMap, nil
		}

		var err error
		_, botLoginMap, err = dbsql.New(db).BotMaps(ctx)
		if err != nil {
			return nil, fmt.Errorf("get bots: %w", err)
		}
		botLoginMapTimestamp = time.Now()
		return botLoginMap, nil
	}

	g.Go(func(ctx context.Context) error {
		return runQueueListener(ctx, queue, c.SQL.DB)
	})
	for range workers {
		g.Go(func(ctx context.Context) error {
			return runMessageWorker(ctx, workCtx, queue, b, c.MessageMaxAge, getBotLoginMap, pollTicker.C)
		})
	}
	g.Go(func(ctx context.Context) error {
		return runDatabaseMaintenance(ctx, db, state)
	})

	if err := g.WaitIgnoreStop(); err != nil {
		ctxlog.Info(ctx, "exiting", zap.Error(err))
	}
}
