package bot

import (
	"context"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"go.deanishe.net/fuzzy"
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

	v := ch.Title
	if v == "" {
		v = "(Not set)"
	}

	return s.Reply(ctx, v)
}

func setStatus(ctx context.Context, s *session, status string) (replied bool, err error) {
	tok, err := s.ChannelTwitchToken(ctx)
	if err != nil {
		return true, err
	}

	if status == "-" {
		status = ""
	}

	if status == "" {
		return true, s.Reply(ctx, "Statuses cannot be unset.")
	}

	newToken, err := s.Deps.Twitch.ModifyChannel(ctx, s.Channel.TwitchID, tok, &status, nil)

	// Check this, even if an error occurred.
	if newToken != nil {
		if err := s.SetChannelTwitchToken(ctx, newToken); err != nil {
			return true, err
		}
	}

	if err != nil {
		switch err {
		case twitch.ErrNotAuthorized, twitch.ErrDeadToken: // TODO: Delete dead token.
			return true, s.Reply(ctx, s.TwitchNotAuthMessage())
		case twitch.ErrServerError:
			return true, s.Reply(ctx, twitchServerErrorReply)
		}
		return true, err
	}

	return false, nil
}

func cmdGame(ctx context.Context, s *session, cmd string, args string) error {
	if args != "" && s.UserLevel.CanAccess(levelModerator) {
		_, err := setGame(ctx, s, args)
		return err
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

// TODO: This is a dupe of the above code.
func setGameAndStatus(ctx context.Context, s *session, game string, status string) (ok bool, err error) {
	tok, err := s.ChannelTwitchToken(ctx)
	if err != nil {
		return false, err
	}

	if game == "-" {
		game = ""
	}

	if status == "-" {
		status = ""
	}

	if status == "" {
		return true, s.Reply(ctx, "Statuses cannot be unset.")
	}

	var gameID int64

	if game != "" {
		found, err := fixGameOrSuggest(ctx, s, game)
		if err != nil || found == nil {
			return false, err
		}
		gameID = found.ID.AsInt64()
	}

	newToken, err := s.Deps.Twitch.ModifyChannel(ctx, s.Channel.TwitchID, tok, &status, &gameID)

	// Check this, even if an error occurred.
	if newToken != nil {
		if err := s.SetChannelTwitchToken(ctx, newToken); err != nil {
			return false, err
		}
	}

	if err != nil {
		switch err {
		case twitch.ErrNotAuthorized, twitch.ErrDeadToken: // TODO: Delete dead token.
			return false, s.Reply(ctx, s.TwitchNotAuthMessage())
		case twitch.ErrServerError:
			return false, s.Reply(ctx, twitchServerErrorReply)
		}
		return false, err
	}

	return true, nil
}

func setGame(ctx context.Context, s *session, game string) (ok bool, err error) {
	tok, err := s.ChannelTwitchToken(ctx)
	if err != nil {
		return false, err
	}

	if game == "-" {
		game = ""
	}

	var gameID int64
	var gameName string

	if game != "" {
		found, err := fixGameOrSuggest(ctx, s, game)
		if err != nil || found == nil {
			return false, err
		}
		gameID = found.ID.AsInt64()
		gameName = found.Name
	}

	newToken, err := s.Deps.Twitch.ModifyChannel(ctx, s.Channel.TwitchID, tok, nil, &gameID)

	// Check this, even if an error occurred.
	if newToken != nil {
		if err := s.SetChannelTwitchToken(ctx, newToken); err != nil {
			return false, err
		}
	}

	if err != nil {
		switch err {
		case twitch.ErrNotAuthorized, twitch.ErrDeadToken: // TODO: Delete dead token.
			return false, s.Reply(ctx, s.TwitchNotAuthMessage())
		case twitch.ErrServerError:
			return false, s.Reply(ctx, twitchServerErrorReply)
		}
		return false, err
	}

	if game == "" {
		return true, s.Reply(ctx, "Game unset.")
	}

	return true, s.Replyf(ctx, "Game updated to: %s", gameName)
}

func fixGameOrSuggest(ctx context.Context, s *session, game string) (*twitch.Category, error) {
	exact, suggestions, err := searchGame(ctx, s, game)
	if err != nil {
		if err == twitch.ErrServerError {
			return nil, s.Reply(ctx, twitchServerErrorReply)
		}
		return nil, err
	}

	if exact != nil {
		return exact, nil
	}

	if suggestions[0] == nil {
		return nil, s.Replyf(ctx, `Could not find a valid game matching "%s".`, game)
	}

	if suggestions[1] != nil {
		return nil, s.Replyf(ctx, `Could not find a valid game matching "%s". Did you mean "%s" or "%s"?`, game, suggestions[0].Name, suggestions[1].Name)
	}

	return nil, s.Replyf(ctx, `Could not find a valid game matching "%s". Did you mean "%s"?`, game, suggestions[0].Name)
}

type gameSuggestion [2]*twitch.Category

func searchGame(ctx context.Context, s *session, name string) (exact *twitch.Category, suggestions gameSuggestion, err error) {
	{
		g, err := s.Deps.Twitch.GetGameByName(ctx, name)
		switch err {
		case nil:
			return g, gameSuggestion{}, nil
		case twitch.ErrNotFound:
			// Do nothing.
		default:
			return nil, gameSuggestion{}, err
		}
	}

	// Game name did not match exactly; search.

	gs, err := s.Deps.Twitch.SearchCategories(ctx, name)
	if err != nil {
		return nil, gameSuggestion{}, err
	}

	if len(gs) == 0 {
		return nil, gameSuggestion{}, nil
	}

	for _, g := range gs {
		eq := strings.EqualFold(name, g.Name)
		if eq {
			return g, gameSuggestion{}, nil
		}
	}

	first := gs[0]

	fuzzy.Sort(sortableCategories(gs), name)
	closest := gs[0]

	if first == closest {
		return nil, gameSuggestion{first}, nil
	}

	return nil, gameSuggestion{closest, first}, nil
}

type sortableCategories []*twitch.Category

func (s sortableCategories) Len() int {
	return len(s)
}

func (s sortableCategories) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

func (s sortableCategories) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortableCategories) Keywords(i int) string {
	return s[i].Name
}

func cmdUptime(ctx context.Context, s *session, cmd string, args string) error {
	stream, err := streamOrReplyNotLive(ctx, s)
	if err != nil || stream == nil {
		return err
	}

	uptime := s.Deps.Clock.Since(stream.StartedAt).Truncate(time.Minute)
	uStr := durafmt.Parse(uptime).String()

	return s.Replyf(ctx, "Live for %s.", uStr)
}

func cmdViewers(ctx context.Context, s *session, cmd string, args string) error {
	stream, err := streamOrReplyNotLive(ctx, s)
	if err != nil || stream == nil {
		return err
	}

	viewers := stream.ViewerCount

	vs := "viewers"
	if viewers == 1 {
		vs = "viewer"
	}

	return s.Replyf(ctx, "%d %s.", viewers, vs)
}

func streamOrReplyNotLive(ctx context.Context, s *session) (*twitch.Stream, error) {
	stream, err := s.TwitchStream(ctx)

	switch err {
	case nil:
		return stream, nil
	case twitch.ErrNotFound:
		return nil, s.Reply(ctx, "Stream is not live.")
	case twitch.ErrServerError:
		return nil, s.Reply(ctx, twitchServerErrorReply)
	default:
		return nil, err
	}
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

	u, err := s.Deps.Twitch.GetUserByUsername(ctx, name)
	if err != nil {
		switch err {
		case twitch.ErrNotFound:
			return s.Replyf(ctx, "User %s does not exist.", name)
		case twitch.ErrServerError:
			return s.Reply(ctx, twitchServerErrorReply)
		}
		return err
	}

	stream, err := s.Deps.Twitch.GetStreamByUserID(ctx, u.ID.AsInt64())
	if err != nil {
		switch err {
		case twitch.ErrNotFound:
			return s.Replyf(ctx, "No, %s isn't live.", name)
		case twitch.ErrServerError:
			return s.Reply(ctx, twitchServerErrorReply)
		case nil:
		default:
			return err
		}
	}

	viewers := stream.ViewerCount

	v := "viewers"
	if viewers == 1 {
		v = "viewer"
	}

	var gameName string
	if stream.GameID == 0 {
		gameName = "(Not set)"
	} else {
		game, err := s.Deps.Twitch.GetGameByID(ctx, stream.GameID.AsInt64())
		if err != nil {
			return err
		}
		gameName = game.Name
	}

	return s.Replyf(ctx, "Yes, %s is live playing %s with %d %s.", name, gameName, viewers, v)
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
