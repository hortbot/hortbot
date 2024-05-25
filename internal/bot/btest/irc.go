package btest

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/irctobot"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/jakebailey/irc"
	"github.com/volatiletech/null/v8"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func (st *scriptTester) handle(t testing.TB, directive, directiveArgs string, lineNum int) {
	if st.needNoSend {
		st.noSend(t, "", "", lineNum)
	}

	if st.needNoNotifyEventsubUpdatesCalls {
		st.noNotifyEventsubUpdatesCalls(t, "", "", lineNum)
	}

	st.needNoSend = true
	st.needNoNotifyEventsubUpdatesCalls = true

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

		if directive == "handle_me" {
			m.Trailing, _ = irc.EncodeCTCP("ACTION", m.Trailing)
		}
	}

	st.handleM(t, irctobot.ToMessage(origin, m))
}

func (st *scriptTester) handleM(t testing.TB, m bot.Message) {
	st.addAction(func(ctx context.Context) {
		st.ensureBot(ctx, t)
		st.doCheckpoint()
		st.b.Handle(ctx, m)
	})
}

func (st *scriptTester) send(t testing.TB, _, args string, lineNum int) {
	callNum := st.counts[countSend]
	st.counts[countSend]++

	sent := strings.SplitN(args, " ", 3)
	assert.Assert(t, len(sent) == 3, "line %d", lineNum)

	st.addAction(func(context.Context) {
		calls := st.sender.SendMessageCalls()
		assert.Assert(t, len(calls) > callNum, "SendMessage not called: line %d", lineNum)
		call := calls[callNum]

		ok := true
		ok = assert.Check(t, cmp.Equal(call.Origin, sent[0]), "line %d", lineNum) && ok
		ok = assert.Check(t, cmp.Equal(call.Target, sent[1]), "line %d", lineNum) && ok
		ok = assert.Check(t, cmp.Equal(call.Message, sent[2]), "line %d", lineNum) && ok
		if !ok {
			t.FailNow()
		}
	})

	st.needNoSend = false
}

func (st *scriptTester) sendMatch(t testing.TB, _, args string, lineNum int) {
	callNum := st.counts[countSend]
	st.counts[countSend]++

	sent := strings.SplitN(args, " ", 3)
	assert.Assert(t, len(sent) == 3, "line %d", lineNum)

	pattern, err := regexp.Compile(sent[2])
	assert.NilError(t, err)

	st.addAction(func(context.Context) {
		calls := st.sender.SendMessageCalls()
		assert.Assert(t, len(calls) > callNum, "SendMessage not called: line %d", lineNum)
		call := calls[callNum]

		ok := true
		ok = assert.Check(t, cmp.Equal(call.Origin, sent[0]), "line %d", lineNum) && ok
		ok = assert.Check(t, cmp.Equal(call.Target, sent[1]), "line %d", lineNum) && ok
		ok = assert.Check(t, pattern.MatchString(call.Message), "pattern=`%s`, message=`%s`: line %d", pattern, call.Message, lineNum) && ok
		if !ok {
			t.FailNow()
		}
	})

	st.needNoSend = false
}

func (st *scriptTester) noSend(t testing.TB, _, _ string, lineNum int) {
	st.addAction(func(context.Context) {
		calls := st.sender.SendMessageCalls()
		sentAfter := len(calls)

		if st.sentBefore != sentAfter {
			call := calls[sentAfter-1]
			t.Errorf("sent message: origin=%s, target=%s, message=%s: line %d", call.Origin, call.Target, call.Message, lineNum)
			t.FailNow()
		}
	})

	st.needNoSend = false
}

func (st *scriptTester) sendAny(t testing.TB, _, _ string, lineNum int) {
	callNum := st.counts["send"]
	st.counts["send"]++

	st.addAction(func(context.Context) {
		assert.Assert(t, len(st.sender.SendMessageCalls()) > callNum, "SendMessage not called: line %d", lineNum)
	})

	st.needNoSend = false
}

func (st *scriptTester) notifyEventsubUpdatesCalls(t testing.TB, _, expected string, lineNum int) {
	callNum := st.counts[countNotifyEventsubUpdates]
	st.counts[countNotifyEventsubUpdates]++

	st.addAction(func(context.Context) {
		calls := st.eventsubUpdateNotifier.NotifyEventsubUpdatesCalls()
		assert.Assert(t, len(calls) > callNum, "NotifyEventsubUpdatesCalls not called: line %d", lineNum)
	})

	st.needNoNotifyEventsubUpdatesCalls = false
}

func (st *scriptTester) noNotifyEventsubUpdatesCalls(t testing.TB, _, _ string, lineNum int) {
	st.addAction(func(context.Context) {
		calls := st.eventsubUpdateNotifier.NotifyEventsubUpdatesCalls()
		notifyAfter := len(calls)

		if st.notifyEventsubUpdatesCallsBefore != notifyAfter {
			t.Errorf("notified eventsub updates: line %d", lineNum)
			t.FailNow()
		}
	})

	st.needNoNotifyEventsubUpdatesCalls = false
}

func (st *scriptTester) join(t testing.TB, _, args string, lineNum int) {
	var botName string
	var botID int
	var userName string
	var userID int

	n, err := fmt.Sscanf(args, "%s %d %s %d", &botName, &botID, &userName, &userID)
	assert.NilError(t, err, "line %d", lineNum)
	assert.Equal(t, n, 4)

	st.idToName[int64(botID)] = botName
	st.idToName[int64(userID)] = userName

	expiry, err := time.Parse(time.RFC3339, "2050-10-01T03:11:00Z")
	assert.NilError(t, err, "line %d", lineNum)

	tt := models.TwitchToken{
		TwitchID:     int64(botID),
		BotName:      null.StringFrom(botName),
		AccessToken:  "some-access-token",
		TokenType:    "bearer",
		RefreshToken: "some-refresh-token",
		Expiry:       expiry,
		Scopes:       twitch.BotScopes,
	}
	assert.NilError(t, modelsx.UpsertToken(context.TODO(), st.db, &tt), "line %d", lineNum)

	tt = models.TwitchToken{
		TwitchID:     int64(userID),
		AccessToken:  "some-access-token",
		TokenType:    "bearer",
		RefreshToken: "some-refresh-token",
		Expiry:       expiry,
		Scopes:       twitch.UserScopes,
	}
	assert.NilError(t, modelsx.UpsertToken(context.TODO(), st.db, &tt), "line %d", lineNum)

	m := irctobot.ToMessage(botName, &irc.Message{
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
	})

	st.handleM(t, m)
	st.sendAny(t, "", "", lineNum)
	st.notifyEventsubUpdatesCalls(t, "", botName, lineNum)
}
