package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/cmdargs"
	"github.com/hortbot/hortbot/internal/cmdargs/twitchargs"
	"github.com/hortbot/hortbot/internal/confimport"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/repeat"
	"github.com/pkg/errors"
	"github.com/volatiletech/null"
	"go.uber.org/zap"
)

var args = struct {
	cmdargs.Common
	twitchargs.Twitch

	JSONs     string `long:"jsons" description:"Directory of CoeBot JSON config files" required:"true"`
	Out       string `long:"out" description:"Output directory for confimport configs" required:"true"`
	SiteDumps string `long:"site-dumps" description:"Directory containing coebot.tv database dumps" required:"true"`

	DefaultBullet string `long:"default-bullet" description:"Bullet to convert to the default"`

	TwitchSleep time.Duration `long:"twitch-sleep" description:"Time to require between twitch API calls"`
}{
	Common: cmdargs.Common{
		Debug: true,
	},
	Twitch:        twitchargs.DefaultTwitch,
	DefaultBullet: "coebotBot",
	TwitchSleep:   time.Second / 4,
}

func main() {
	cmdargs.Run(&args, mainCtx)
}

func mainCtx(ctx context.Context) {
	logger := ctxlog.FromContext(ctx)
	tw = args.TwitchClient()

	logger = logger.WithOptions(zap.AddStacktrace(noTrace{}))
	ctx = ctxlog.WithLogger(ctx, logger)

	dir := filepath.Clean(args.JSONs)
	outDir := filepath.Clean(args.Out)

	if d, err := os.Stat(outDir); err != nil {
		if os.IsNotExist(err) {
			logger.Fatal("output directory does not exist")
		}
		logger.Fatal("error stat-ing output directory", PlainError(err))
	} else if !d.IsDir() {
		logger.Fatal("output is not a directory")
	}

	files, err := ioutil.ReadDir(args.JSONs)
	if err != nil {
		logger.Fatal("error reading dir", PlainError(err))
	}

	logger.Info("starting site database")
	defer prepareSiteDB(ctx)()
	logger.Info("site database started")

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

		n := strings.TrimSuffix(name, ".json")
		filename := filepath.Join(dir, name)
		out := filepath.Join(outDir, name)

		processFile(ctx, n, filename, out)
	}
}

func processFile(ctx context.Context, name, filename, out string) {
	ctx, logger := ctxlog.FromContextWith(ctx, zap.String("filename", filename))

	config, err := convert(ctx, name, filename)
	if err != nil {
		logger.Error("error importing config", PlainError(err))
		return
	}

	if config == nil {
		return
	}

	f, err := os.Create(out)
	if err != nil {
		logger.Error("error opening output file", PlainError(err))
		return
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(config); err != nil {
		logger.Error("error encoding config", PlainError(err))
	}
}

func convert(ctx context.Context, expectedName, filename string) (*confimport.Config, error) {
	c := &coeBotConfig{}

	if err := c.load(filename); err != nil {
		return nil, errors.Wrap(err, "loading file")
	}

	var (
		err         error
		name        string
		displayName string
		twitchID    int64
	)

	if c.ChannelID == "" {
		name = expectedName
		twitchID, displayName, err = getChannelbyName(ctx, expectedName)
		if err != nil {
			if err == twitch.ErrNotFound {
				ctxlog.FromContext(ctx).Warn("user does not exist on twitch, skipping")
				return nil, nil
			}
			return nil, errors.Wrap(err, "getting channel by name from twitch")
		}
	} else {
		twitchID, err = strconv.ParseInt(c.ChannelID, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "parsing channel ID")
		}

		name, displayName, err = getChannelByID(ctx, twitchID)
		if err != nil {
			if err == twitch.ErrNotFound {
				ctxlog.FromContext(ctx).Warn("user does not exist on twitch, skipping")
				return nil, nil
			}
			return nil, errors.Wrap(err, "getting channel info from twitch")
		}
	}

	ctx, logger := ctxlog.FromContextWith(ctx, zap.String("channel_name", name))

	botName, active, err := getSiteInfo(ctx, expectedName)
	if err != nil {
		return nil, errors.Wrapf(err, "querying site channel info for %s", expectedName)
	}

	if botName == name {
		logger.Warn("bot, skipping")
		return nil, nil
	}

	if name != expectedName {
		logger.Warn("name does not match twitch's, converting as inactive with new name and ID", zap.String("expected", expectedName))
		active = false
	}

	config := &confimport.Config{
		Channel:     c.loadChannel(ctx, twitchID, name, displayName, botName),
		Quotes:      c.loadQuotes(),
		Commands:    c.loadCommands(ctx),
		Autoreplies: c.loadAutoreplies(ctx),
	}

	config.Channel.Active = active

	config.Variables, err = getVariables(ctx, name)
	if err != nil {
		return nil, errors.Wrap(err, "loading variables")
	}

	return config, nil
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

	return errors.Wrap(json.NewDecoder(f).Decode(c), "decoding JSON")
}

func (c *coeBotConfig) loadChannel(ctx context.Context, userID int64, name, displayName string, botName string) *models.Channel {
	channel := modelsx.NewChannel(userID, name, displayName, botName)

	switch c.Bullet {
	case args.DefaultBullet, "":
	default:
		channel.Bullet = null.StringFrom(c.Bullet)
	}

	channel.Prefix = c.CommandPrefix
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

func (c *coeBotConfig) loadCommands(ctx context.Context) []*confimport.Command {
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
	init := time.Unix(0, 0).UTC()
	const offset = 37 * time.Second

	for _, rc := range c.RepeatedCommands {
		name := strings.ToLower(rc.Name)
		_, logger := ctxlog.FromContextWith(ctx, zap.String("name", name))

		command, ok := commands[name]
		if !ok {
			logger.Warn("missing command info for repeat")
			continue
		}

		if rc.Delay < 30 {
			logger.Warn("delay is under 30 seconds", zap.Int("delay", rc.Delay))
			continue
		}

		diff := rc.MessageDifference
		if diff < 1 {
			logger.Warn("message diff is less than 1, converting to 1")
			diff = 1
		}

		command.Repeat = &models.RepeatedCommand{
			Enabled:       rc.Active,
			Delay:         rc.Delay,
			MessageDiff:   diff,
			InitTimestamp: null.TimeFrom(init),
		}

		init = init.Add(offset)
	}

	for _, sc := range c.ScheduledCommands {
		name := strings.ToLower(sc.Name)
		_, logger := ctxlog.FromContextWith(ctx, zap.String("name", name))

		command, ok := commands[name]
		if !ok {
			logger.Warn("missing command info for schedule")
			continue
		}

		if _, err := repeat.ParseCron(sc.Pattern); err != nil {
			logger.Warn("cron expression did not parse", PlainError(err))
			continue
		}

		diff := sc.MessageDifference
		if diff < 1 {
			logger.Warn("message diff is less than 1, converting to 1")
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
			ctxlog.FromContext(ctx).Error("failed to compile trigger", PlainError(err), zap.String("trigger", trigger))
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
		panic(fmt.Sprintf("bad mode %d", c.Mode))
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
			ctxlog.FromContext(ctx).Error("empty pattern")
			continue
		}

		if _, err := regexp.Compile(pattern); err != nil {
			ctxlog.FromContext(ctx).Error("error compiling banned phrase", PlainError(err), zap.String("word", word))
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
		ctxlog.FromContext(ctx).Warn("malformed command", zap.String("message", m))
	}
	return m
}
