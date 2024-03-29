package btest

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/irctobot"
	"github.com/jakebailey/irc"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func (st *scriptTester) handle(t testing.TB, directive, directiveArgs string, lineNum int) {
	if st.needNoSend {
		st.noSend(t, "", "", lineNum)
	}

	if st.needNoNotifyChannelUpdates {
		st.noNotifyChannelUpdates(t, "", "", lineNum)
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

		if directive == "handle_me" {
			m.Trailing, _ = irc.EncodeCTCP("ACTION", m.Trailing)
		}
	}

	st.handleM(t, origin, irctobot.IRCToMessage(m))
}

func (st *scriptTester) handleM(t testing.TB, origin string, m bot.Message) {
	st.addAction(func(ctx context.Context) {
		st.ensureBot(ctx, t)
		st.doCheckpoint()
		st.b.Handle(ctx, origin, m)
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

func (st *scriptTester) notifyChannelUpdates(t testing.TB, _, expected string, lineNum int) {
	callNum := st.counts[countNotifyChannelUpdates]
	st.counts[countNotifyChannelUpdates]++

	st.addAction(func(context.Context) {
		calls := st.notifier.NotifyChannelUpdatesCalls()
		assert.Assert(t, len(calls) > callNum, "NotifyChannelUpdates not called: line %d", lineNum)
		call := calls[callNum]
		assert.Equal(t, call.BotName, expected, "line %d", lineNum)
	})

	st.needNoNotifyChannelUpdates = false
}

func (st *scriptTester) noNotifyChannelUpdates(t testing.TB, _, _ string, lineNum int) {
	st.addAction(func(context.Context) {
		calls := st.notifier.NotifyChannelUpdatesCalls()
		notifyAfter := len(calls)

		if st.notifyChannelUpdatesBefore != notifyAfter {
			call := calls[notifyAfter-1]
			t.Errorf("notified channel updates for %s: line %d", call.BotName, lineNum)
			t.FailNow()
		}
	})

	st.needNoNotifyChannelUpdates = false
}

func (st *scriptTester) join(t testing.TB, _, args string, lineNum int) {
	var botName string
	var botID int
	var userName string
	var userID int

	n, err := fmt.Sscanf(args, "%s %d %s %d", &botName, &botID, &userName, &userID)
	assert.NilError(t, err, "line %d", lineNum)
	assert.Equal(t, n, 4)

	m := irctobot.IRCToMessage(&irc.Message{
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

	st.handleM(t, botName, m)
	st.sendAny(t, "", "", lineNum)
	st.notifyChannelUpdates(t, "", botName, lineNum)
}
