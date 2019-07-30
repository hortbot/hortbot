package bot

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
)

func cmdStatus(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.Twitch == nil {
		return errBuiltinDisabled
	}

	// TODO: Set status if args != "" and mod+

	ch, err := s.Deps.Twitch.GetChannelByID(ctx, s.Channel.UserID)
	if err != nil {
		if err == twitch.ErrServerError {
			return s.Reply("A Twitch server error occurred.")
		}
		// Any other type of error is the bot's fault.
		// TODO: Reply?
		return err
	}

	v := ch.Status
	if v == "" {
		v = "(Not set)"
	}

	return s.Reply(v)
}

func cmdGame(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.Twitch == nil {
		return errBuiltinDisabled
	}

	// TODO: Set status if args != "" and mod+

	ch, err := s.Deps.Twitch.GetChannelByID(ctx, s.Channel.UserID)
	if err != nil {
		if err == twitch.ErrServerError {
			return s.Reply("A Twitch server error occurred.")
		}
		// Any other type of error is the bot's fault.
		// TODO: Reply?
		return err
	}

	v := ch.Game
	if v == "" {
		v = "(Not set)"
	}

	return s.Reply("Current game: " + v)
}
