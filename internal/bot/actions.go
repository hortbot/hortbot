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
	"unicode"

	"github.com/araddon/dateparse"
	"github.com/dghubble/trie"
	"github.com/hako/durafmt"
	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/stringsx"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/zikaeroh/ctxlog"
	"go.opencensus.io/trace"
)

var testingAction func(ctx context.Context, action string) (string, error, bool)

const (
	actionMsgError  = "(error)"
	actionMsgNotSet = "(Not set)"
)

type (
	actionTopFunc func(ctx context.Context, s *session, action string) (string, error)
	actionFunc    func(ctx context.Context, s *session, actionName, value string) (string, error)
)

var actionTrie = trie.NewRuneTrie()

func init() {
	// To prevent initialization loop.
	add := func(name string, fn actionTopFunc) {
		actionTrie.Put(name, fn)
	}

	addExact := func(name string, fn actionFunc) {
		add(name, func(ctx context.Context, s *session, action string) (string, error) {
			if action != name {
				return "(_" + action + "_)", nil
			}

			if err := s.Deps.Redis.IncrementActionUsageStat(ctx, action); err != nil {
				return "", err
			}

			return fn(ctx, s, name, "")
		})
	}

	addPrefix := func(prefix string, fn actionFunc) {
		add(prefix, func(ctx context.Context, s *session, action string) (string, error) {
			if err := s.Deps.Redis.IncrementActionUsageStat(ctx, prefix); err != nil {
				return "", err
			}

			value := strings.TrimPrefix(action, prefix)
			return fn(ctx, s, prefix, value)
		})
	}

	addExact("PARAMETER", actionParameter)
	addExact("P", actionParameter)
	addExact("PARAMETER_CAPS", actionParameter)
	addExact("P_CAPS", actionParameter)
	addExact("MESSAGE_COUNT", actionMessageCount)
	addExact("SONG", actionSong)
	addExact("SONG_URL", actionSong)
	addExact("LAST_SONG", actionSong)
	addExact("QUOTE", actionQuote)
	addExact("USER", actionUser)
	addExact("USER_DISPLAY", actionUserDisplay)
	addExact("CHANNEL_URL", actionChannelURL)
	addExact("SUBMODE_ON", actionSubmode)
	addExact("SUBMODE_OFF", actionSubmode)
	addExact("SILENT", actionSilent)
	addExact("NUMCHANNELS", actionNumChannels)
	addExact("PURGE", actionPurge)
	addExact("TIMEOUT", actionTimeout)
	addExact("BAN", actionBan)
	addExact("DELETE", actionDelete)
	addExact("REGULARS_ONLY", actionRegularsOnly)
	addExact("EXTRALIFE_AMOUNT", actionExtraLifeAmount)
	addExact("ONLINE_CHECK", actionOnlineCheck)
	addExact("GAME", actionGame)
	addExact("GAME_CLEAN", actionGame)
	addExact("STATUS", actionStatus)
	addExact("VIEWERS", actionViewers)
	addExact("DATE", actionTime("Jan 2, 2006"))
	addExact("TIME", actionTime("3:04 PM"))
	addExact("TIME24", actionTime("15:04"))
	addExact("DATETIME", actionTime("Jan 2, 2006 3:04 PM"))
	addExact("DATETIME24", actionTime("Jan 2, 2006 15:04"))
	addExact("STEAM_PROFILE", actionSteamProfile)
	addExact("STEAM_GAME", actionSteamGame)
	addExact("STEAM_SERVER", actionSteamServer)
	addExact("STEAM_STORE", actionSteamStore)
	addExact("TWEET_URL", actionTweet)
	addExact("BOT_HELP", actionBotHelp)
	addExact("GAME_LINK", actionGameLink)

	addPrefix("PARAMETER_", actionParameterIndex)
	addPrefix("P_", actionParameterIndex)
	addPrefix("PARAMETER_OR_", actionParameterOr)
	addPrefix("P_OR_", actionParameterOr)
	addPrefix("DATE_", actionTime("Jan 2, 2006"))
	addPrefix("TIME_", actionTime("3:04 PM"))
	addPrefix("TIME24_", actionTime("15:04"))
	addPrefix("DATETIME_", actionTime("Jan 2, 2006 3:04 PM"))
	addPrefix("DATETIME24_", actionTime("Jan 2, 2006 15:04"))
	addPrefix("UNTIL_", actionUntil)
	addPrefix("UNTILLONG_", actionUntil)
	addPrefix("UNTILSHORT_", actionUntil)
	addPrefix("RANDOM_", actionRandom)
	addPrefix("VARS_", actionVars)
	addPrefix("COMMAND_", actionCommand)
	addPrefix("LIST_", actionList)
	addPrefix("TEXTAPI_", actionTextAPI)
	addPrefix("PESC_", actionPathEscape)
	addPrefix("QESC_", actionQueryEscape)
	addPrefix("CAPS_", actionCaps)
	addPrefix("QUIET_", actionQuiet)
}

func findAction(action string) actionTopFunc {
	var found interface{}

	_ = actionTrie.WalkPath(action, func(key string, value interface{}) error {
		found = value
		return nil
	})

	fn, _ := found.(actionTopFunc)
	return fn
}

func (s *session) doAction(ctx context.Context, action string) (string, error) {
	if action == "" {
		panic("doAction with an empty action")
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

	if fn := findAction(action); fn != nil {
		return fn(ctx, s, action)
	}

	if strings.HasSuffix(action, "_COUNT") {
		name := strings.TrimSuffix(action, "_COUNT")
		name = cleanCommandName(name)

		info, err := s.Channel.CommandInfos(models.CommandInfoWhere.Name.EQ(name)).One(ctx, s.Tx)
		switch err {
		case nil:
			return strconv.FormatInt(info.Count, 10), nil
		case sql.ErrNoRows:
			return actionMsgError, nil
		default:
			return "", err
		}
	}

	return "(_" + action + "_)", nil
}

func walk(ctx context.Context, nodes []cbp.Node, fn func(ctx context.Context, action string) (string, error)) (string, error) {
	// Process all commands, converting them to text nodes.
	for i, node := range nodes {
		if err := ctx.Err(); err != nil {
			return "", err
		}

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

func (s *session) Parameters() []string {
	if s.parameters != nil {
		return *s.parameters
	}

	var params []string
	if s.CommandParams != "" {
		params = strings.Split(s.CommandParams, ";")
		for i, p := range params {
			params[i] = strings.TrimSpace(p)
		}
	}
	s.parameters = &params
	return params
}

func (s *session) ParameterAt(i int) *string {
	if i < 0 {
		return nil
	}

	params := s.Parameters()
	if i < len(params) {
		return &params[i]
	}

	return nil
}

func (s *session) FirstParameter() *string {
	return s.ParameterAt(0)
}

// NextParameter iterates over the parameters, returning nil when no more
// have been found. If there's only one parameter, then instead of returning
// nil when running out, it will continue to return the first.
func (s *session) NextParameter() *string {
	p := s.ParameterAt(s.parameterIndex)
	if p != nil {
		s.parameterIndex++
		return p
	}

	if len(s.Parameters()) == 1 {
		return s.ParameterAt(0)
	}

	return nil
}

func (s *session) UserForModAction() (name string, display string, do bool) {
	if s.Type == sessionAutoreply {
		return s.User, s.UserDisplay, true
	}

	p := s.FirstParameter()
	if p == nil {
		return "", "", false
	}
	u, _ := splitSpace(*p)
	u2 := cleanUsername(u)
	return u2, u, true
}

func actionParameter(ctx context.Context, s *session, actionName, value string) (string, error) {
	upper := false
	if strings.HasSuffix(actionName, "_CAPS") {
		upper = true
	}

	if p := s.NextParameter(); p != nil {
		if upper {
			return strings.ToUpper(*p), nil
		}
		return *p, nil
	}
	return "", nil
}

func actionParameterOr(ctx context.Context, s *session, actionName, value string) (string, error) {
	v := ""

	if p := s.NextParameter(); p != nil {
		v = *p
	}

	if v != "" {
		return v, nil
	}

	return value, nil
}

func actionParameterIndex(ctx context.Context, s *session, actionName, value string) (string, error) {
	upper := false
	is := value

	const orStr = "_OR_"
	or := false
	orV := ""

	if i := strings.Index(value, orStr); i >= 0 {
		is = value[:i]
		orV = value[i+len(orStr):]
		or = true
	} else if strings.HasSuffix(is, "_CAPS") {
		upper = true
		is = strings.TrimSuffix(is, "_CAPS")
	}

	v := ""

	if i, err := strconv.Atoi(is); err == nil && i > 0 {
		if p := s.ParameterAt(i - 1); p != nil {
			if upper {
				v = strings.ToUpper(*p)
			} else {
				return *p, nil
			}
		}
	} else {
		return "(_" + actionName + value + "_)", nil
	}

	if v != "" {
		return v, nil
	}

	if or {
		return orV, nil
	}

	return "", nil
}

func actionMessageCount(ctx context.Context, s *session, actionName, value string) (string, error) {
	return strconv.FormatInt(s.Channel.MessageCount, 10), nil
}

func actionSong(ctx context.Context, s *session, actionName, value string) (string, error) {
	tracks, err := s.Tracks(ctx)
	if err != nil {
		if err == errLastFMDisabled {
			return "(Unknown)", nil
		}

		var apiErr *apiclient.Error
		if !errors.As(err, &apiErr) {
			return "", err
		}

		return actionMsgError, nil
	}

	url := false
	i := 0

	switch actionName {
	case "SONG":
	case "SONG_URL":
		url = true
	case "LAST_SONG":
		i = 1
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

func actionQuote(ctx context.Context, s *session, actionName, value string) (string, error) {
	q, ok, err := getRandomQuote(ctx, s.Tx, s.Channel)
	if err != nil {
		return "", err
	}

	if !ok {
		return "No quotes.", nil
	}

	return q.Quote, nil
}

func actionUser(ctx context.Context, s *session, actionName, value string) (string, error) {
	return s.User, nil
}

func actionUserDisplay(ctx context.Context, s *session, actionName, value string) (string, error) {
	return s.UserDisplay, nil
}

func actionChannelURL(ctx context.Context, s *session, actionName, value string) (string, error) {
	return "twitch.tv/" + s.Channel.Name, nil
}

func actionSubmode(ctx context.Context, s *session, actionName, value string) (string, error) {
	if s.UserLevel.CanAccess(levelModerator) {
		subMode := true
		if actionName == "SUBMODE_OFF" {
			subMode = false
		}

		return "", s.UpdateChatSettings(ctx, &twitch.ChatSettingsPatch{SubscriberMode: &subMode})
	}
	return "", nil
}

func actionSilent(ctx context.Context, s *session, actionName, value string) (string, error) {
	s.Silent = true
	return "", nil
}

func actionNumChannels(ctx context.Context, s *session, actionName, value string) (string, error) {
	count, err := models.Channels(models.ChannelWhere.Active.EQ(true)).Count(ctx, s.Tx)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(count, 10), nil
}

func actionPurge(ctx context.Context, s *session, actionName, value string) (string, error) {
	u, d, do := s.UserForModAction()
	if do && u != "" {
		return d, s.BanByUsername(ctx, strings.ToLower(u), 1, "Purging chat messages")
	}
	return d, nil
}

func actionTimeout(ctx context.Context, s *session, actionName, value string) (string, error) {
	u, d, do := s.UserForModAction()
	if do && u != "" {
		return d, s.BanByUsername(ctx, strings.ToLower(u), 600, "Timeout via command action")
	}
	return d, nil
}

func actionBan(ctx context.Context, s *session, actionName, value string) (string, error) {
	u, d, do := s.UserForModAction()
	if do && u != "" {
		return d, s.BanByUsername(ctx, strings.ToLower(u), 0, "Ban via command action")
	}
	return d, nil
}

func actionDelete(ctx context.Context, s *session, actionName, value string) (string, error) {
	if s.Type == sessionAutoreply {
		return s.User, s.DeleteMessage(ctx)
	}
	return "", nil
}

func actionRegularsOnly(ctx context.Context, s *session, actionName, value string) (string, error) {
	return "", nil
}

func actionExtraLifeAmount(ctx context.Context, s *session, actionName, value string) (string, error) {
	if s.Deps.ExtraLife == nil || s.Channel.ExtraLifeID == 0 {
		return "?", nil
	}

	amount, err := s.Deps.ExtraLife.GetDonationAmount(ctx, s.Channel.ExtraLifeID)
	if err == nil {
		return fmt.Sprintf("$%.2f", amount), nil
	}

	var apiErr *apiclient.Error
	if errors.As(err, &apiErr) {
		if apiErr.IsNotFound() {
			return "Bad Extra Life participant ID", nil
		}
		return "Extra Life server error", nil
	}

	return "", err
}

func actionOnlineCheck(ctx context.Context, s *session, actionName, value string) (string, error) {
	isLive, err := s.IsLive(ctx)
	if err != nil {
		return "", err
	}

	if !isLive {
		panic("ONLINE_CHECK should have been handled earlier")
	}

	return "", nil
}

func actionGame(ctx context.Context, s *session, actionName, value string) (string, error) {
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

	if actionName == "GAME_CLEAN" {
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
}

func actionGameLink(ctx context.Context, s *session, actionName, value string) (string, error) {
	links, err := s.GameLinks(ctx)
	if err != nil {
		return actionMsgError, nil //nolint:nilerr
	}

	if len(links) == 0 {
		return "(unavailable)", nil
	}

	return links[0].URL, nil
}

func actionStatus(ctx context.Context, s *session, actionName, value string) (string, error) {
	ch, err := s.TwitchChannel(ctx)
	if err != nil {
		return actionMsgError, nil //nolint:nilerr
	}

	status := ch.Title

	if status == "" {
		return actionMsgNotSet, nil
	}

	return status, nil
}

func actionViewers(ctx context.Context, s *session, actionName, value string) (string, error) {
	var viewers int64
	stream, err := s.TwitchStream(ctx)
	if err == nil && stream != nil {
		viewers = int64(stream.ViewerCount)
	}
	return strconv.FormatInt(viewers, 10), nil
}

func actionUntil(ctx context.Context, s *session, actionName, value string) (string, error) {
	short := strings.HasSuffix(actionName, "SHORT_")

	t, err := parseUntilTimestamp(value)
	if err != nil {
		return actionMsgError, nil //nolint:nilerr
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

func actionSteamProfile(ctx context.Context, s *session, actionName, value string) (string, error) {
	summary, err := s.SteamSummary(ctx)
	if err != nil {
		return actionMsgError, nil //nolint:nilerr
	}

	url := summary.ProfileURL
	if url == "" {
		return "(unavailable)", nil
	}
	return url, nil
}

func actionSteamGame(ctx context.Context, s *session, actionName, value string) (string, error) {
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
}

func actionSteamServer(ctx context.Context, s *session, actionName, value string) (string, error) {
	summary, err := s.SteamSummary(ctx)
	if err != nil {
		return actionMsgError, nil //nolint:nilerr
	}

	server := summary.GameServer
	if server == "" {
		return "(unavailable)", nil
	}
	return server, nil
}

func actionSteamStore(ctx context.Context, s *session, actionName, value string) (string, error) {
	summary, err := s.SteamSummary(ctx)
	if err != nil {
		return actionMsgError, nil //nolint:nilerr
	}

	gameID := summary.GameID
	if gameID == "" {
		return "(unavailable)", nil
	}
	return "https://store.steampowered.com/app/" + gameID, nil
}

func actionTweet(ctx context.Context, s *session, actionName, value string) (string, error) {
	const tweetGuard = "?tweet"

	if ctx.Value(commandGuard(tweetGuard)) != nil {
		return "", nil
	}

	ctx = withCommandGuard(ctx, tweetGuard)
	tweet := s.Channel.Tweet

	text, err := processCommand(ctx, s, tweet)
	if err != nil {
		return "", err
	}

	u := "https://twitter.com/intent/tweet?text=" + url.QueryEscape(text)
	return s.ShortenLink(ctx, u)
}

func actionBotHelp(ctx context.Context, s *session, actionName, value string) (string, error) {
	return s.HelpMessage(), nil
}

func actionRandom(ctx context.Context, s *session, actionName, value string) (string, error) {
	if strings.HasPrefix(value, "INT_") {
		value = strings.TrimPrefix(value, "INT_")
		minStr, maxStr := stringsx.SplitByte(value, '_')
		if minStr == "" || maxStr == "" {
			return "0", nil
		}

		min, err := strconv.Atoi(minStr)
		if err != nil {
			return "0", nil //nolint:nilerr
		}

		max, err := strconv.Atoi(maxStr)
		if err != nil {
			return "0", nil //nolint:nilerr
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

	minStr, maxStr := stringsx.SplitByte(value, '_')
	if minStr == "" || maxStr == "" {
		return "0.0", nil
	}

	min, err := strconv.ParseFloat(minStr, 64)
	if err != nil {
		return "0.0", nil //nolint:nilerr
	}

	max, err := strconv.ParseFloat(maxStr, 64)
	if err != nil {
		return "0.0", nil //nolint:nilerr
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

func actionVars(ctx context.Context, s *session, actionName, value string) (string, error) {
	name, value := stringsx.SplitByte(value, '_')
	if name == "" || value == "" {
		return actionMsgError, nil
	}

	switch {
	case value == "GET":
		v, _, err := s.VarGet(ctx, name)
		return v, err

	case strings.HasPrefix(value, "GET_"):
		ch := strings.TrimPrefix(value, "GET_")
		if ch == "" {
			return actionMsgError, nil
		}

		v, _, err := s.VarGetByChannel(ctx, ch, name)
		return v, err

	case strings.HasPrefix(value, "SET_"):
		value := strings.TrimPrefix(value, "SET_")

		if err := s.VarSet(ctx, name, value); err != nil {
			return "", err
		}

		return value, nil

	case strings.HasPrefix(value, "INCREMENT_"):
		incStr := strings.TrimPrefix(value, "INCREMENT_")
		return actionVarInc(ctx, s, name, incStr, false)

	case strings.HasPrefix(value, "DECREMENT_"):
		decStr := strings.TrimPrefix(value, "DECREMENT_")
		return actionVarInc(ctx, s, name, decStr, true)

	default:
		return actionMsgError, nil
	}
}

func actionVarInc(ctx context.Context, s *session, name, incStr string, dec bool) (string, error) {
	if incStr == "" {
		return actionMsgError, nil
	}

	inc, err := strconv.ParseInt(incStr, 10, 64)
	if err != nil {
		return actionMsgError, nil //nolint:nilerr
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

func actionTime(timeFormat string) actionFunc {
	return func(ctx context.Context, s *session, actionName, value string) (string, error) {
		loc, err := time.LoadLocation(value)
		if err != nil {
			loc = time.UTC // TODO: Default to a per-channel timezone.
		}

		now := s.Deps.Clock.Now().In(loc)

		return now.Format(timeFormat), nil
	}
}

const (
	dayDur                    = 24 * time.Hour
	minDuration time.Duration = -1 << 63
	maxDuration time.Duration = 1<<63 - 1
)

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

func isCommandGuarded(ctx context.Context, name string) bool {
	return ctx.Value(commandGuard(name)) != nil
}

func withCommandGuard(ctx context.Context, command string) context.Context {
	return context.WithValue(ctx, commandGuard(command), true)
}

func actionCommand(ctx context.Context, s *session, prefix, name string) (string, error) {
	name = cleanCommandName(name)

	if isCommandGuarded(ctx, name) {
		return actionMsgError, nil
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

func actionList(ctx context.Context, s *session, prefix, name string) (string, error) {
	if !strings.HasSuffix(name, "_RANDOM") {
		return "(_" + prefix + name + "_)", nil
	}

	name = strings.TrimSuffix(name, "_RANDOM")
	name = cleanCommandName(name)

	if isCommandGuarded(ctx, name) {
		return actionMsgError, nil
	}

	ctx = withCommandGuard(ctx, name)

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

	return processCommand(ctx, s, item)
}

func actionTextAPI(ctx context.Context, s *session, prefix, u string) (string, error) {
	if s.Type == sessionAutoreply {
		return actionMsgError, nil
	}

	body, err := s.Deps.Simple.Plaintext(ctx, u)
	if err != nil {
		var apiErr *apiclient.Error
		if !errors.As(err, &apiErr) {
			// This usually indicates an error with the bot.
			ctxlog.Error(ctx, "error fetching API", ctxlog.PlainError(err))
			return actionMsgError, nil
		}
		// If it's an API error (i.e. 404, 500, etc), then just use the body below.
	}

	body = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, body)

	return strings.TrimSpace(body), nil
}

func actionPathEscape(ctx context.Context, s *session, actionName, value string) (string, error) {
	return url.PathEscape(value), nil
}

func actionQueryEscape(ctx context.Context, s *session, actionName, value string) (string, error) {
	return url.QueryEscape(value), nil
}

func actionCaps(ctx context.Context, s *session, actionName, value string) (string, error) {
	return strings.ToUpper(value), nil
}

func actionQuiet(ctx context.Context, s *session, actionName, value string) (string, error) {
	return "", nil
}
