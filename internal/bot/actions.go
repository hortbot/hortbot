package bot

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/apis/extralife"
)

var testingAction func(ctx context.Context, action string) (string, error, bool)

//nolint:gocyclo
func (s *session) doAction(ctx context.Context, action string) (string, error) {
	if isTesting && testingAction != nil {
		s, err, ok := testingAction(ctx, action)
		if ok {
			return s, err
		}
	}

	// TODO: ORIG_PARAMS to always fetch the entire thing.
	// TODO: Figure out how to deal with change in behavior for PARAMETER (DFS versus BFS)
	// 	     Maybe PARAMETER[0]?

	// TODO: run auto-reply only things first, then check if autoreply and return.

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
		// TODO: check user level
		return "", s.SendCommand("subscribers")
	case "SUBMODE_OFF":
		// TODO: check user level
		return "", s.SendCommand("subscribersoff")
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
		// TODO: check user level
		if u := s.FirstParameter(); u != "" {
			u, _ = splitSpace(u)
			u = strings.ToLower(u)
			return "", s.SendCommand("timeout", strings.ToLower(u), "1")
		}
		return "", nil // TODO: error?
	case "TIMEOUT":
		// TODO: check user level
		if u := s.FirstParameter(); u != "" {
			u, _ = splitSpace(u)
			u = strings.ToLower(u)
			return "", s.SendCommand("timeout", strings.ToLower(u))
		}
		return "", nil // TODO: error?
	case "BAN":
		// TODO: check user level
		if u := s.FirstParameter(); u != "" {
			u, _ = splitSpace(u)
			u = strings.ToLower(u)
			return "", s.SendCommand("ban", strings.ToLower(u))
		}
		return "", nil // TODO: error?
	case "DELETE":
		return "", s.DeleteMessage()
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
		chatters, _ := s.Deps.Twitch.GetChatters(ctx, s.Channel.Name)
		return strconv.FormatInt(chatters, 10), nil
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
	}

	switch {
	case strings.HasPrefix(action, "DATE_"):
		return s.actionTime(ctx, strings.TrimPrefix(action, "DATE_"), "Jan 2, 2006")
	case strings.HasPrefix(action, "TIME_"):
		return s.actionTime(ctx, strings.TrimPrefix(action, "TIME_"), "3:04 PM")
	case strings.HasPrefix(action, "TIME24_"):
		return s.actionTime(ctx, strings.TrimPrefix(action, "TIME24_"), "15:04")
	case strings.HasPrefix(action, "DATETIME_"):
		return s.actionTime(ctx, strings.TrimPrefix(action, "DATETIME_"), "Jan 2, 2006 3:04 PM")
	case strings.HasPrefix(action, "DATETIME24_"):
		return s.actionTime(ctx, strings.TrimPrefix(action, "DATETIME24_"), "Jan 2, 2006 15:04")

	case strings.HasPrefix(action, "RANDOM_"):
		return s.actionRandom(strings.TrimPrefix(action, "RANDOM_"))

	case strings.HasPrefix(action, "HOST_"):
		ch := strings.TrimPrefix(action, "HOST_")
		return "", s.SendCommand("host", strings.ToLower(ch))

	case strings.HasPrefix(action, "VARS_"):
		return s.actionVars(ctx, strings.TrimPrefix(action, "VARS_"))

	case strings.HasSuffix(action, "_COUNT"): // This case must come last.
		name := strings.TrimSuffix(action, "_COUNT")
		name = cleanCommandName(name)

		command, err := s.Channel.CustomCommands(models.CustomCommandWhere.Name.EQ(name)).One(ctx, s.Tx)
		switch err {
		case nil:
			return strconv.FormatInt(command.Count, 10), nil
		case sql.ErrNoRows:
			return "?", nil
		default:
			return "", err
		}

	default:
		// TODO: Should this return "(_" + action "_)" to match the old behavior of not replacing things?
		return "", fmt.Errorf("unknown action: %s", action)
	}

	// No return here; use default in the switch case to catch
	// unwanted fallthrough at compile time.
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
