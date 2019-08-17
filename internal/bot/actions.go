package bot

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apis/extralife"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

var testingAction func(ctx context.Context, action string) (string, error, bool)

//nolint:gocyclo
func (s *session) doAction(ctx context.Context, action string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	if isTesting && testingAction != nil {
		s, err, ok := testingAction(ctx, action)
		if ok {
			return s, err
		}
	}

	// Exact matches
	switch action {
	case "PARAMETER":
		return s.NextParameter(), nil
	case "PARAMETER_CAPS":
		return strings.ToUpper(s.NextParameter()), nil
	case "MESSAGE_COUNT":
		return strconv.FormatInt(s.N, 10), nil
	case "SONG":
		return s.actionSong(0, false)
	case "SONG_URL":
		return s.actionSong(0, true)
	case "LAST_SONG":
		return s.actionSong(1, false)
	case "QUOTE":
		q, err := getRandomQuote(ctx, s.Tx, s.Channel)
		if err != nil {
			return "", err
		}

		if q == nil {
			return "No quotes.", nil
		}

		return q.Quote, nil
	case "USER":
		return s.User, nil
	case "USER_DISPLAY":
		return s.UserDisplay, nil
	case "CHANNEL_URL":
		return "twitch.tv/" + s.Channel.Name, nil
	case "SUBMODE_ON":
		if s.UserLevel.CanAccess(levelModerator) {
			return "", s.SendCommand("subscribers")
		}
		return "", nil
	case "SUBMODE_OFF":
		if s.UserLevel.CanAccess(levelModerator) {
			return "", s.SendCommand("subscribersoff")
		}
		return "", nil
	case "SILENT":
		// TODO: handle s.Silent elsewhere.
		s.Silent = true
		return "", nil
	case "NUMCHANNELS":
		count, err := models.Channels(models.ChannelWhere.Active.EQ(true)).Count(ctx, s.Tx)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(count, 10), nil
	case "UNHOST":
		// TODO: check user level
		return "", s.SendCommand("unhost")
	case "PURGE":
		u, do := s.UserForModAction()
		if do && u != "" {
			return u, s.SendCommand("timeout", strings.ToLower(u), "1")
		}
		return u, nil
	case "TIMEOUT":
		u, do := s.UserForModAction()
		if do && u != "" {
			return u, s.SendCommand("timeout", strings.ToLower(u))
		}
		return u, nil
	case "BAN":
		u, do := s.UserForModAction()
		if do && u != "" {
			return u, s.SendCommand("ban", strings.ToLower(u))
		}
		return u, nil
	case "DELETE":
		if s.Type == sessionAutoreply {
			return s.User, s.DeleteMessage()
		}
		return "", nil
	case "REGULARS_ONLY":
		return "", nil
	case "EXTRALIFE_AMOUNT":
		if s.Deps.ExtraLife == nil || s.Channel.ExtraLifeID == 0 {
			return "?", nil
		}

		amount, err := s.Deps.ExtraLife.GetDonationAmount(s.Channel.ExtraLifeID)
		switch err {
		case nil:
			return fmt.Sprintf("$%.2f", amount), nil
		case extralife.ErrNotFound:
			return "Bad Extra Life participant ID", nil
		case extralife.ErrServerError:
			return "Extra Life server error", nil
		default:
			return "", err
		}
	case "ONLINE_CHECK":
		isLive, err := s.IsLive(ctx)
		if err != nil {
			return "", err
		}

		if !isLive {
			panic("ONLINE_CHECK should have been handled earlier")
		}

		return "", nil
	case "GAME", "GAME_CLEAN":
		var game string

		ch, err := s.TwitchChannel(ctx)
		if err != nil {
			game = "(error)"
		} else {
			game = ch.Game
		}

		if game == "" {
			game = "(Not set)"
		}

		if action == "GAME_CLEAN" {
			game = strings.Map(func(r rune) rune {
				switch {
				case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', '0' <= r && r <= '9':
					return r
				default:
					return '-'
				}
			}, game)
		}

		return game, nil
	case "STATUS":
		ch, err := s.TwitchChannel(ctx)
		if err != nil {
			return "(error)", nil
		}

		status := ch.Status

		if status == "" {
			return "(Not set)", nil
		}

		return status, nil
	case "VIEWERS":
		var viewers int64
		stream, err := s.TwitchStream(ctx)
		if err == nil && stream != nil {
			viewers = stream.Viewers
		}
		return strconv.FormatInt(viewers, 10), nil
	case "CHATTERS":
		chatters, _ := s.TwitchChatters(ctx)
		var count int64
		if chatters != nil {
			count = chatters.Count
		}
		return strconv.FormatInt(count, 10), nil
	case "DATE":
		return s.actionTime(ctx, "", "Jan 2, 2006")
	case "TIME":
		return s.actionTime(ctx, "", "3:04 PM")
	case "TIME24":
		return s.actionTime(ctx, "", "15:04")
	case "DATETIME":
		return s.actionTime(ctx, "", "Jan 2, 2006 3:04 PM")
	case "DATETIME24":
		return s.actionTime(ctx, "", "Jan 2, 2006 15:04")
	case "STEAM_PROFILE":
		summary, err := s.SteamSummary(ctx)
		if err != nil {
			return "(error)", nil
		}

		url := summary.ProfileURL
		if url == "" {
			return "(unavailable)", nil
		}
		return url, nil
	case "STEAM_GAME":
		var game string
		summary, err := s.SteamSummary(ctx)
		if err == nil {
			game = summary.Game
		}

		if game == "" || err != nil {
			ch, err := s.TwitchChannel(ctx)
			if err != nil {
				game = "(error)"
			} else {
				game = ch.Game
			}
		}

		if game == "" {
			game = "(unavailable)"
		}

		return game, nil
	case "STEAM_SERVER":
		summary, err := s.SteamSummary(ctx)
		if err != nil {
			return "(error)", nil
		}

		server := summary.GameServer
		if server == "" {
			return "(unavailable)", nil
		}
		return server, nil
	case "STEAM_STORE":
		summary, err := s.SteamSummary(ctx)
		if err != nil {
			return "(error)", nil
		}

		gameID := summary.GameID
		if gameID == "" {
			return "(unavailable)", nil
		}
		return "http://store.steampowered.com/app/" + gameID, nil
	case "TWEET_URL":
		return s.actionTweet(ctx)
	}

	switch {
	case strings.HasPrefix(action, "DATE_"):
		tz := strings.TrimPrefix(action, "DATE_")
		return s.actionTime(ctx, tz, "Jan 2, 2006")
	case strings.HasPrefix(action, "TIME_"):
		tz := strings.TrimPrefix(action, "TIME_")
		return s.actionTime(ctx, tz, "3:04 PM")
	case strings.HasPrefix(action, "TIME24_"):
		tz := strings.TrimPrefix(action, "TIME24_")
		return s.actionTime(ctx, tz, "15:04")
	case strings.HasPrefix(action, "DATETIME_"):
		tz := strings.TrimPrefix(action, "DATETIME_")
		return s.actionTime(ctx, tz, "Jan 2, 2006 3:04 PM")
	case strings.HasPrefix(action, "DATETIME24_"):
		tz := strings.TrimPrefix(action, "DATETIME24_")
		return s.actionTime(ctx, tz, "Jan 2, 2006 15:04")

	case strings.HasPrefix(action, "UNTIL_"):
		event := strings.TrimPrefix(action, "UNTIL_")
		return s.actionUntil(ctx, event, false)
	case strings.HasPrefix(action, "UNTILLONG_"):
		event := strings.TrimPrefix(action, "UNTILLONG_")
		return s.actionUntil(ctx, event, false)
	case strings.HasPrefix(action, "UNTILSHORT_"):
		event := strings.TrimPrefix(action, "UNTILSHORT_")
		return s.actionUntil(ctx, event, true)

	case strings.HasPrefix(action, "RANDOM_"):
		return s.actionRandom(strings.TrimPrefix(action, "RANDOM_"))

	case strings.HasPrefix(action, "HOST_"):
		ch := strings.TrimPrefix(action, "HOST_")
		return "", s.SendCommand("host", strings.ToLower(ch))

	case strings.HasPrefix(action, "VARS_"):
		return s.actionVars(ctx, strings.TrimPrefix(action, "VARS_"))

	case strings.HasPrefix(action, "COMMAND_"):
		name := strings.TrimPrefix(action, "COMMAND_")
		return s.actionCommand(ctx, name)

	case strings.HasPrefix(action, "LIST_") && strings.HasSuffix(action, "_RANDOM"):
		name := strings.TrimPrefix(action, "LIST_")
		name = strings.TrimSuffix(name, "_RANDOM")
		return s.actionList(ctx, name)

	case strings.HasSuffix(action, "_COUNT"): // This case must come last.
		name := strings.TrimSuffix(action, "_COUNT")
		name = cleanCommandName(name)

		info, err := s.Channel.CommandInfos(models.CommandInfoWhere.Name.EQ(name)).One(ctx, s.Tx)
		if err != nil {
			if err == sql.ErrNoRows {
				return "?", nil
			}
			return "", err
		}

		return strconv.FormatInt(info.Count, 10), nil
	}

	return "(_" + action + "_)", nil
}

func walk(ctx context.Context, nodes []cbp.Node, fn func(ctx context.Context, action string) (string, error)) (string, error) {
	// Process all commands, converting them to text nodes.
	for i, node := range nodes {
		if node.Text != "" {
			continue
		}

		action, err := walk(ctx, node.Children, fn)
		if err != nil {
			return "", err
		}

		s, err := fn(ctx, action)
		if err != nil {
			return "", err
		}

		nodes[i] = cbp.Node{
			Text: s,
		}
	}

	var sb strings.Builder

	// Merge all strings.
	for _, node := range nodes {
		sb.WriteString(node.Text)
	}

	return sb.String(), nil
}

func (s *session) FirstParameter() string {
	param, _ := splitFirstSep(s.OrigCommandParams, ";")
	return strings.TrimSpace(param)
}

func (s *session) UserForModAction() (string, bool) {
	switch {
	case s.Type == sessionAutoreply:
		return s.User, true
	case s.UserLevel.CanAccess(levelModerator):
		p := s.FirstParameter()
		p, _ = splitSpace(p)
		return p, true
	default:
		return s.User, false
	}
}

func (s *session) NextParameter() string {
	var param string
	param, s.CommandParams = splitFirstSep(s.CommandParams, ";")
	return strings.TrimSpace(param)
}

func (s *session) actionSong(i int, url bool) (string, error) {
	// TODO: Precheck commands before running them for simple things (like using SONG without lastfm set).

	tracks, err := s.Tracks()
	if err != nil {
		if err == errLastFMDisabled {
			return "(Unknown)", nil
		}

		// TODO: return a message here?
		return "", err
	}

	if len(tracks) < i+1 {
		return "(Nothing)", nil
	}

	track := tracks[i]

	if url {
		return track.URL, nil
	}

	return track.Name + " by " + track.Artist, nil
}

func (s *session) actionRandom(action string) (string, error) {
	if strings.HasPrefix(action, "INT_") {
		action = strings.TrimPrefix(action, "INT_")
		minStr, maxStr := splitFirstSep(action, "_")
		if minStr == "" || maxStr == "" {
			return "0", nil
		}

		min, err := strconv.Atoi(minStr)
		if err != nil {
			return "0", nil
		}

		max, err := strconv.Atoi(maxStr)
		if err != nil {
			return "0", nil
		}

		switch {
		case max < min:
			return "0", nil
		case max == min:
			return strconv.Itoa(max), nil
		}

		x := s.Deps.Rand.Intn(max-min) + min
		return strconv.Itoa(x), nil
	}

	minStr, maxStr := splitFirstSep(action, "_")
	if minStr == "" || maxStr == "" {
		return "0.0", nil
	}

	min, err := strconv.ParseFloat(minStr, 64)
	if err != nil {
		return "0.0", nil
	}

	max, err := strconv.ParseFloat(maxStr, 64)
	if err != nil {
		return "0.0", nil
	}

	var x float64

	switch {
	case max < min:
		return "0.0", nil
	case max == min:
		x = max
	default:
		r := s.Deps.Rand.Float64()
		x = r*(max-min) + min
	}

	return strconv.FormatFloat(x, 'f', 1, 64), nil
}

func (s *session) actionVars(ctx context.Context, action string) (string, error) {
	name, action := splitFirstSep(action, "_")
	if name == "" || action == "" {
		return "(error)", nil
	}

	switch {
	case strings.HasPrefix(action, "GET_"):
		ch := strings.TrimPrefix(action, "GET_")
		if ch == "" {
			return "(error)", nil
		}

		v, _, err := s.VarGetByChannel(ctx, ch, name)
		if err != nil {
			return "", err
		}

		return v, nil

	case strings.HasPrefix(action, "SET_"):
		value := strings.TrimPrefix(action, "SET_")

		if err := s.VarSet(ctx, name, value); err != nil {
			return "", err
		}

		return value, nil

	case strings.HasPrefix(action, "INCREMENT_"):
		incStr := strings.TrimPrefix(action, "INCREMENT_")
		return s.actionVarInc(ctx, name, incStr, false)

	case strings.HasPrefix(action, "DECREMENT_"):
		decStr := strings.TrimPrefix(action, "DECREMENT_")
		return s.actionVarInc(ctx, name, decStr, true)

	default:
		return "(error)", nil
	}
}

func (s *session) actionVarInc(ctx context.Context, name, incStr string, dec bool) (string, error) {
	if incStr == "" {
		return "(error)", nil
	}

	inc, err := strconv.ParseInt(incStr, 10, 64)
	if err != nil {
		return "(error)", nil
	}

	if dec {
		inc = 0 - inc
	}

	v, badVar, err := s.VarIncrement(ctx, name, inc)
	if err != nil {
		return "", err
	}

	if badVar {
		return "(error)", nil
	}

	return strconv.FormatInt(v, 10), nil
}

func (s *session) actionTime(ctx context.Context, tz string, layout string) (string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	now := s.Deps.Clock.Now().In(loc)

	return now.Format(layout), nil
}

const dayDur = 24 * time.Hour

func (s *session) actionUntil(ctx context.Context, timestamp string, short bool) (string, error) {
	t, err := parseUntilTimestamp(timestamp)
	if err != nil {
		return "(error)", nil
	}

	dur := s.Deps.Clock.Until(t).Round(time.Minute)

	if short {
		if dur/dayDur == 0 {
			return trimSeconds(dur.String()), nil
		}

		// Extra logic to add the days to the duration string.
		negative := dur < 0
		if negative {
			dur = 0 - dur
		}

		days := dur / dayDur
		dur -= days * dayDur

		var builder strings.Builder
		if negative {
			builder.WriteByte('-')
		}
		builder.WriteString(strconv.FormatInt(int64(days), 10))
		builder.WriteByte('d')
		builder.WriteString(dur.String())

		return trimSeconds(builder.String()), nil
	}

	return durafmt.Parse(dur).String(), nil
}

func trimSeconds(d string) string {
	if d == "0s" {
		return d
	}
	return strings.TrimSuffix(d, "0s")
}

var easternTime = mustLoadLocation("America/New_York")

func mustLoadLocation(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}

func parseUntilTimestamp(timestamp string) (time.Time, error) {
	if timestamp == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}

	t, err := time.Parse(time.RFC3339, timestamp)
	if err == nil {
		return t, nil
	}

	// CoeBot would parse using a not-quite RFC3339 string in the host system's timezone.
	// Do that here, assuming an Eastern time zone.
	return dateparse.ParseIn(timestamp, easternTime)
}

type commandGuard string

func withCommandGuard(ctx context.Context, command string) context.Context {
	return context.WithValue(ctx, commandGuard(command), true)
}

func (s *session) actionCommand(ctx context.Context, name string) (string, error) {
	name = cleanCommandName(name)

	if ctx.Value(commandGuard(name)) != nil {
		return "", nil
	}

	ctx = withCommandGuard(ctx, name)

	_, commandMsg, found, err := modelsx.FindCommand(ctx, s.Tx, s.Channel.ID, name, false)
	if err != nil || !found {
		return "", err
	}

	if commandMsg.Valid {
		return processCommand(ctx, s, commandMsg.String)
	}

	return "(error)", nil
}

func (s *session) actionList(ctx context.Context, name string) (string, error) {
	name = cleanCommandName(name)

	info, err := s.Channel.CommandInfos(
		models.CommandInfoWhere.Name.EQ(name),
		qm.Load(models.CommandInfoRels.CommandList),
	).One(ctx, s.Tx)
	switch {
	case err == sql.ErrNoRows:
		return "(error)", nil
	case err != nil:
		return "", err
	case info.R.CommandList == nil:
		return "(error)", nil
	}

	items := info.R.CommandList.Items
	if len(items) == 0 {
		return "", nil
	}

	i := s.Deps.Rand.Intn(len(items))
	item := items[i]

	ctx = withCommandGuard(ctx, name)
	return processCommand(ctx, s, item)
}

func (s *session) actionTweet(ctx context.Context) (string, error) {
	const tweetGuard = "?tweet"

	if ctx.Value(commandGuard(tweetGuard)) != nil {
		return "", nil
	}

	ctx = withCommandGuard(ctx, tweetGuard)

	tweet := s.Channel.Tweet
	ctx = withCommandGuard(ctx, "?tweet")
	text, err := processCommand(ctx, s, tweet)
	if err != nil {
		return "", err
	}

	u := "https://twitter.com/intent/tweet?text=" + url.QueryEscape(text)
	return s.ShortenLink(ctx, u)
}
