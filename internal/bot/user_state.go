package bot

import (
	"context"
	"time"

	"github.com/jakebailey/irc"
	"go.opencensus.io/trace"
)

func (b *Bot) handleUserState(ctx context.Context, origin string, m *irc.Message) error {
	ctx, span := trace.StartSpan(ctx, "handleUserState")
	defer span.End()

	if len(m.Tags) == 0 {
		return errInvalidMessage
	}

	if len(m.Params) == 0 {
		return errInvalidMessage
	}

	channel := m.Params[0]
	if len(channel) <= 1 {
		return errInvalidMessage
	}

	fast := true
	switch {
	case m.Tags["mod"] == "1":
	case m.Tags["user-type"] == "mod":
	case origin == channel[1:]:
	default:
		fast = false
	}

	if !fast {
		fast = true
		badges := parseBadges(m.Tags["badges"])

		switch {
		case badges["broadcaster"] != "":
		case badges["moderator"] != "":
		case badges["vip"] != "":
		default:
			fast = false
		}
	}

	return b.deps.Redis.SetUserState(ctx, origin, channel, fast, 24*time.Hour)
}
