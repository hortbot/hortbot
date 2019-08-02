package bot

import (
	"context"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
)

const (
	notAuthorizedReply = "The bot wasn't authorized. Head over to the website to give the bot permission." // TODO: provide link
	serverErrorReply   = "A Twitch server error occurred."
)

func cmdStatus(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.Twitch == nil {
		return errBuiltinDisabled
	}

	if args != "" && s.UserLevel.CanAccess(levelModerator) {
		tok, err := s.TwitchToken(ctx)
		if err != nil {
			return err
		}

		if args == "-" {
			args = ""
		}

		setStatus, newToken, err := s.Deps.Twitch.SetChannelStatus(ctx, s.Channel.UserID, tok, args)

		// Check this, even if an error occurred.
		if newToken != nil {
			if err := s.SetTwitchToken(ctx, newToken); err != nil {
				return err
			}
		}

		if err != nil {
			switch err {
			case twitch.ErrNotAuthorized:
				return s.Reply(notAuthorizedReply)
			case twitch.ErrServerError:
				return s.Reply(serverErrorReply)
			}
			return err
		}

		setStatus = strings.TrimSpace(setStatus)
		if !strings.EqualFold(args, setStatus) {
			return s.Reply("Status update sent, but did not stick.")
		}

		return s.Reply("Status updated.")
	}

	ch, err := s.Deps.Twitch.GetChannelByID(ctx, s.Channel.UserID)
	if err != nil {
		if err == twitch.ErrServerError {
			return s.Reply(serverErrorReply)
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

	if args != "" && s.UserLevel.CanAccess(levelModerator) {
		tok, err := s.TwitchToken(ctx)
		if err != nil {
			return err
		}

		if args == "-" {
			args = ""
		}

		setGame, newToken, err := s.Deps.Twitch.SetChannelGame(ctx, s.Channel.UserID, tok, args)

		// Check this, even if an error occurred.
		if newToken != nil {
			if err := s.SetTwitchToken(ctx, newToken); err != nil {
				return err
			}
		}

		if err != nil {
			switch err {
			case twitch.ErrNotAuthorized:
				return s.Reply(notAuthorizedReply)
			case twitch.ErrServerError:
				return s.Reply(serverErrorReply)
			}
			return err
		}

		setGame = strings.TrimSpace(setGame)
		if !strings.EqualFold(args, setGame) {
			return s.Reply("Game update sent, but did not stick.")
		}

		return s.Reply("Game updated.")
	}

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

func cmdUptime(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.Twitch == nil {
		return errBuiltinDisabled
	}

	stream, err := streamOrReplyNotLive(ctx, s)
	if err != nil || stream == nil {
		return err
	}

	uptime := s.Deps.Clock.Since(stream.CreatedAt).Round(time.Minute)
	uStr := durafmt.Parse(uptime).String()

	return s.Replyf("Live for %s.", uStr)
}

func cmdViewers(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.Twitch == nil {
		return errBuiltinDisabled
	}

	stream, err := streamOrReplyNotLive(ctx, s)
	if err != nil || stream == nil {
		return err
	}

	viewers := stream.Viewers

	vs := "viewers"
	if viewers == 1 {
		vs = "viewer"
	}

	return s.Replyf("%d %s.", viewers, vs)
}

func streamOrReplyNotLive(ctx context.Context, s *session) (*twitch.Stream, error) {
	stream, err := s.Deps.Twitch.GetCurrentStream(ctx, s.Channel.UserID)

	switch err {
	case twitch.ErrServerError:
		return nil, s.Reply(serverErrorReply)
	case nil:
	default:
		return nil, err
	}

	if stream == nil {
		return nil, s.Reply("Stream is not live.")
	}

	return stream, nil
}

func cmdChatters(ctx context.Context, s *session, cmd string, args string) error {
	if s.Deps.Twitch == nil {
		return errBuiltinDisabled
	}

	chatters, err := s.Deps.Twitch.GetChatters(ctx, s.Channel.Name)
	switch err {
	case twitch.ErrServerError, twitch.ErrNotFound:
		return s.Reply(serverErrorReply)
	case nil:
	default:
		return err
	}

	u := "users"
	if chatters == 1 {
		u = "user"
	}

	return s.Replyf("%d %s currently connected to chat.", chatters, u)
}
