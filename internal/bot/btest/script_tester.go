// Package btest implements a script test system for the bot package.
package btest

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"math/rand"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/alicebob/miniredis/v2"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/botmocks"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/extralife/extralifemocks"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/hltb/hltbmocks"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/lastfm/lastfmmocks"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/simple/simplemocks"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/steam/steammocks"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/tinyurl/tinyurlmocks"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/twitchmocks"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/urban/urbanmocks"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/xkcd/xkcdmocks"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/youtube/youtubemocks"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"github.com/leononame/clock"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/zikaeroh/ctxlog"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

// RunScript runs the a single script test.
func RunScript(t testing.TB, filename string, freshDB func(t testing.TB) *sql.DB) {
	db := freshDB(t)
	defer db.Close()

	st := scriptTester{
		filename: filename,
		db:       db,
	}

	st.test(t)
}

const (
	countSend                  = "send"
	countNotifyEventsubUpdates = "notify_eventsub_updates"
)

type scriptTester struct {
	filename string

	db                     *sql.DB
	redis                  *miniredis.Miniredis
	sender                 *SenderMock
	eventsubUpdateNotifier *botmocks.EventsubUpdateNotifierMock
	clock                  *clock.Mock

	lastFM    *lastfmmocks.APIMock
	youtube   *youtubemocks.APIMock
	xkcd      *xkcdmocks.APIMock
	extraLife *extralifemocks.APIMock
	twitch    *twitchmocks.APIMock
	steam     *steammocks.APIMock
	tinyURL   *tinyurlmocks.APIMock
	urban     *urbanmocks.APIMock
	simple    *simplemocks.APIMock
	hltb      *hltbmocks.APIMock

	bc bot.Config
	b  *bot.Bot

	counts   map[string]int
	idToName map[int64]string

	ctx         context.Context
	currLine    string
	currLineNum int
	actions     []action

	sentBefore int
	needNoSend bool

	notifyEventsubUpdatesCallsBefore int
	needNoNotifyEventsubUpdatesCalls bool
	needPanic                        bool
}

type action struct {
	number int
	line   string
	fn     func(context.Context)
}

func (st *scriptTester) addAction(fn func(context.Context)) {
	st.actions = append(st.actions, action{
		number: st.currLineNum,
		line:   st.currLine,
		fn:     fn,
	})
}

func (st *scriptTester) ensureBot(ctx context.Context, t testing.TB) {
	if st.b == nil {
		st.b = bot.New(&st.bc)
		assert.NilError(t, st.b.Init(ctx))
	}
}

func (st *scriptTester) test(t testing.TB) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic: %v", r)
			t.Logf("%s", debug.Stack())
			t.FailNow()
		}
	}()

	st.counts = make(map[string]int)
	st.idToName = make(map[int64]string)
	st.sender = &SenderMock{
		SendMessageFunc: func(ctx context.Context, origin, target, message string) error { return nil },
	}
	st.eventsubUpdateNotifier = &botmocks.EventsubUpdateNotifierMock{
		NotifyEventsubUpdatesFunc: func(ctx context.Context) error { return nil },
	}
	st.clock = clock.NewMock()
	st.lastFM = &lastfmmocks.APIMock{}
	st.youtube = &youtubemocks.APIMock{}
	st.xkcd = &xkcdmocks.APIMock{}
	st.extraLife = &extralifemocks.APIMock{}
	st.twitch = &twitchmocks.APIMock{
		SendChatMessageFunc: func(ctx context.Context, broadcasterID, modID int64, modToken *oauth2.Token, message string) (*oauth2.Token, error) {
			if modID == 0 {
				panic("modID is 0")
			}

			// Backwards compat hack; make chat message API pretend to be the old API.
			origin := st.idToName[modID]
			if origin == "" {
				origin = "@" + strconv.FormatInt(modID, 10)
			}
			target := "#" + st.idToName[broadcasterID]
			if target == "#" {
				target = "@" + strconv.FormatInt(broadcasterID, 10)
			}
			return nil, st.sender.SendMessage(ctx, origin, target, message)
		},
	}
	st.steam = &steammocks.APIMock{}
	st.tinyURL = &tinyurlmocks.APIMock{}
	st.urban = &urbanmocks.APIMock{}
	st.simple = &simplemocks.APIMock{}
	st.hltb = &hltbmocks.APIMock{}

	st.ctx = ctxlog.WithLogger(context.Background(), testutil.Logger(t))

	rServer, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer rCleanup()

	st.redis = rServer

	st.bc = bot.Config{
		DB:                     st.db,
		Redis:                  redis.New(rClient),
		EventsubUpdateNotifier: st.eventsubUpdateNotifier,
		Clock:                  st.clock,
		LastFM:                 st.lastFM,
		YouTube:                st.youtube,
		XKCD:                   st.xkcd,
		ExtraLife:              st.extraLife,
		Twitch:                 st.twitch,
		Steam:                  st.steam,
		TinyURL:                st.tinyURL,
		Urban:                  st.urban,
		Simple:                 st.simple,
		HLTB:                   st.hltb,
		NoDedupe:               true,
		PublicJoin:             true,
	}

	st.clock.Set(time.Now())

	f, err := os.Open(st.filename)
	assert.NilError(t, err)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++

		line := scanner.Text()
		line = strings.TrimLeftFunc(line, unicode.IsSpace)
		st.currLine = line
		st.currLineNum = lineNum

		if line == "" || line[0] == '#' {
			continue
		}

		var directive string
		var args string

		i := strings.IndexByte(line, ' ')
		if i > 0 {
			directive = line[:i]
			args = strings.TrimLeftFunc(line[i+1:], unicode.IsSpace)
		} else {
			directive = line
		}

		fn := directiveFuncs[directive]
		if fn == nil {
			t.Fatalf("line %d: unknown directive %s", lineNum, directive)
		}

		fn(st, t, directive, args, lineNum)
	}

	assert.NilError(t, scanner.Err())

	if len(st.actions) == 0 {
		t.Error("no actions were provided")
		t.FailNow()
	}

	defer func() {
		if st.b != nil {
			st.b.Stop() // Inside its on func, as st.b is set inside an action.
		}
	}()

	for _, action := range st.actions {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if st.needPanic {
						st.needPanic = false
						return
					}

					panic(r)
				}
			}()

			t.Logf("line %d: %s", action.number, action.line)
			action.fn(st.ctx)
		}()
	}

	assert.Equal(t, len(st.sender.SendMessageCalls()), st.counts[countSend])
	assert.Equal(t, len(st.eventsubUpdateNotifier.NotifyEventsubUpdatesCalls()), st.counts[countNotifyEventsubUpdates])
}

func (st *scriptTester) skip(t testing.TB, _, reason string, lineNum int) {
	if reason == "" {
		t.Skipf("line %d", lineNum)
	} else {
		t.Skipf("line %d: %s", lineNum, reason)
	}
}

func (st *scriptTester) boilDebug(t testing.TB, _, _ string, _ int) {
	st.addAction(func(_ context.Context) {
		st.ctx = boil.WithDebug(st.ctx, true)
		st.ctx = boil.WithDebugWriter(st.ctx, testutil.Writer{T: t})
	})
}

func (st *scriptTester) botConfig(t testing.TB, _, args string, lineNum int) {
	assert.Assert(t, st.b == nil, "bot has already been created, cannot configure")

	var bcj struct {
		*bot.Config

		Clock string
		Rand  *int
	}

	bcj.Config = &st.bc

	assert.NilError(t, json.Unmarshal([]byte(args), &bcj), "line %d", lineNum)

	switch bcj.Clock {
	case "real":
		st.bc.Clock = clock.New()

	case "", "mock":
		st.bc.Clock = st.clock

	default:
		t.Fatalf("line %d: unknown clock type %s", lineNum, bcj.Clock)
	}

	if bcj.Rand != nil {
		rng := rand.New(rand.NewSource(int64(*bcj.Rand))) //nolint:gosec

		fakeRand := &botmocks.RandMock{}
		fakeRand.IntnFunc = rng.Intn
		fakeRand.Float64Func = rng.Float64

		st.bc.Rand = fakeRand

		st.redis.Seed(*bcj.Rand)
	}
}

func (st *scriptTester) checkpoint(_ testing.TB, _, _ string, _ int) {
	st.addAction(func(ctx context.Context) {
		st.doCheckpoint()
	})
}

func (st *scriptTester) doCheckpoint() {
	st.sentBefore = len(st.sender.SendMessageCalls())
	st.notifyEventsubUpdatesCallsBefore = len(st.eventsubUpdateNotifier.NotifyEventsubUpdatesCalls())
}

func (st *scriptTester) dumpRedis(t testing.TB, _, _ string, lineNum int) {
	st.addAction(func(ctx context.Context) {
		t.Logf("line %d:\n%s", lineNum, st.redis.Dump())
	})
}

func (st *scriptTester) shouldPanic(t testing.TB, _ string, _ string, lineNum int) {
	st.addAction(func(ctx context.Context) {
		st.needPanic = true
	})
}

var directiveFuncs = map[string]func(st *scriptTester, t testing.TB, directive, args string, lineNum int){
	"skip":                          (*scriptTester).skip,
	"boil_debug":                    (*scriptTester).boilDebug,
	"bot_config":                    (*scriptTester).botConfig,
	"dump_redis":                    (*scriptTester).dumpRedis,
	"insert_channel":                (*scriptTester).insertChannel,
	"insert_custom_command":         (*scriptTester).insertCustomCommand,
	"insert_repeated_command":       (*scriptTester).insertRepeatedCommand,
	"insert_scheduled_command":      (*scriptTester).insertScheduledCommand,
	"insert_command_info":           (*scriptTester).insertCommandInfo,
	"upsert_twitch_token":           (*scriptTester).upsertTwitchToken,
	"delete_twitch_token":           (*scriptTester).deleteTwitchToken,
	"checkpoint":                    (*scriptTester).checkpoint,
	"handle":                        (*scriptTester).handle,
	"handle_me":                     (*scriptTester).handle,
	"send":                          (*scriptTester).send,
	"send_match":                    (*scriptTester).sendMatch,
	"send_any":                      (*scriptTester).sendAny,
	"no_send":                       (*scriptTester).noSend,
	"notify_eventsub_updates":       (*scriptTester).notifyEventsubUpdatesCalls,
	"no_notify_eventsub_updates":    (*scriptTester).noNotifyEventsubUpdatesCalls,
	"clock_forward":                 (*scriptTester).clockForward,
	"clock_set":                     (*scriptTester).clockSet,
	"sleep":                         (*scriptTester).sleep,
	"join":                          (*scriptTester).join,
	"should_panic":                  (*scriptTester).shouldPanic,
	"no_lastfm":                     (*scriptTester).noLastFM,
	"lastfm_recent_tracks":          (*scriptTester).lastFMRecentTracks,
	"no_youtube":                    (*scriptTester).noYouTube,
	"youtube_video_titles":          (*scriptTester).youtubeVideoTitles,
	"no_xkcd":                       (*scriptTester).noXKCD,
	"xkcd_comics":                   (*scriptTester).xkcdComics,
	"no_extra_life":                 (*scriptTester).noExtraLife,
	"extra_life_amounts":            (*scriptTester).extraLifeAmounts,
	"twitch_get_channel_by_id":      (*scriptTester).twitchGetChannelByID,
	"twitch_get_user_by_username":   (*scriptTester).twitchGetUserByUsername,
	"no_steam":                      (*scriptTester).noSteam,
	"steam_get_player_summary":      (*scriptTester).steamGetPlayerSummary,
	"steam_get_owned_games":         (*scriptTester).steamGetOwnedGames,
	"no_tiny_url":                   (*scriptTester).noTinyURL,
	"tiny_url_shorten":              (*scriptTester).tinyURLShorten,
	"no_urban":                      (*scriptTester).noUrban,
	"urban_define":                  (*scriptTester).urbanDefine,
	"simple_plaintext":              (*scriptTester).simplePlaintext,
	"hltb_search":                   (*scriptTester).hltbSearch,
	"twitch_modify_channel":         (*scriptTester).twitchModifyChannel,
	"twitch_get_game_by_name":       (*scriptTester).twitchGetGameByName,
	"twitch_get_game_by_id":         (*scriptTester).twitchGetGameByID,
	"twitch_search_categories":      (*scriptTester).twitchSearchCategories,
	"twitch_get_stream_by_user_id":  (*scriptTester).twitchGetStreamByUserID,
	"twitch_get_stream_by_username": (*scriptTester).twitchGetStreamByUsername,
	"twitch_get_game_links":         (*scriptTester).twitchGetGameLinks,
	"twitch_ban":                    (*scriptTester).twitchBan,
	"twitch_unban":                  (*scriptTester).twitchUnban,
	"twitch_update_chat_settings":   (*scriptTester).twitchUpdateChatSettings,
	"twitch_set_chat_color":         (*scriptTester).twitchSetChatColor,
	"twitch_delete_chat_message":    (*scriptTester).twitchDeleteChatMessage,
	"twitch_clear_chat":             (*scriptTester).twitchClearChat,
	"twitch_announce":               (*scriptTester).twitchAnnounce,
}
