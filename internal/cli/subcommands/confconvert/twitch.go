package confconvert

import (
	"context"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
)

var (
	tw       *twitch.Twitch
	lastCall time.Time
)

func (cmd *cmd) getChannelByID(ctx context.Context, id int64) (name string, displayName string, err error) {
	if err := cmd.twitchThrottle(ctx); err != nil {
		return "", "", err
	}

	ch, err := tw.GetChannelByID(ctx, id)
	if err != nil {
		return "", "", err
	}
	return ch.Name, ch.DisplayName, nil
}

func (cmd *cmd) getChannelbyName(ctx context.Context, name string) (id int64, displayName string, err error) {
	if err := cmd.twitchThrottle(ctx); err != nil {
		return 0, "", err
	}

	ch, err := tw.GetUserForUsername(ctx, name)
	if err != nil {
		return 0, "", err
	}
	return ch.ID, ch.DisplayName, nil
}

func (cmd *cmd) twitchThrottle(ctx context.Context) error {
	last := lastCall
	lastCall = time.Now()

	d := time.Since(last)
	if d > cmd.TwitchSleep {
		return nil
	}

	d = cmd.TwitchSleep - d

	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
