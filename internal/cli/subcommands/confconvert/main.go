// Package confconvert implements the main command for the CoeBot config converter.
package confconvert

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/friendsofgo/errors"
	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/cli"
	"github.com/hortbot/hortbot/internal/cli/flags/httpflags"
	"github.com/hortbot/hortbot/internal/cli/flags/twitchflags"
	"github.com/hortbot/hortbot/internal/confimport"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/volatiletech/null/v8"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

type cmd struct {
	cli.Common
	Twitch twitchflags.Twitch
	HTTP   httpflags.HTTP

	Dir        []string `long:"dir" description:"Directory containing CoeBot JSON config files"`
	Positional struct {
		Files []string `positional-arg-name:"FILE"`
	} `positional-args:"true"`

	Out       string `long:"out" description:"Output directory for confimport configs" required:"true"`
	SiteDumps string `long:"site-dumps" description:"Directory containing coebot.tv database dumps" required:"true"`

	DefaultBullet string            `long:"default-bullet" description:"Bullet to convert to the default"`
	BotBullet     map[string]string `long:"bot-bullet" description:"Mapping from bots to their default bullets"`

	TwitchSleep time.Duration `long:"twitch-sleep" description:"Time to require between twitch API calls"`

	Pretty bool `long:"pretty" description:"Pretty print JSON output files"`

	ForceInactive bool `long:"force-inactive" description:"Force all converted channels to be inactive"`
}

// Command returns a fresh conf-convert command.
func Command() cli.Command {
	return &cmd{
		Common: cli.Common{
			Debug: true,
		},
		Twitch:        twitchflags.Default,
		HTTP:          httpflags.Default,
		DefaultBullet: "coebotBot",
		TwitchSleep:   time.Second / 4,
	}
}

func (*cmd) Name() string {
	return "conf-convert"
}

func (cmd *cmd) Main(ctx context.Context, _ []string) {
	loadSiteDB(ctx, cmd.SiteDumps)

	tw = cmd.Twitch.Client(cmd.HTTP.Client())

	ctx = ctxlog.WithOptions(ctx, ctxlog.NoTrace())

	outDir := filepath.Clean(cmd.Out)

	if d, err := os.Stat(outDir); err != nil {
		if os.IsNotExist(err) {
			ctxlog.Fatal(ctx, "output directory does not exist")
		}
		ctxlog.Fatal(ctx, "error stat-ing output directory", ctxlog.PlainError(err))
	} else if !d.IsDir() {
		ctxlog.Fatal(ctx, "output is not a directory")
	}

	todo := make([]string, 0, len(cmd.Positional.Files))

	for _, file := range cmd.Positional.Files {
		file = filepath.Clean(file)
		todo = append(todo, file)
	}

	for _, dir := range cmd.Dir {
		dir = filepath.Clean(dir)

		files, err := os.ReadDir(dir)
		if err != nil {
			ctxlog.Fatal(ctx, "error reading dir", ctxlog.PlainError(err))
		}

		for _, file := range files {
			if ctx.Err() != nil {
				break
			}

			if file.IsDir() {
				continue
			}

			name := file.Name()

			if filepath.Ext(name) != ".json" {
				continue
			}

			filename := filepath.Join(dir, name)
			todo = append(todo, filename)
		}
	}

	if len(todo) == 0 {
		ctxlog.Fatal(ctx, "no files to convert")
	}

	for _, file := range todo {
		if ctx.Err() != nil {
			return
		}

		_, name := filepath.Split(file)
		name = strings.TrimLeft(name, "#")

		n := strings.TrimSuffix(name, ".json")
		out := filepath.Join(outDir, name)

		cmd.processFile(ctx, n, file, out)
	}
}

func (cmd *cmd) processFile(ctx context.Context, name, filename, out string) {
	ctx = ctxlog.With(ctx, zap.String("filename", filename))

	config, ok, err := cmd.convert(ctx, name, filename)
	if err != nil {
		ctxlog.Error(ctx, "error importing config", ctxlog.PlainError(err))
		return
	}

	if !ok {
		return
	}

	if cmd.ForceInactive {
		config.Channel.Active = false
	}

	if err := writeJSON(out, cmd.Pretty, config); err != nil {
		ctxlog.Error(ctx, "error writing config", ctxlog.PlainError(err))
		return
	}

	if err := copyTimes(filename, out); err != nil {
		ctxlog.Error(ctx, "copyTimes", ctxlog.PlainError(err))
	}
}

func writeJSON(out string, pretty bool, v interface{}) error {
	f, err := os.Create(out)
	if err != nil {
		return errors.Wrap(err, "creating file")
	}
	defer f.Close()

	enc := json.NewEncoder(f)

	if pretty {
		enc.SetIndent("", "    ")
	}

	if err := enc.Encode(v); err != nil {
		return errors.Wrap(err, "writing JSON")
	}

	return nil
}

func copyTimes(from, to string) error {
	info, err := os.Stat(from)
	if err != nil {
		return errors.Wrap(err, "stat-ing")
	}

	err = os.Chtimes(to, time.Now(), info.ModTime())
	return errors.Wrap(err, "chtime-ing")
}

func (cmd *cmd) convert(ctx context.Context, expectedName, filename string) (conf *confimport.Config, ok bool, err error) {
	cbConfig := &coeBotConfig{}

	if err := cbConfig.load(filename); err != nil {
		return nil, false, errors.Wrap(err, "loading file")
	}

	var (
		name        string
		displayName string
		twitchID    int64
	)

	if cbConfig.ChannelID == "" {
		name = expectedName
		twitchID, displayName, err = cmd.getChannelByName(ctx, expectedName)
		if err != nil {
			if err == twitch.ErrNotFound {
				ctxlog.Warn(ctx, "user does not exist on twitch, skipping")
				return nil, false, nil
			}
			return nil, false, errors.Wrap(err, "getting channel by name from twitch")
		}
	} else {
		twitchID, err = strconv.ParseInt(cbConfig.ChannelID, 10, 64)
		if err != nil {
			return nil, false, errors.Wrap(err, "parsing channel ID")
		}

		name, displayName, err = cmd.getChannelByID(ctx, twitchID)
		if err != nil {
			if err == twitch.ErrNotFound {
				ctxlog.Warn(ctx, "user does not exist on twitch, skipping")
				return nil, false, nil
			}
			return nil, false, errors.Wrap(err, "getting channel info from twitch")
		}
	}

	ctx = ctxlog.With(ctx, zap.String("channel_name", name))

	botName, active, found := getSiteInfo(expectedName)
	if !found {
		ctxlog.Warn(ctx, "user not found in site database")
		return nil, false, nil
	}

	if botName == name {
		ctxlog.Warn(ctx, "bot, skipping")
		return nil, false, nil
	}

	if name != expectedName {
		ctxlog.Warn(ctx, "name does not match twitch's, converting as inactive with new name and ID", zap.String("expected", expectedName))
		active = false
	}

	defaultBullet := cmd.DefaultBullet
	if len(cmd.BotBullet) > 0 {
		if b := cmd.BotBullet[botName]; b != "" {
			defaultBullet = b
		}
	}

	channel := cbConfig.loadChannel(ctx, defaultBullet, twitchID, name, displayName, botName)
	repInit := time.Unix(channel.TwitchID%60, 0).UTC() // Pick a "random" user-specific initial time.

	config := &confimport.Config{
		Channel:     channel,
		Quotes:      cbConfig.loadQuotes(),
		Commands:    cbConfig.loadCommands(ctx, repInit),
		Autoreplies: cbConfig.loadAutoreplies(ctx),
		Variables:   getVariables(name),
	}

	config.Channel.Active = active

	return config, true, nil
}

type coeBotConfig struct {
	ChannelID     string `json:"channelID"`
	Bullet        string `json:"bullet"`
	CommandPrefix string `json:"commandPrefix"`
	Cooldown      int    `json:"cooldown"`
	Mode          int    `json:"mode"`

	Quotes []struct {
		Editor    *string `json:"editor"`
		Quote     string  `json:"quote"`
		Timestamp *int64  `json:"timestamp"`
	} `json:"quotes"`

	Commands []struct {
		Count       int64   `json:"count"`
		Editor      *string `json:"editor"`
		Key         string  `json:"key"`
		Restriction int     `json:"restriction"`
		Value       string  `json:"value"`
	} `json:"commands"`

	Lists map[string]struct {
		Items       []string `json:"items"`
		Restriction int      `json:"restriction"`
	} `json:"lists"`

	RepeatedCommands []struct {
		Active            bool   `json:"active"`
		Delay             int    `json:"delay"`
		MessageDifference int64  `json:"messageDifference"`
		Name              string `json:"name"`
	} `json:"repeatedCommands"`

	ScheduledCommands []struct {
		Active            bool   `json:"active"`
		MessageDifference int64  `json:"messageDifference"`
		Name              string `json:"name"`
		Pattern           string `json:"pattern"`
	} `json:"scheduledCommands"`

	AutoReplies []struct {
		Response string `json:"response"`
		Trigger  string `json:"trigger"`
	} `json:"autoReplies"`

	OffensiveWords   []string `json:"offensiveWords"`
	ParseYoutube     bool     `json:"parseYoutube"`
	PermittedDomains []string `json:"permittedDomains"`

	Regulars     []string `json:"regulars"`
	Moderators   []string `json:"moderators"`
	Owners       []string `json:"owners"`
	IgnoredUsers []string `json:"ignoredUsers"`

	SubscriberRegulars bool `json:"subscriberRegulars"`
	SubsRegsMinusLinks bool `json:"subsRegsMinusLinks"`

	ShouldModerate  bool `json:"shouldModerate"`
	SignKicks       bool `json:"signKicks"`
	EnableWarnings  bool `json:"enableWarnings"`
	TimeoutDuration int  `json:"timeoutDuration"`

	SteamID            string `json:"steamID"`
	LastFM             string `json:"lastfm"`
	ExtraLifeID        int    `json:"extraLifeID"`
	ClickToTweetFormat string `json:"clickToTweetFormat"`

	SubMessage      string `json:"subMessage"`
	ResubAlert      bool   `json:"resubAlert"`
	ResubMessage    string `json:"resubMessage"`
	SubscriberAlert bool   `json:"subscriberAlert"`

	UrbanEnabled bool `json:"urbanEnabled"`

	RollCooldown int    `json:"rollCooldown"`
	RollDefault  int    `json:"rollDefault"`
	RollLevel    string `json:"rollLevel"`
	RollTimeout  bool   `json:"rollTimeout"`

	UseFilters              bool `json:"useFilters"`
	FilterCaps              bool `json:"filterCaps"`
	FilterCapsMinCapitals   int  `json:"filterCapsMinCapitals"`
	FilterCapsMinCharacters int  `json:"filterCapsMinCharacters"`
	FilterCapsPercent       int  `json:"filterCapsPercent"`
	FilterColors            bool `json:"filterColors"`
	FilterEmotes            bool `json:"filterEmotes"`
	FilterEmotesMax         int  `json:"filterEmotesMax"`
	FilterEmotesSingle      bool `json:"filterEmotesSingle"`
	FilterLinks             bool `json:"filterLinks"`
	FilterMaxLength         int  `json:"filterMaxLength"`
	FilterMe                bool `json:"filterMe"`
	FilterOffensive         bool `json:"filterOffensive"`
	FilterSymbols           bool `json:"filterSymbols"`
	FilterSymbolsMin        int  `json:"filterSymbolsMin"`
	FilterSymbolsPercent    int  `json:"filterSymbolsPercent"`
}

func (c *coeBotConfig) load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrap(err, "opening file")
	}
	defer f.Close()

	return errors.Wrap(jsonx.DecodeSingle(f, c), "decoding JSON")
}

func (c *coeBotConfig) loadChannel(ctx context.Context, defaultBullet string, userID int64, name, displayName string, botName string) *models.Channel {
	channel := modelsx.NewChannel(userID, name, displayName, botName)

	switch c.Bullet {
	case defaultBullet, "":
	default:
		channel.Bullet = null.StringFrom(c.Bullet)
	}

	prefix := c.CommandPrefix
	if utf8.RuneCountInString(prefix) != 1 {
		ctxlog.Warn(ctx, "prefix was not a single character, setting to !", zap.String("commandPrefix", prefix))
		prefix = "!"
	}

	channel.Prefix = prefix
	channel.Mode = c.newMode()
	channel.Ignored = c.IgnoredUsers
	channel.CustomOwners = c.Owners
	channel.CustomMods = c.Moderators
	channel.CustomRegulars = c.Regulars

	channel.Cooldown = null.IntFrom(c.Cooldown)
	channel.ParseYoutube = c.ParseYoutube
	channel.LastFM = c.LastFM
	channel.ExtraLifeID = c.ExtraLifeID
	channel.SteamID = c.SteamID
	channel.UrbanEnabled = c.UrbanEnabled
	channel.Tweet = c.ClickToTweetFormat

	channel.RollLevel = c.rollLevel()
	channel.RollCooldown = c.RollCooldown
	channel.RollDefault = c.RollDefault

	channel.ShouldModerate = c.ShouldModerate
	channel.DisplayWarnings = c.SignKicks
	channel.EnableWarnings = c.EnableWarnings
	channel.TimeoutDuration = c.TimeoutDuration
	channel.EnableFilters = c.UseFilters

	channel.FilterLinks = c.FilterLinks
	channel.PermittedLinks = c.PermittedDomains
	channel.SubsMayLink = c.subsMayLink()

	channel.FilterCaps = c.FilterCaps
	channel.FilterCapsMinChars = c.FilterCapsMinCharacters
	channel.FilterCapsPercentage = c.FilterCapsPercent
	channel.FilterCapsMinCaps = c.FilterCapsMinCapitals

	channel.FilterEmotes = c.FilterEmotes
	channel.FilterEmotesMax = c.FilterEmotesMax
	channel.FilterEmotesSingle = c.FilterEmotesSingle

	channel.FilterSymbols = c.FilterSymbols
	channel.FilterSymbolsPercentage = c.FilterSymbolsPercent
	channel.FilterSymbolsMinSymbols = c.FilterSymbolsMin

	channel.FilterMe = c.FilterMe
	channel.FilterMaxLength = c.FilterMaxLength

	channel.FilterBannedPhrases = c.FilterOffensive
	channel.FilterBannedPhrasesPatterns = c.bannedPhrases(ctx)

	channel.SubMessage = c.SubMessage
	channel.SubMessageEnabled = c.SubscriberAlert
	channel.ResubMessage = c.ResubMessage
	channel.ResubMessageEnabled = c.ResubAlert

	return channel
}

func (c *coeBotConfig) loadQuotes() []*models.Quote {
	quotes := make([]*models.Quote, 0, len(c.Quotes))

	for i, q := range c.Quotes {
		editor := maybeString(q.Editor)
		t := unixMilliPtr(q.Timestamp)

		quote := &models.Quote{
			CreatedAt: t,
			UpdatedAt: t,
			Num:       i + 1,
			Quote:     q.Quote,
			Creator:   editor,
			Editor:    editor,
		}

		quotes = append(quotes, quote)
	}

	return quotes
}

func (c *coeBotConfig) loadCommands(ctx context.Context, repInit time.Time) []*confimport.Command {
	commands := make(map[string]*confimport.Command, len(c.Commands)+len(c.Lists))

	for _, cmd := range c.Commands {
		editor := maybeString(cmd.Editor)

		info := &models.CommandInfo{
			Name:        strings.ToLower(cmd.Key),
			AccessLevel: commandLevelToLevel(cmd.Restriction),
			Count:       cmd.Count,
			Creator:     editor,
			Editor:      editor,
		}

		command := &models.CustomCommand{
			Message: checkCommand(ctx, cmd.Value),
		}

		commands[info.Name] = &confimport.Command{
			Info:          info,
			CustomCommand: command,
		}
	}

	for name, l := range c.Lists {
		name = strings.ToLower(name)

		info := &models.CommandInfo{
			Name:        name,
			AccessLevel: commandLevelToLevel(l.Restriction),
		}

		list := &models.CommandList{
			Items: make([]string, len(l.Items)),
		}

		for i, v := range l.Items {
			list.Items[i] = checkCommand(ctx, v)
		}

		commands[info.Name] = &confimport.Command{
			Info:        info,
			CommandList: list,
		}
	}

	// Try and start repeated commands off with some space between them.
	const offset = 37 * time.Second

	for _, rc := range c.RepeatedCommands {
		name := strings.ToLower(rc.Name)
		ctx := ctxlog.With(ctx, zap.String("name", name))

		command, ok := commands[name]
		if !ok {
			ctxlog.Warn(ctx, "missing command info for repeat")
			continue
		}

		if rc.Delay < 30 {
			ctxlog.Warn(ctx, "delay is under 30 seconds", zap.Int("delay", rc.Delay))
			continue
		}

		diff := rc.MessageDifference
		if diff < 1 {
			ctxlog.Warn(ctx, "message diff is less than 1, converting to 1")
			diff = 1
		}

		command.Repeat = &models.RepeatedCommand{
			Enabled:       rc.Active,
			Delay:         rc.Delay,
			MessageDiff:   diff,
			InitTimestamp: null.TimeFrom(repInit),
		}

		repInit = repInit.Add(offset)
	}

	for _, sc := range c.ScheduledCommands {
		name := strings.ToLower(sc.Name)
		ctx := ctxlog.With(ctx, zap.String("name", name))

		command, ok := commands[name]
		if !ok {
			ctxlog.Warn(ctx, "missing command info for schedule")
			continue
		}

		if _, err := repeat.ParseCron(sc.Pattern); err != nil {
			ctxlog.Warn(ctx, "cron expression did not parse", ctxlog.PlainError(err))
			continue
		}

		diff := sc.MessageDifference
		if diff < 1 {
			ctxlog.Warn(ctx, "message diff is less than 1, converting to 1")
			diff = 1
		}

		command.Schedule = &models.ScheduledCommand{
			Enabled:        sc.Active,
			CronExpression: sc.Pattern,
			MessageDiff:    diff,
		}
	}

	commandList := make([]*confimport.Command, 0, len(commands))

	for _, v := range commands {
		commandList = append(commandList, v)
	}

	sort.Slice(commandList, func(i, j int) bool {
		return commandList[i].Info.Name < commandList[j].Info.Name
	})

	return commandList
}

func (c *coeBotConfig) loadAutoreplies(ctx context.Context) []*models.Autoreply {
	autoreplies := make([]*models.Autoreply, 0, len(c.AutoReplies))

	for i, a := range c.AutoReplies {
		trigger := a.Trigger

		if _, err := regexp.Compile(trigger); err != nil {
			ctxlog.Error(ctx, "failed to compile trigger", ctxlog.PlainError(err), zap.String("trigger", trigger))
			continue
		}

		autoreplies = append(autoreplies, &models.Autoreply{
			Num:      i + 1,
			Trigger:  trigger,
			Response: checkCommand(ctx, a.Response),
		})
	}

	return autoreplies
}

func (c *coeBotConfig) subsMayLink() bool {
	if c.SubscriberRegulars {
		return true
	}
	return !c.SubsRegsMinusLinks
}

func (c *coeBotConfig) newMode() string {
	switch c.Mode {
	case 0:
		return models.AccessLevelBroadcaster
	case 1:
		return models.AccessLevelModerator
	case 2:
		return models.AccessLevelEveryone
	case 3:
		return models.AccessLevelSubscriber
	default:
		// -1 was admin in CoeBot; that mode no longer exists so just use broadcaster.
		return models.AccessLevelBroadcaster
	}
}

func (c *coeBotConfig) rollLevel() string {
	switch c.RollLevel {
	case "everyone":
		return models.AccessLevelEveryone
	case "regulars":
		return models.AccessLevelSubscriber
	case "moderators":
		return models.AccessLevelModerator
	default:
		return models.AccessLevelBroadcaster
	}
}

func (c *coeBotConfig) bannedPhrases(ctx context.Context) []string {
	patterns := make([]string, 0, len(c.OffensiveWords))

	for _, word := range c.OffensiveWords {
		var pattern string
		if strings.HasPrefix(word, "REGEX:") {
			pattern = strings.TrimPrefix(word, "REGEX:")
		} else {
			pattern = regexp.QuoteMeta(word)
		}

		if pattern == "" {
			ctxlog.Error(ctx, "empty pattern")
			continue
		}

		if _, err := regexp.Compile(pattern); err != nil {
			ctxlog.Error(ctx, "error compiling banned phrase", ctxlog.PlainError(err), zap.String("word", word))
			continue
		}

		patterns = append(patterns, pattern)
	}

	return patterns
}

func commandLevelToLevel(l int) string {
	switch l {
	case 3:
		return models.AccessLevelBroadcaster
	case 2:
		return models.AccessLevelModerator
	case 1:
		return models.AccessLevelSubscriber
	case 0:
		return models.AccessLevelEveryone
	default:
		return models.AccessLevelBroadcaster
	}
}

func checkCommand(ctx context.Context, m string) string {
	if _, malformed := cbp.Parse(m); malformed {
		ctxlog.Warn(ctx, "malformed command", zap.String("message", m))
	}
	return m
}
