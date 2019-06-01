package bot_test

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/bmatcuk/doublestar"
	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/botfakes"
	"github.com/hortbot/hortbot/internal/ctxlog"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/dedupe"
	dedupemem "github.com/hortbot/hortbot/internal/dedupe/memory"
	"github.com/hortbot/hortbot/internal/testutil"
	"github.com/jakebailey/irc"
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
	sender   *botfakes.FakeSender
	notifier *botfakes.FakeNotifier

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

func (st *scriptTester) ensureBot() {
	if st.b == nil {
		st.b = bot.New(&st.bc)
	}
}

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

	defer func() {
		for _, cleanup := range st.cleanups {
			defer cleanup()
		}
	}()

	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(t))

	db, undb := freshDB(t)
	defer undb()

	st.db = db

	st.bc = bot.Config{
		DB:       db,
		Dedupe:   dedupe.NeverSeen,
		Sender:   st.sender,
		Notifier: st.notifier,
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

		default:
			t.Fatalf("line %d: unknown directive %s", st.lineNum, directive)
		}
	}

	assert.NilError(t, scanner.Err())

	if len(st.actions) == 0 {
		t.Error("no actions were provided")
		t.FailNow()
	}

	for _, action := range st.actions {
		action(ctx)
	}

	// TODO: make constants for these
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
		Prefix           string
		Bullet           string
		Dedupe           string
		Admins           []string
		WhitelistEnabled bool
		Whitelist        []string
	}

	assert.NilError(t, json.Unmarshal([]byte(args), &bcj), "line %d", lineNum)

	st.bc.Prefix = bcj.Prefix
	st.bc.Bullet = bcj.Bullet
	st.bc.Admins = bcj.Admins
	st.bc.WhitelistEnabled = bcj.WhitelistEnabled
	st.bc.Whitelist = bcj.Whitelist

	switch bcj.Dedupe {
	case "", "never":
		st.bc.Dedupe = dedupe.NeverSeen

	case "memory":
		d := dedupemem.New(time.Minute, 5*time.Minute)
		st.addCleanup(d.Stop)
		st.bc.Dedupe = d

	default:
		t.Fatalf("line %d: unknown dedupe type %s", st.lineNum, bcj.Dedupe)
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

func (st *scriptTester) handle(t *testing.T, directiveArgs string, me bool) {
	lineNum := st.lineNum

	if st.needNoSend {
		// noSend(lineNum)
	}

	if st.needNoNotifyChannelUpdates {
		// noNotify(lineNum)
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
		st.ensureBot()

		st.sentBefore = st.sender.SendMessageCallCount()
		st.notifyChannelUpdatesBefore = st.notifier.NotifyChannelUpdatesCallCount()

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
		assert.Assert(t, pattern.MatchString(message), "line %d", lineNum)
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
