package bot

import (
	"context"
	"database/sql"
	"errors"
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
	"github.com/hortbot/hortbot/internal/pkg/apis"
	"github.com/hortbot/hortbot/internal/pkg/stringsx"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"go.opencensus.io/trace"
)

var testingAction func(ctx context.Context, action string) (string, error, bool)

const (
	actionMsgError  = "(error)"
	actionMsgNotSet = "(Not set)"
)

//nolint:gocyclo
func (s *session) doAction(ctx context.Context, action string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	ctx, span := trace.StartSpan(ctx, "doAction")
	defer span.End()

	span.AddAttributes(trace.StringAttribute("action", action))

	if isTesting && testingAction != nil {
		s, err, ok := testingAction(ctx, action)
		if ok {
			return s, err
		}
	}

	// Exact matches
	switch action {
	case "PARAMETER":
		p := s.NextParameter()
		if p != nil {
			return *p, nil
		}
		// Emulate this odd CoeBot behavior.
		return "(_" + action + "_)", nil
	case "PARAMETER_CAPS":
		p := s.NextParameter()
		if p != nil {
			return strings.ToUpper(*p), nil
		}
		return "(_" + action + "_)", nil
	case "MESSAGE_COUNT":
		return strconv.FormatInt(s.Channel.MessageCount, 10), nil
	case "SONG":
		return s.actionSong(ctx, 0, false)
	case "SONG_URL":
		return s.actionSong(ctx, 0, true)
	case "LAST_SONG":
		return s.actionSong(ctx, 1, false)
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
			return "", s.SendCommand(ctx, "subscribers")
		}
		return "", nil
	case "SUBMODE_OFF":
		if s.UserLevel.CanAccess(levelModerator) {
			return "", s.SendCommand(ctx, "subscribersoff")
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
		return "", s.SendCommand(ctx, "unhost")
	case "PURGE":
		u, do := s.UserForModAction()
		if do && u != "" {
			return u, s.SendCommand(ctx, "timeout", strings.ToLower(u), "1")
		}
		return u, nil
	case "TIMEOUT":
		u, do := s.UserForModAction()
		if do && u != "" {
			return u, s.SendCommand(ctx, "timeout", strings.ToLower(u))
		}
		return u, nil
	case "BAN":
		u, do := s.UserForModAction()
		if do && u != "" {
			return u, s.SendCommand(ctx, "ban", strings.ToLower(u))
		}
		return u, nil
	case "DELETE":
		if s.Type == sessionAutoreply {
			return s.User, s.DeleteMessage(ctx)
		}
		return "", nil
	case "REGULARS_ONLY":
		return "", nil
	case "EXTRALIFE_AMOUNT":
		if s.Deps.ExtraLife == nil || s.Channel.ExtraLifeID == 0 {
			return "?", nil
		}

		amount, err := s.Deps.ExtraLife.GetDonationAmount(ctx, s.Channel.ExtraLifeID)
		if err == nil {
			return fmt.Sprintf("$%.2f", amount), nil
		}

		var apiErr *apis.Error
		if errors.As(err, &apiErr) {
			if apiErr.IsNotFound() {
				return "Bad Extra Life participant ID", nil
			}
			return "Extra Life server error", nil
		}

		return "", err

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
			game = actionMsgError
		} else {
			game = ch.Game
		}

		if game == "" {
			game = actionMsgNotSet
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
			return actionMsgError, nil
		}

		status := ch.Status

		if status == "" {
			return actionMsgNotSet, nil
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
			return actionMsgError, nil
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
				game = actionMsgError
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
			return actionMsgError, nil
		}

		server := summary.GameServer
		if server == "" {
			return "(unavailable)", nil
		}
		return server, nil
	case "STEAM_STORE":
		summary, err := s.SteamSummary(ctx)
		if err != nil {
			return actionMsgError, nil
		}

		gameID := summary.GameID
		if gameID == "" {
			return "(unavailable)", nil
		}
		return "http://store.steampowered.com/app/" + gameID, nil
	case "TWEET_URL":
		return s.actionTweet(ctx)

	case "BOT_HELP":
		return s.HelpMessage(), nil
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
		return "", s.SendCommand(ctx, "host", strings.ToLower(ch))

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

func (s *session) parameterAt(i int) *string {
	var params []string
	if s.Parameters == nil {
		params = strings.Split(s.CommandParams, ";")
		for i, p := range params {
			params[i] = strings.TrimSpace(p)
		}
		s.Parameters = &params
	} else {
		params = *s.Parameters
	}

	if i < len(params) {
		return &params[i]
	}

	return nil
}

func (s *session) FirstParameter() *string {
	return s.parameterAt(0)
}

// NextParameter iterates over the parameters, returning nil when no more
// have been found. If there's only one parameter, then instead of returning
// nil when running out, it will continue to return the first.
func (s *session) NextParameter() *string {
	p := s.parameterAt(s.ParameterIndex)
	if p != nil {
		s.ParameterIndex++
		return p
	}

	if len(*s.Parameters) == 1 {
		return s.parameterAt(0)
	}

	return nil
}

func (s *session) UserForModAction() (string, bool) {
	switch {
	case s.Type == sessionAutoreply:
		return s.User, true
	case s.UserLevel.CanAccess(levelModerator):
		p := s.FirstParameter()
		if p == nil {
			return "", false
		}
		u, _ := splitSpace(*p)
		return u, true
	default:
		return s.User, false
	}
}

func (s *session) actionSong(ctx context.Context, i int, url bool) (string, error) {
	tracks, err := s.Tracks(ctx)
	if err != nil {
		if err == errLastFMDisabled {
			return "(Unknown)", nil
		}
		return actionMsgError, err
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
		minStr, maxStr := stringsx.SplitByte(action, '_')
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

	minStr, maxStr := stringsx.SplitByte(action, '_')
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
	name, action := stringsx.SplitByte(action, '_')
	if name == "" || action == "" {
		return actionMsgError, nil
	}

	switch {
	case strings.HasPrefix(action, "GET_"):
		ch := strings.TrimPrefix(action, "GET_")
		if ch == "" {
			return actionMsgError, nil
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
		return actionMsgError, nil
	}
}

func (s *session) actionVarInc(ctx context.Context, name, incStr string, dec bool) (string, error) {
	if incStr == "" {
		return actionMsgError, nil
	}

	inc, err := strconv.ParseInt(incStr, 10, 64)
	if err != nil {
		return actionMsgError, nil
	}

	if dec {
		inc = 0 - inc
	}

	v, badVar, err := s.VarIncrement(ctx, name, inc)
	if err != nil {
		return "", err
	}

	if badVar {
		return actionMsgError, nil
	}

	return strconv.FormatInt(v, 10), nil
}

func (s *session) actionTime(_ context.Context, tz string, layout string) (string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	now := s.Deps.Clock.Now().In(loc)

	return now.Format(layout), nil
}

const (
	dayDur                    = 24 * time.Hour
	minDuration time.Duration = -1 << 63
	maxDuration time.Duration = 1<<63 - 1
)

func (s *session) actionUntil(ctx context.Context, timestamp string, short bool) (string, error) {
	t, err := parseUntilTimestamp(timestamp)
	if err != nil {
		return actionMsgError, nil
	}

	now := s.Deps.Clock.Now()

	dur := t.Sub(now)
	switch dur {
	case maxDuration, minDuration:
		return actionMsgError, nil // TODO: Attempt to handle dates which are further 290+ years apart, for fun.
	}

	dur = dur.Round(time.Minute)

	if short {
		if dur/dayDur == 0 {
			return trimSeconds(dur.String()), nil
		}

		// Extra logic to add the days to the duration string.
		negative := dur < 0
		if negative {
			dur = 0 - dur
		}

		tmp := dur / dayDur
		dur -= tmp * dayDur
		days := int64(tmp)

		var builder strings.Builder
		if negative {
			builder.WriteByte('-')
		}
		builder.WriteString(strconv.FormatInt(days, 10))
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

	if x, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		return time.Unix(x, 0), nil
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

	return actionMsgError, nil
}

func (s *session) actionList(ctx context.Context, name string) (string, error) {
	name = cleanCommandName(name)

	info, err := s.Channel.CommandInfos(
		models.CommandInfoWhere.Name.EQ(name),
		qm.Load(models.CommandInfoRels.CommandList),
	).One(ctx, s.Tx)
	switch {
	case err == sql.ErrNoRows:
		return actionMsgError, nil
	case err != nil:
		return "", err
	case info.R.CommandList == nil:
		return actionMsgError, nil
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
