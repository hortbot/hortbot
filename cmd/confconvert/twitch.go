package main

import (
	"context"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
)

var (
	tw       *twitch.Twitch
	lastCall time.Time
)

func getChannelByID(ctx context.Context, id int64) (name string, displayName string, err error) {
	if err := twitchThrottle(ctx); err != nil {
		return "", "", err
	}

	ch, err := tw.GetChannelByID(ctx, id)
	if err != nil {
		return "", "", err
	}
	return ch.Name, ch.DisplayName, nil
}

func getChannelbyName(ctx context.Context, name string) (id int64, displayName string, err error) {
	if err := twitchThrottle(ctx); err != nil {
		return 0, "", err
	}

	ch, err := tw.GetUserForUsername(ctx, name)
	if err != nil {
		return 0, "", err
	}
	return ch.ID, ch.DisplayName, nil
}

func twitchThrottle(ctx context.Context) error {
	last := lastCall
	lastCall = time.Now()

	d := time.Since(last)
	if d > args.TwitchSleep {
		return nil
	}

	d = args.TwitchSleep - d

	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
