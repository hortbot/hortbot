package bot

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
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

	v := ch.Status
	if v == "" {
		v = "(Not set)"
	}

	return s.Reply(ctx, v)
}

func setStatus(ctx context.Context, s *session, status string) (replied bool, err error) {
	if s.BetaFeatures() {
		return setStatusHelix(ctx, s, status)
	}

	tok, err := s.TwitchToken(ctx)
	if err != nil {
		return true, err
	}

	if status == "-" {
		status = ""
	}

	setStatus, newToken, err := s.Deps.Twitch.SetChannelStatus(ctx, s.Channel.TwitchID, tok, status)

	// Check this, even if an error occurred.
	if newToken != nil {
		if err := s.SetTwitchToken(ctx, newToken); err != nil {
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

	setStatus = strings.TrimSpace(setStatus)
	if !strings.EqualFold(status, setStatus) {
		return true, s.Reply(ctx, "Status update sent, but did not stick.")
	}

	return false, nil
}

func setStatusHelix(ctx context.Context, s *session, status string) (replied bool, err error) {
	tok, err := s.TwitchToken(ctx)
	if err != nil {
		return true, err
	}

	if status == "-" {
		status = ""
	}

	if status == "" {
		return true, s.Reply(ctx, "Cannot unset status.")
	}

	newToken, err := s.Deps.Twitch.ModifyChannel(ctx, s.Channel.TwitchID, tok, status, 0)

	// Check this, even if an error occurred.
	if newToken != nil {
		if err := s.SetTwitchToken(ctx, newToken); err != nil {
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
		_, err := setGame(ctx, s, args, true)
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

func setGame(ctx context.Context, s *session, game string, replyOnSuccess bool) (ok bool, err error) {
	if s.BetaFeatures() {
		return setGameHelix(ctx, s, game, replyOnSuccess)
	}

	tok, err := s.TwitchToken(ctx)
	if err != nil {
		return false, err
	}

	if game == "-" {
		game = ""
	}

	if game != "" {
		found, err := fixGameOrSuggest(ctx, s, game)
		if err != nil || found == nil {
			return false, err
		}
		game = found.Name
	}

	setGame, newToken, err := s.Deps.Twitch.SetChannelGame(ctx, s.Channel.TwitchID, tok, game)

	// Check this, even if an error occurred.
	if newToken != nil {
		if err := s.SetTwitchToken(ctx, newToken); err != nil {
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

	setGame = strings.TrimSpace(setGame)
	if !strings.EqualFold(game, setGame) {
		if err := s.Reply(ctx, "Game update sent, but did not stick."); err != nil {
			return false, err
		}
		return false, nil
	}

	if !replyOnSuccess {
		return true, nil
	}

	if game == "" {
		return true, s.Reply(ctx, "Game unset.")
	}

	return true, s.Replyf(ctx, "Game updated to: %s", game)
}

func setGameHelix(ctx context.Context, s *session, game string, replyOnSuccess bool) (ok bool, err error) {
	tok, err := s.TwitchToken(ctx)
	if err != nil {
		return false, err
	}

	if game == "-" {
		game = ""
	}

	if game == "" {
		return false, s.Reply(ctx, "Cannot unset game.")
	}

	found, err := fixGameOrSuggest(ctx, s, game)
	if err != nil || found == nil {
		return false, err
	}

	// NOTE: This is broken on Twitch's end, as updates using this API do not propagate correctly.
	// For example, any queries to Kraken will still return the old game, clips and highlights will
	// show the wrong game, and so on, but the channel page and dashboard will be correct.
	newToken, err := s.Deps.Twitch.ModifyChannel(ctx, s.Channel.TwitchID, tok, "", found.ID.AsInt64())

	// Check this, even if an error occurred.
	if newToken != nil {
		if err := s.SetTwitchToken(ctx, newToken); err != nil {
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

	if !replyOnSuccess {
		return true, nil
	}

	return true, s.Replyf(ctx, "Game updated to: %s", found.Name)
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

	uptime := s.Deps.Clock.Since(stream.CreatedAt).Truncate(time.Minute)
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

	stream, err := s.Deps.Twitch.GetCurrentStream(ctx, u.ID.AsInt64())
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
	err := followUser(ctx, s.Tx, s.Deps.Twitch, s.Channel.BotName, s.RoomID)

	switch err {
	case twitch.ErrServerError:
		return s.Reply(ctx, twitchServerErrorReply)
	case sql.ErrNoRows, twitch.ErrNotAuthorized:
		return s.Reply(ctx, followAuthError)
	case nil:
		return s.Reply(ctx, "Follow update sent.")
	default:
		return err
	}
}

func followUser(ctx context.Context, db boil.ContextExecutor, tw twitch.API, botName string, userID int64) error {
	tt, err := models.TwitchTokens(models.TwitchTokenWhere.BotName.EQ(null.StringFrom(botName))).One(ctx, db)
	if err != nil {
		return err
	}

	tok := modelsx.ModelToToken(tt)

	newToken, err := tw.FollowChannel(ctx, tt.TwitchID, tok, userID)
	if err != nil {
		return err
	}

	if newToken != nil {
		tt.AccessToken = newToken.AccessToken
		tt.TokenType = newToken.TokenType
		tt.RefreshToken = newToken.RefreshToken
		tt.Expiry = newToken.Expiry

		if err := tt.Update(ctx, db, boil.Blacklist()); err != nil {
			return err
		}
	}

	return nil
}
