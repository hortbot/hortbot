package btest

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"math/rand"
	"os"
	"runtime/debug"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/alicebob/miniredis/v2"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/botfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/extralife/extralifefakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm/lastfmfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch/twitchfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/xkcd/xkcdfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/youtube/youtubefakes"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/dedupe"
	dedupemem "github.com/hortbot/hortbot/internal/pkg/dedupe/memory"
	"github.com/hortbot/hortbot/internal/pkg/rdb"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"github.com/leononame/clock"
	"github.com/volatiletech/sqlboiler/boil"
	"gotest.tools/assert"
)

func RunScript(t testing.TB, filename string, freshDB func(t testing.TB) (*sql.DB, func())) {
	db, undb := freshDB(t)
	defer undb()

	st := scriptTester{
		filename: filename,
		db:       db,
	}

	st.test(t)
}

const (
	countSend                 = "send"
	countNotifyChannelUpdates = "notify_channel_updates"
)

type scriptTester struct {
	filename string
	lineNum  int

	db       *sql.DB
	redis    *miniredis.Miniredis
	sender   *botfakes.FakeSender
	notifier *botfakes.FakeNotifier
	clock    *clock.Mock

	lastFM    *lastfmfakes.FakeAPI
	youtube   *youtubefakes.FakeAPI
	xkcd      *xkcdfakes.FakeAPI
	extraLife *extralifefakes.FakeAPI
	twitch    *twitchfakes.FakeAPI

	bc bot.Config
	b  *bot.Bot

	counts map[string]int

	actions  []func(context.Context)
	cleanups []func()

	sentBefore int
	needNoSend bool

	notifyChannelUpdatesBefore int
	needNoNotifyChannelUpdates bool
}

func (st *scriptTester) addAction(fn func(context.Context)) {
	st.actions = append(st.actions, fn)
}

func (st *scriptTester) addCleanup(fn func()) {
	st.cleanups = append(st.cleanups, fn)
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
	st.sender = &botfakes.FakeSender{}
	st.notifier = &botfakes.FakeNotifier{}
	st.clock = clock.NewMock()
	st.lastFM = newFakeLastFM(t)
	st.youtube = newFakeYouTube(t)
	st.xkcd = newFakeXKCD(t)
	st.extraLife = newFakeExtraLife(t)
	st.twitch = newFakeTwitch(t)

	defer func() {
		for _, cleanup := range st.cleanups {
			defer cleanup()
		}
	}()

	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(t))

	rServer, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer rCleanup()

	rDB, err := rdb.New(rClient)
	assert.NilError(t, err)

	st.redis = rServer

	st.bc = bot.Config{
		DB:        st.db,
		RDB:       rDB,
		Dedupe:    dedupe.NeverSeen,
		Sender:    st.sender,
		Notifier:  st.notifier,
		Clock:     st.clock,
		LastFM:    st.lastFM,
		YouTube:   st.youtube,
		XKCD:      st.xkcd,
		ExtraLife: st.extraLife,
		Twitch:    st.twitch,
	}

	st.clock.Set(time.Now())

	f, err := os.Open(st.filename)
	assert.NilError(t, err)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		st.lineNum++

		line := scanner.Text()
		line = strings.TrimLeftFunc(line, unicode.IsSpace)

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
			t.Fatalf("line %d: unknown directive %s", st.lineNum, directive)
		}

		fn(st, t, directive, args)
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
		action(ctx)
	}

	assert.Equal(t, st.sender.SendMessageCallCount(), st.counts[countSend])
	assert.Equal(t, st.notifier.NotifyChannelUpdatesCallCount(), st.counts[countNotifyChannelUpdates])
}

func (st *scriptTester) skip(t testing.TB, _, reason string) {
	lineNum := st.lineNum

	if reason == "" {
		t.Skipf("line %d", lineNum)
	} else {
		t.Skipf("line %d: %s", lineNum, reason)
	}
}

func (st *scriptTester) boilDebug(t testing.TB, _, _ string) {
	oldMode := boil.DebugMode
	oldWriter := boil.DebugWriter

	boil.DebugMode = true
	boil.DebugWriter = testutil.Writer{T: t}

	st.addCleanup(func() {
		boil.DebugMode = oldMode
		boil.DebugWriter = oldWriter
	})
}

func (st *scriptTester) botConfig(t testing.TB, _, args string) {
	lineNum := st.lineNum

	assert.Assert(t, st.b == nil, "bot has already been created, cannot configure")

	var bcj struct {
		*bot.Config

		Dedupe string
		Clock  string
		Rand   *int
	}

	bcj.Config = &st.bc

	assert.NilError(t, json.Unmarshal([]byte(args), &bcj), "line %d", lineNum)

	switch bcj.Dedupe {
	case "", "never":
		st.bc.Dedupe = dedupe.NeverSeen

	case "memory":
		d := dedupemem.New(time.Minute, 5*time.Minute)
		st.addCleanup(d.Stop)
		st.bc.Dedupe = d

	default:
		t.Fatalf("line %d: unknown dedupe type %s", lineNum, bcj.Dedupe)
	}

	switch bcj.Clock {
	case "real":
		st.bc.Clock = clock.New()

	case "", "mock":
		st.bc.Clock = st.clock

	default:
		t.Fatalf("line %d: unknown clock type %s", lineNum, bcj.Clock)
	}

	if bcj.Rand != nil {
		rng := rand.New(rand.NewSource(int64(*bcj.Rand)))

		fakeRand := newFakeRand(t)
		fakeRand.IntnCalls(rng.Intn)
		fakeRand.Float64Calls(rng.Float64)

		st.bc.Rand = fakeRand

		st.redis.Seed(*bcj.Rand)
	}
}

func (st *scriptTester) checkpoint(_ testing.TB, _, _ string) {
	st.addAction(func(ctx context.Context) {
		st.doCheckpoint()
	})
}

func (st *scriptTester) doCheckpoint() {
	st.sentBefore = st.sender.SendMessageCallCount()
	st.notifyChannelUpdatesBefore = st.notifier.NotifyChannelUpdatesCallCount()
}

func (st *scriptTester) dumpRedis(t testing.TB, _, _ string) {
	lineNum := st.lineNum

	st.addAction(func(ctx context.Context) {
		t.Logf("line %d:\n%s", lineNum, st.redis.Dump())
	})
}

var directiveFuncs = map[string]func(st *scriptTester, t testing.TB, directive, args string){
	"skip":                       (*scriptTester).skip,
	"boil_debug":                 (*scriptTester).boilDebug,
	"bot_config":                 (*scriptTester).botConfig,
	"dump_redis":                 (*scriptTester).dumpRedis,
	"insert_channel":             (*scriptTester).insertChannel,
	"insert_custom_command":      (*scriptTester).insertCustomCommand,
	"insert_repeated_command":    (*scriptTester).insertRepeatedCommand,
	"insert_scheduled_command":   (*scriptTester).insertScheduledCommand,
	"insert_command_info":        (*scriptTester).insertCommandInfo,
	"upsert_twitch_token":        (*scriptTester).upsertTwitchToken,
	"checkpoint":                 (*scriptTester).checkpoint,
	"handle":                     (*scriptTester).handle,
	"handle_me":                  (*scriptTester).handle,
	"send":                       (*scriptTester).send,
	"send_match":                 (*scriptTester).sendMatch,
	"send_any":                   (*scriptTester).sendAny,
	"no_send":                    (*scriptTester).noSend,
	"notify_channel_updates":     (*scriptTester).notifyChannelUpdates,
	"no_notify_channel_updates":  (*scriptTester).noNotifyChannelUpdates,
	"clock_forward":              (*scriptTester).clockForward,
	"clock_set":                  (*scriptTester).clockSet,
	"sleep":                      (*scriptTester).sleep,
	"join":                       (*scriptTester).join,
	"no_lastfm":                  (*scriptTester).noLastFM,
	"lastfm_recent_tracks":       (*scriptTester).lastFMRecentTracks,
	"no_youtube":                 (*scriptTester).noYouTube,
	"youtube_video_titles":       (*scriptTester).youtubeVideoTitles,
	"no_xkcd":                    (*scriptTester).noXKCD,
	"xkcd_comics":                (*scriptTester).xkcdComics,
	"no_extra_life":              (*scriptTester).noExtraLife,
	"extra_life_amounts":         (*scriptTester).extraLifeAmounts,
	"twitch_get_channel_by_id":   (*scriptTester).twitchGetChannelByID,
	"twitch_set_channel_status":  (*scriptTester).twitchSetChannel,
	"twitch_set_channel_game":    (*scriptTester).twitchSetChannel,
	"twitch_get_current_stream":  (*scriptTester).twitchGetCurrentStream,
	"twitch_get_chatters":        (*scriptTester).twitchGetChatters,
	"twitch_get_id_for_username": (*scriptTester).twitchGetIDForUsername,
}
