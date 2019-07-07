package bot_test

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/alicebob/miniredis"
	"github.com/bmatcuk/doublestar"
	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/botfakes"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/dedupe"
	dedupemem "github.com/hortbot/hortbot/internal/pkg/dedupe/memory"
	"github.com/hortbot/hortbot/internal/pkg/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/lastfm/lastfmfakes"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"github.com/jakebailey/irc"
	"github.com/leononame/clock"
	"github.com/volatiletech/sqlboiler/boil"
	"gotest.tools/assert"
)

func TestScripts(t *testing.T) {
	files, err := doublestar.Glob(filepath.Join("testdata", "script", "**", "*.txt"))
	assert.NilError(t, err)
	assert.Assert(t, len(files) != 0)

	prefix := filepath.Join("testdata", "script")

	for _, file := range files {
		file := file
		name := strings.TrimSuffix(strings.TrimPrefix(file, prefix)[1:], ".txt")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			st := scriptTester{filename: file}
			st.test(t)
		})
	}
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
	lastFM   *lastfmfakes.FakeAPI

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

func (st *scriptTester) ensureBot(ctx context.Context, t *testing.T) {
	if st.b == nil {
		st.b = bot.New(&st.bc)
		assert.NilError(t, st.b.Init(ctx))
	}
}

//nolint:gocyclo
func (st *scriptTester) test(t *testing.T) {
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
	st.lastFM = &lastfmfakes.FakeAPI{}

	defer func() {
		for _, cleanup := range st.cleanups {
			defer cleanup()
		}
	}()

	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(t))

	rServer, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(t, err)
	defer rCleanup()

	st.redis = rServer

	db, undb := freshDB(t)
	defer undb()

	st.db = db

	st.bc = bot.Config{
		DB:       db,
		Redis:    rClient,
		Dedupe:   dedupe.NeverSeen,
		Sender:   st.sender,
		Notifier: st.notifier,
		LastFM:   st.lastFM,
	}

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

		switch directive {
		case "skip":
			st.skip(t, args)

		case "boil_debug":
			st.boilDebug(t)

		case "bot_config":
			st.botConfig(t, args)

		case "insert_channel":
			st.insertChannel(t, args)

		case "insert_simple_command":
			st.insertSimpleCommand(t, args)

		case "insert_repeated_command":
			st.insertRepeatedCommand(t, args)

		case "insert_scheduled_command":
			st.insertScheduledCommand(t, args)

		case "checkpoint":
			st.checkpoint()

		case "handle":
			st.handle(t, args, false)

		case "handle_me":
			st.handle(t, args, true)

		case "send":
			st.send(t, args)

		case "send_match":
			st.sendMatch(t, args)

		case "send_any":
			st.sendAny(t)

		case "no_send":
			st.noSend(t)

		case "notify_channel_updates":
			st.notifyChannelUpdates(t, args)

		case "no_notify_channel_updates":
			st.noNotifyChannelUpdates(t)

		case "clock_forward":
			st.clockForward(t, args)

		case "clock_set":
			st.clockSet(t, args)

		case "join":
			st.join(t, args)

		case "sleep":
			st.sleep(t, args)

		case "no_lastfm":
			st.noLastFM(t)

		case "lastfm_recent_tracks":
			st.lastFMRecentTracks(t, args)

		default:
			t.Fatalf("line %d: unknown directive %s", st.lineNum, directive)
		}
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

func (st *scriptTester) skip(t *testing.T, reason string) {
	lineNum := st.lineNum

	if reason == "" {
		t.Skipf("line %d", lineNum)
	} else {
		t.Skipf("line %d: %s", lineNum, reason)
	}
}

func (st *scriptTester) boilDebug(t *testing.T) {
	oldMode := boil.DebugMode
	oldWriter := boil.DebugWriter

	boil.DebugMode = true
	boil.DebugWriter = testutil.Writer{T: t}

	st.addCleanup(func() {
		boil.DebugMode = oldMode
		boil.DebugWriter = oldWriter
	})
}

func (st *scriptTester) botConfig(t *testing.T, args string) {
	lineNum := st.lineNum

	assert.Assert(t, st.b == nil, "bot has already been created, cannot configure")

	var bcj struct {
		*bot.Config

		Dedupe string
		Clock  string
		Rand   *int64
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
	case "", "real":
		st.bc.Clock = clock.New()

	case "mock":
		st.bc.Clock = st.clock

	default:
		t.Fatalf("line %d: unknown clock type %s", lineNum, bcj.Clock)
	}

	if bcj.Rand != nil {
		rng := rand.New(rand.NewSource(*bcj.Rand))

		fakeRand := &botfakes.FakeRand{}
		fakeRand.IntnCalls(rng.Intn)

		st.bc.Rand = fakeRand
	}
}

func (st *scriptTester) insertChannel(t *testing.T, args string) {
	lineNum := st.lineNum

	var channel models.Channel
	assert.NilError(t, json.Unmarshal([]byte(args), &channel), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		assert.NilError(t, channel.Insert(ctx, st.db, boil.Infer()), "line %d", lineNum)
	})
}

func (st *scriptTester) insertSimpleCommand(t *testing.T, args string) {
	lineNum := st.lineNum

	var sc models.SimpleCommand
	assert.NilError(t, json.Unmarshal([]byte(args), &sc), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		assert.NilError(t, sc.Insert(ctx, st.db, boil.Infer()), "line %d", lineNum)
	})
}

func (st *scriptTester) insertRepeatedCommand(t *testing.T, args string) {
	lineNum := st.lineNum

	var rc models.RepeatedCommand
	assert.NilError(t, json.Unmarshal([]byte(args), &rc), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		ctx = boil.SkipTimestamps(ctx)
		assert.NilError(t, rc.Insert(ctx, st.db, boil.Infer()), "line %d", lineNum)
	})
}

func (st *scriptTester) insertScheduledCommand(t *testing.T, args string) {
	lineNum := st.lineNum

	var sc models.ScheduledCommand
	assert.NilError(t, json.Unmarshal([]byte(args), &sc), "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		ctx = boil.SkipTimestamps(ctx)
		assert.NilError(t, sc.Insert(ctx, st.db, boil.Infer()), "line %d", lineNum)
	})
}

func (st *scriptTester) checkpoint() {
	st.addAction(func(ctx context.Context) {
		st.doCheckpoint()
	})
}

func (st *scriptTester) doCheckpoint() {
	st.sentBefore = st.sender.SendMessageCallCount()
	st.notifyChannelUpdatesBefore = st.notifier.NotifyChannelUpdatesCallCount()
}

func (st *scriptTester) handle(t *testing.T, directiveArgs string, me bool) {
	lineNum := st.lineNum

	if st.needNoSend {
		st.noSend(t)
	}

	if st.needNoNotifyChannelUpdates {
		st.noNotifyChannelUpdates(t)
	}

	st.needNoSend = true
	st.needNoNotifyChannelUpdates = true

	args := strings.SplitN(directiveArgs, " ", 2)
	assert.Assert(t, len(args) == 2, "line %d", lineNum)

	origin := args[0]
	mRaw := args[1]

	var m *irc.Message

	if mRaw != "nil" {
		u := uuid.Must(uuid.NewV4())
		mRaw = strings.ReplaceAll(mRaw, "__UUID__", u.String())

		var err error
		m, err = irc.ParseMessage(mRaw)
		assert.NilError(t, err, "line %d", lineNum)

		if me {
			m.Trailing, _ = irc.EncodeCTCP("ACTION", m.Trailing)
		}
	}

	st.handleM(t, origin, m)
}

func (st *scriptTester) handleM(t *testing.T, origin string, m *irc.Message) {
	st.addAction(func(ctx context.Context) {
		st.ensureBot(ctx, t)
		st.doCheckpoint()
		st.b.Handle(ctx, origin, m)
	})
}

func (st *scriptTester) send(t *testing.T, args string) {
	lineNum := st.lineNum

	callNum := st.counts[countSend]
	st.counts[countSend]++

	sent := strings.SplitN(args, " ", 3)
	assert.Assert(t, len(sent) == 3, "line %d", lineNum)

	st.addAction(func(context.Context) {
		assert.Assert(t, st.sender.SendMessageCallCount() > callNum, "SendMessage not called: line %d", lineNum)
		origin, target, message := st.sender.SendMessageArgsForCall(callNum)
		assert.Equal(t, origin, sent[0], "line %d", lineNum)
		assert.Equal(t, target, sent[1], "line %d", lineNum)
		assert.Equal(t, message, sent[2], "line %d", lineNum)
	})

	st.needNoSend = false
}

func (st *scriptTester) sendMatch(t *testing.T, args string) {
	lineNum := st.lineNum

	callNum := st.counts[countSend]
	st.counts[countSend]++

	sent := strings.SplitN(args, " ", 3)
	assert.Assert(t, len(sent) == 3, "line %d", lineNum)

	pattern, err := regexp.Compile(sent[2])
	assert.NilError(t, err)

	st.addAction(func(context.Context) {
		assert.Assert(t, st.sender.SendMessageCallCount() > callNum, "SendMessage not called: line %d", lineNum)
		origin, target, message := st.sender.SendMessageArgsForCall(callNum)
		assert.Equal(t, origin, sent[0], "line %d", lineNum)
		assert.Equal(t, target, sent[1], "line %d", lineNum)
		assert.Assert(t, pattern.MatchString(message), "pattern=`%s`, message=`%s`: line %d", pattern, message, lineNum)
	})

	st.needNoSend = false
}

func (st *scriptTester) noSend(t *testing.T) {
	lineNum := st.lineNum

	st.addAction(func(context.Context) {
		sentAfter := st.sender.SendMessageCallCount()

		if st.sentBefore != sentAfter {
			origin, target, message := st.sender.SendMessageArgsForCall(sentAfter - 1)
			t.Errorf("sent message: origin=%s, target=%s, message=%s: line %d", origin, target, message, lineNum)
			t.FailNow()
		}
	})

	st.needNoSend = false
}

func (st *scriptTester) sendAny(t *testing.T) {
	lineNum := st.lineNum

	callNum := st.counts["send"]
	st.counts["send"]++

	st.addAction(func(context.Context) {
		assert.Assert(t, st.sender.SendMessageCallCount() > callNum, "SendMessage not called: line %d", lineNum)
	})

	st.needNoSend = false
}

func (st *scriptTester) notifyChannelUpdates(t *testing.T, expected string) {
	lineNum := st.lineNum

	callNum := st.counts[countNotifyChannelUpdates]
	st.counts[countNotifyChannelUpdates]++

	st.addAction(func(context.Context) {
		assert.Assert(t, st.notifier.NotifyChannelUpdatesCallCount() > callNum, "NotifyChannelUpdates not called: line %d", lineNum)
		botName := st.notifier.NotifyChannelUpdatesArgsForCall(callNum)
		assert.Equal(t, botName, expected, "line %d", lineNum)
	})

	st.needNoNotifyChannelUpdates = false
}

func (st *scriptTester) noNotifyChannelUpdates(t *testing.T) {
	lineNum := st.lineNum

	st.addAction(func(context.Context) {
		notifyAfter := st.notifier.NotifyChannelUpdatesCallCount()

		if st.notifyChannelUpdatesBefore != notifyAfter {
			botName := st.notifier.NotifyChannelUpdatesArgsForCall(notifyAfter - 1)
			t.Errorf("notified channel updates for %s: line %d", botName, lineNum)
			t.FailNow()
		}
	})

	st.needNoNotifyChannelUpdates = false
}

func (st *scriptTester) clockForward(t *testing.T, args string) {
	lineNum := st.lineNum

	if _, ok := st.bc.Clock.(*clock.Mock); !ok {
		t.Fatalf("clock must be a mock: line %d", lineNum)
	}

	dur, err := time.ParseDuration(args)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.clock.Forward(dur)
		st.redis.FastForward(dur)
	})
}

func (st *scriptTester) clockSet(t *testing.T, args string) {
	lineNum := st.lineNum

	if _, ok := st.bc.Clock.(*clock.Mock); !ok {
		t.Fatalf("clock must be a mock: line %d", lineNum)
	}

	var tm time.Time

	if args == "now" {
		tm = time.Now()
	} else {
		var err error
		tm, err = time.Parse(time.RFC3339, args)
		assert.NilError(t, err, "line %d", lineNum)
	}

	st.addAction(func(ctx context.Context) {
		st.clock.Set(tm)
		st.redis.SetTime(tm)
	})
}

func (st *scriptTester) join(t *testing.T, args string) {
	lineNum := st.lineNum

	var botName string
	var botID int
	var userName string
	var userID int

	n, err := fmt.Sscanf(args, "%s %d %s %d", &botName, &botID, &userName, &userID)
	assert.NilError(t, err, "line %d", lineNum)
	assert.Equal(t, n, 4)

	m := &irc.Message{
		Tags: map[string]string{
			"id":      uuid.Must(uuid.NewV4()).String(),
			"room-id": strconv.Itoa(botID),
			"user-id": strconv.Itoa(userID),
		},
		Prefix: irc.Prefix{
			Name: userName,
			User: userName,
			Host: userName + ".tmi.twitch.tv",
		},
		Command:  "PRIVMSG",
		Params:   []string{"#" + botName},
		Trailing: "!join",
	}

	st.handleM(t, botName, m)
	st.sendAny(t)
	st.notifyChannelUpdates(t, botName)
}

func (st *scriptTester) sleep(t *testing.T, args string) {
	lineNum := st.lineNum

	dur, err := time.ParseDuration(args)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		time.Sleep(dur)
	})
}

func (st *scriptTester) noLastFM(t *testing.T) {
	st.addAction(func(ctx context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable LastFM")
		st.bc.LastFM = nil
	})
}

func (st *scriptTester) lastFMRecentTracks(t *testing.T, args string) {
	lineNum := st.lineNum

	var v map[string][]lastfm.Track

	err := json.Unmarshal([]byte(args), &v)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.lastFM.RecentTracksCalls(func(user string, n int) ([]lastfm.Track, error) {
			x := v[user]

			if len(x) > n {
				x = x[:n]
			}

			return x, nil
		})
	})
}
