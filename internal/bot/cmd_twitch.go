package bot

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
)

const twitchServerErrorReply = "A Twitch server error occurred."

func cmdStatus(ctx context.Context, s *session, cmd string, args string) error {
	if args != "" && s.UserLevel.CanAccess(levelModerator) {
		replied, err := setStatus(ctx, s, args)
		if replied || err != nil {
			return err
		}
		return s.Reply(ctx, "Status updated.")
	}

	ch, err := s.TwitchChannel(ctx)
	if err != nil {
		if err == twitch.ErrServerError {
			return s.Reply(ctx, twitchServerErrorReply)
		}
		return err
	}

	v := ch.Status
	if v == "" {
		v = "(Not set)"
	}

	return s.Reply(ctx, v)
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
			return true, s.Reply(ctx, s.TwitchNotAuthMessage())
		case twitch.ErrServerError:
			return true, s.Reply(ctx, twitchServerErrorReply)
		}
		return true, err
	}

	setStatus = strings.TrimSpace(setStatus)
	if !strings.EqualFold(status, setStatus) {
		return true, s.Reply(ctx, "Status update sent, but did not stick.")
	}

	return false, nil
}

func cmdGame(ctx context.Context, s *session, cmd string, args string) error {
	if args != "" && s.UserLevel.CanAccess(levelModerator) {
		replied, err := setGame(ctx, s, args)
		if replied || err != nil {
			return err
		}

		return s.Reply(ctx, "Game updated.")
	}

	ch, err := s.TwitchChannel(ctx)
	if err != nil {
		if err == twitch.ErrServerError {
			return s.Reply(ctx, twitchServerErrorReply)
		}
		return err
	}

	v := ch.Game
	if v == "" {
		v = "(Not set)"
	}

	return s.Reply(ctx, "Current game: "+v)
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
			return true, s.Reply(ctx, s.TwitchNotAuthMessage())
		case twitch.ErrServerError:
			return true, s.Reply(ctx, twitchServerErrorReply)
		}
		return true, err
	}

	setGame = strings.TrimSpace(setGame)
	if !strings.EqualFold(game, setGame) {
		return true, s.Reply(ctx, "Game update sent, but did not stick.")
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

	return s.Replyf(ctx, "Live for %s.", uStr)
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

	return s.Replyf(ctx, "%d %s.", viewers, vs)
}

func streamOrReplyNotLive(ctx context.Context, s *session) (*twitch.Stream, error) {
	stream, err := s.TwitchStream(ctx)

	switch err {
	case twitch.ErrServerError:
		return nil, s.Reply(ctx, twitchServerErrorReply)
	case nil:
	default:
		return nil, err
	}

	if stream == nil {
		return nil, s.Reply(ctx, "Stream is not live.")
	}

	return stream, nil
}

func cmdChatters(ctx context.Context, s *session, cmd string, args string) error {
	chatters, err := s.TwitchChatters(ctx)
	switch err {
	case twitch.ErrServerError, twitch.ErrNotFound:
		return s.Reply(ctx, twitchServerErrorReply)
	case nil:
	default:
		return err
	}

	count := chatters.Count

	u := "users"
	if count == 1 {
		u = "user"
	}

	return s.Replyf(ctx, "%d %s currently connected to chat.", count, u)
}

func cmdIsLive(ctx context.Context, s *session, cmd string, args string) error {
	name, _ := splitSpace(args)
	name = strings.ToLower(name)

	if name == "" {
		isLive, err := s.IsLive(ctx)
		switch err {
		case twitch.ErrServerError:
			return s.Reply(ctx, twitchServerErrorReply)
		case nil:
		default:
			return err
		}

		if isLive {
			return s.Replyf(ctx, "Yes, %s is live.", s.Channel.Name)
		}

		return s.Replyf(ctx, "No, %s isn't live.", s.Channel.Name)
	}

	u, err := s.Deps.Twitch.GetUserForUsername(ctx, name)
	if err != nil {
		switch err {
		case twitch.ErrNotFound:
			return s.Replyf(ctx, "User %s does not exist.", name)
		case twitch.ErrServerError:
			return s.Reply(ctx, twitchServerErrorReply)
		}
		return err
	}

	stream, err := s.Deps.Twitch.GetCurrentStream(ctx, u.ID)
	if err != nil {
		switch err {
		case twitch.ErrServerError:
			return s.Reply(ctx, twitchServerErrorReply)
		case nil:
		default:
			return err
		}
	}

	if stream == nil {
		return s.Replyf(ctx, "No, %s isn't live.", name)
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

	return s.Replyf(ctx, "Yes, %s is live playing %s with %d %s.", name, game, viewers, v)
}

func cmdIsHere(ctx context.Context, s *session, cmd string, args string) error {
	name, _ := splitSpace(args)

	if name == "" {
		return s.ReplyUsage(ctx, "<username>")
	}

	chatters, err := s.TwitchChatters(ctx)
	switch err {
	case twitch.ErrServerError, twitch.ErrNotFound:
		return s.Reply(ctx, twitchServerErrorReply)
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
			return s.Replyf(ctx, "Yes, %s is connected to chat.", name)
		}
	}

	return s.Replyf(ctx, "No, %s is not connected to chat.", name)
}

func cmdHost(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<username>")
	}

	username, _ := splitSpace(args)

	if err := s.SendCommand(ctx, "host", strings.ToLower(username)); err != nil {
		return err
	}

	return s.Replyf(ctx, "Now hosting: %s", username)
}

func cmdUnhost(ctx context.Context, s *session, cmd string, args string) error {
	if err := s.SendCommand(ctx, "unhost"); err != nil {
		return err
	}

	return s.Reply(ctx, "Exited host mode.")
}

func cmdWinner(ctx context.Context, s *session, cmd string, args string) error {
	chatters, err := s.TwitchChatters(ctx)
	switch err {
	case twitch.ErrServerError, twitch.ErrNotFound:
		return s.Reply(ctx, twitchServerErrorReply)
	case nil:
	default:
		return err
	}

	lists := [][]string{
		chatters.Chatters.Vips,
		chatters.Chatters.Moderators,
		chatters.Chatters.Staff,
		chatters.Chatters.Admins,
		chatters.Chatters.GlobalMods,
		chatters.Chatters.Viewers,
	}

	count := 0
	for _, l := range lists {
		count += len(l)
	}

	if count == 0 {
		return s.Reply(ctx, "Nobody in chat.")
	}

	i := s.Deps.Rand.Intn(count)

	for _, l := range lists {
		if i < len(l) {
			return s.Reply(ctx, "And the winner is... "+l[i]+"!")
		}

		i -= len(l)
	}

	panic("unreachable")
}

const followAuthError = "Could not get authorization to follow your channel; please contact an admin."

func cmdFollowMe(ctx context.Context, s *session, cmd string, args string) error {
	tt, err := models.TwitchTokens(models.TwitchTokenWhere.BotName.EQ(null.StringFrom(s.Channel.BotName))).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			return s.Reply(ctx, followAuthError)
		}
		return err
	}

	tok := modelsx.ModelToToken(tt)

	newToken, err := s.Deps.Twitch.FollowChannel(ctx, tt.TwitchID, tok, s.RoomID)
	if err != nil {
		switch err {
		case twitch.ErrServerError:
			return s.Reply(ctx, twitchServerErrorReply)
		case twitch.ErrNotAuthorized:
			return s.Reply(ctx, followAuthError)
		case nil:
		default:
			return err
		}
	}

	if newToken != nil {
		tt.AccessToken = newToken.AccessToken
		tt.TokenType = newToken.TokenType
		tt.RefreshToken = newToken.RefreshToken
		tt.Expiry = newToken.Expiry

		if err := tt.Update(ctx, s.Tx, boil.Blacklist()); err != nil {
			return err
		}
	}

	return s.Reply(ctx, "Follow update sent.")
}
