package bot_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/botfakes"
	"github.com/hortbot/hortbot/internal/ctxlog"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/dedupe"
	"github.com/hortbot/hortbot/internal/testutil"
	"github.com/jakebailey/irc"
	"github.com/volatiletech/sqlboiler/boil"
	"gotest.tools/assert"
)

func TestScripts(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "script", "*.txt"))
	assert.NilError(t, err)
	assert.Assert(t, len(files) != 0)

	for _, file := range files {
		file := file
		name := strings.TrimSuffix(filepath.Base(file), ".txt")
		t.Run(name, func(t *testing.T) {
			ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(t))
			testScriptFile(ctx, t, file)
		})
	}
}

func testScriptFile(ctx context.Context, t *testing.T, filename string) {
	assert.NilError(t, migrations.Reset(db, nil))

	f, err := os.Open(filename)
	assert.NilError(t, err)
	defer f.Close()

	sender := &botfakes.FakeMessageSender{}

	bc := bot.Config{
		DB:     db,
		Dedupe: dedupe.NeverSeen,
		Sender: sender,
	}

	var b *bot.Bot
	counts := make(map[string]int)
	var actions []func()

	ensureBot := func() {
		if b != nil {
			return
		}
		b = bot.New(&bc)
	}

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" || line[0] == '#' {
			continue
		}

		directive := strings.SplitN(line, " ", 2)

		switch directive[0] {
		case "skip":
			if len(directive) > 1 {
				reason := strings.TrimSpace(directive[1])
				t.Skip(reason)
			} else {
				t.SkipNow()
			}

		case "bot_config":
			actions = append(actions, func() {
				assert.Assert(t, b == nil, "bot has already been created, cannot configure")
				assert.NilError(t, json.Unmarshal([]byte(directive[1]), &bc))
			})

		case "channel":
			var channel models.Channel
			assert.NilError(t, json.Unmarshal([]byte(directive[1]), &channel))

			actions = append(actions, func() {
				assert.NilError(t, channel.Insert(ctx, db, boil.Infer()))
			})

		case "simple_command":
			var sc models.SimpleCommand
			assert.NilError(t, json.Unmarshal([]byte(directive[1]), &sc))

			actions = append(actions, func() {
				assert.NilError(t, sc.Insert(ctx, db, boil.Infer()))
			})

		case "handle":
			m, err := irc.ParseMessage(directive[1])
			assert.NilError(t, err)

			actions = append(actions, func() {
				ensureBot()
				b.Handle(ctx, m)
			})

		case "send":
			callNum := counts["send"]
			counts["send"]++

			sent := strings.SplitN(directive[1], " ", 2)

			actions = append(actions, func() {
				target, message := sender.SendMessageArgsForCall(callNum)
				assert.Equal(t, target, sent[0])
				assert.Equal(t, message, sent[1])
			})

		default:
			t.Fatalf("unknown directive %s", directive[0])
		}
	}

	assert.NilError(t, scanner.Err())

	assert.Assert(t, len(actions) != 0)

	for _, fn := range actions {
		fn()
	}
}
