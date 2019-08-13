package bot

import (
	"context"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
)

const (
	twitchNotAuthorizedReply = "The bot wasn't authorized. Head over to the website to give the bot permission." // TODO: provide link
	twitchServerErrorReply   = "A Twitch server error occurred."
)

func cmdStatus(ctx context.Context, s *session, cmd string, args string) error {
	if args != "" && s.UserLevel.CanAccess(levelModerator) {
		replied, err := setStatus(ctx, s, args)
		if replied || err != nil {
			return err
		}
		return s.Reply("Status updated.")
	}

	ch, err := s.TwitchChannel(ctx)
	if err != nil {
		if err == twitch.ErrServerError {
			return s.Reply(twitchServerErrorReply)
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

func setStatus(ctx context.Context, s *session, status string) (bool, error) {
	tok, err := s.TwitchToken(ctx)
	if err != nil {
		return true, err
	}

	if status == "-" {
		status = ""
	}

	setStatus, newToken, err := s.Deps.Twitch.SetChannelStatus(ctx, s.Channel.UserID, tok, status)

	// Check this, even if an error occurred.
	if newToken != nil {
		if err := s.SetTwitchToken(ctx, newToken); err != nil {
			return true, err
		}
	}

	if err != nil {
		switch err {
		case twitch.ErrNotAuthorized:
			return true, s.Reply(twitchNotAuthorizedReply)
		case twitch.ErrServerError:
			return true, s.Reply(twitchServerErrorReply)
		}
		return true, err
	}

	setStatus = strings.TrimSpace(setStatus)
	if !strings.EqualFold(status, setStatus) {
		return true, s.Reply("Status update sent, but did not stick.")
	}

	return false, nil
}

func cmdGame(ctx context.Context, s *session, cmd string, args string) error {
	if args != "" && s.UserLevel.CanAccess(levelModerator) {
		replied, err := setGame(ctx, s, args)
		if replied || err != nil {
			return err
		}

		return s.Reply("Game updated.")
	}

	ch, err := s.TwitchChannel(ctx)
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

func setGame(ctx context.Context, s *session, game string) (bool, error) {
	tok, err := s.TwitchToken(ctx)
	if err != nil {
		return true, err
	}

	if game == "-" {
		game = ""
	}

	setGame, newToken, err := s.Deps.Twitch.SetChannelGame(ctx, s.Channel.UserID, tok, game)

	// Check this, even if an error occurred.
	if newToken != nil {
		if err := s.SetTwitchToken(ctx, newToken); err != nil {
			return true, err
		}
	}

	if err != nil {
		switch err {
		case twitch.ErrNotAuthorized:
			return true, s.Reply(twitchNotAuthorizedReply)
		case twitch.ErrServerError:
			return true, s.Reply(twitchServerErrorReply)
		}
		return true, err
	}

	setGame = strings.TrimSpace(setGame)
	if !strings.EqualFold(game, setGame) {
		return true, s.Reply("Game update sent, but did not stick.")
	}

	return false, nil
}

func cmdUptime(ctx context.Context, s *session, cmd string, args string) error {
	stream, err := streamOrReplyNotLive(ctx, s)
	if err != nil || stream == nil {
		return err
	}

	uptime := s.Deps.Clock.Since(stream.CreatedAt).Round(time.Minute)
	uStr := durafmt.Parse(uptime).String()

	return s.Replyf("Live for %s.", uStr)
}

func cmdViewers(ctx context.Context, s *session, cmd string, args string) error {
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
	stream, err := s.TwitchStream(ctx)

	switch err {
	case twitch.ErrServerError:
		return nil, s.Reply(twitchServerErrorReply)
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
	chatters, err := s.TwitchChatters(ctx)
	switch err {
	case twitch.ErrServerError, twitch.ErrNotFound:
		return s.Reply(twitchServerErrorReply)
	case nil:
	default:
		return err
	}

	count := chatters.Count

	u := "users"
	if count == 1 {
		u = "user"
	}

	return s.Replyf("%d %s currently connected to chat.", count, u)
}

func cmdIsLive(ctx context.Context, s *session, cmd string, args string) error {
	name, _ := splitSpace(args)
	name = strings.ToLower(name)

	if name == "" {
		isLive, err := s.IsLive(ctx)
		switch err {
		case twitch.ErrServerError:
			return s.Reply(twitchServerErrorReply)
		case nil:
		default:
			return err
		}

		if isLive {
			return s.Replyf("Yes, %s is live.", s.Channel.Name)
		}

		return s.Replyf("No, %s isn't live.", s.Channel.Name)
	}

	id, err := s.Deps.Twitch.GetIDForUsername(ctx, name)
	if err != nil {
		switch err {
		case twitch.ErrNotFound:
			return s.Replyf("User %s does not exist.", name)
		case twitch.ErrServerError:
			return s.Reply(twitchServerErrorReply)
		}
		return err
	}

	stream, err := s.Deps.Twitch.GetCurrentStream(ctx, id)
	if err != nil {
		switch err {
		case twitch.ErrServerError:
			return s.Reply(twitchServerErrorReply)
		case nil:
		default:
			return err
		}
	}

	if stream == nil {
		return s.Replyf("No, %s isn't live.", name)
	}

	viewers := stream.Viewers

	v := "viewers"
	if viewers == 1 {
		v = "viewer"
	}

	game := stream.Game
	if game == "" {
		game = "(Not set)"
	}

	return s.Replyf("Yes, %s is live playing %s with %d %s.", name, game, viewers, v)
}

func cmdIsHere(ctx context.Context, s *session, cmd string, args string) error {
	name, _ := splitSpace(args)

	if name == "" {
		return s.ReplyUsage("<username>")
	}

	chatters, err := s.TwitchChatters(ctx)
	switch err {
	case twitch.ErrServerError, twitch.ErrNotFound:
		return s.Reply(twitchServerErrorReply)
	case nil:
	default:
		return err
	}

	lists := [][]string{
		chatters.Chatters.Broadcaster,
		chatters.Chatters.Vips,
		chatters.Chatters.Moderators,
		chatters.Chatters.Staff,
		chatters.Chatters.Admins,
		chatters.Chatters.GlobalMods,
		chatters.Chatters.Viewers,
	}

	nameLower := strings.ToLower(name)

	for _, l := range lists {
		if _, found := stringSliceIndex(l, nameLower); found {
			return s.Replyf("Yes, %s is connected to chat.", name)
		}
	}

	return s.Replyf("No, %s is not connected to chat.", name)
}

func cmdHost(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage("<username>")
	}

	username, _ := splitSpace(args)

	if err := s.SendCommand("host", strings.ToLower(username)); err != nil {
		return err
	}

	return s.Replyf("Now hosting: %s", username)
}

func cmdUnhost(ctx context.Context, s *session, cmd string, args string) error {
	if err := s.SendCommand("unhost"); err != nil {
		return err
	}

	return s.Reply("Exited host mode.")
}
