package btest

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
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

	if strings.HasSuffix(directiveArgs, " nil") {
		st.handleM(t, nil)
		return
	}

	header, text, _ := strings.Cut(directiveArgs, " :")
	fields := strings.Fields(header)
	assert.Assert(t, len(fields) >= 3, "line %d", lineNum)

	origin := fields[0]
	m := &testChatMessage{
		botLogin:    origin,
		id:          rand.Text(),
		broadcaster: parseIdentity(t, fields[1], lineNum),
		chatter:     parseIdentity(t, fields[2], lineNum),
		text:        text,
		action:      directive == "handle_me",
	}

	for _, option := range fields[3:] {
		key, value, ok := strings.Cut(option, "=")
		assert.Assert(t, ok, "line %d", lineNum)
		if value == "-" {
			value = ""
		}

		switch key {
		case "message-id":
			m.id = value
		case "sent-at":
			sentAt, err := time.Parse(time.RFC3339Nano, value)
			assert.NilError(t, err, "line %d", lineNum)
			m.sentAt = sentAt
		case "broadcaster-display":
			m.broadcaster.DisplayName = value
		case "chatter-display":
			m.chatter.DisplayName = value
		case "emote-count":
			emotes, err := strconv.Atoi(value)
			assert.NilError(t, err, "line %d", lineNum)
			m.emoteCount = emotes
		case "access":
			m.accessLevel = parseAccessLevel(t, value, lineNum)
		default:
			t.Fatalf("line %d: unknown message option %s", lineNum, key)
		}
	}

	st.handleM(t, m)
}

type testChatMessage struct {
	botLogin    string
	id          string
	sentAt      time.Time
	broadcaster bot.ChatIdentity
	chatter     bot.ChatIdentity
	text        string
	action      bool
	emoteCount  int
	accessLevel bot.AccessLevel
}

func (m *testChatMessage) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(struct {
		BotLogin    string           `json:"bot_login"`
		ID          string           `json:"id"`
		SentAt      time.Time        `json:"sent_at"`
		Broadcaster bot.ChatIdentity `json:"broadcaster"`
		Chatter     bot.ChatIdentity `json:"chatter"`
		Text        string           `json:"text"`
		IsAction    bool             `json:"is_action"`
	}{
		BotLogin:    m.botLogin,
		ID:          m.id,
		SentAt:      m.sentAt,
		Broadcaster: m.broadcaster,
		Chatter:     m.chatter,
		Text:        m.text,
		IsAction:    m.action,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal test chat message: %w", err)
	}
	return data, nil
}

func (m *testChatMessage) Bot() string                   { return m.botLogin }
func (m *testChatMessage) MessageID() string             { return m.id }
func (m *testChatMessage) MessageTimestamp() time.Time   { return m.sentAt }
func (m *testChatMessage) Broadcaster() bot.ChatIdentity { return m.broadcaster }
func (m *testChatMessage) Chatter() bot.ChatIdentity     { return m.chatter }
func (m *testChatMessage) Text() string                  { return m.text }
func (m *testChatMessage) IsAction() bool                { return m.action }
func (m *testChatMessage) CountEmotes() int              { return m.emoteCount }
func (m *testChatMessage) ChatterAccessLevel() bot.AccessLevel {
	if m.accessLevel != bot.AccessLevelUnknown {
		return m.accessLevel
	}
	if m.broadcaster.ID != 0 && m.broadcaster.ID == m.chatter.ID {
		return bot.AccessLevelBroadcaster
	}
	return bot.AccessLevelUnknown
}

func parseIdentity(t testing.TB, value string, lineNum int) bot.ChatIdentity {
	if value == "-" {
		return bot.ChatIdentity{}
	}

	name, id, ok := strings.Cut(value, "/")
	assert.Assert(t, ok, "line %d", lineNum)
	if name == "#" {
		name = ""
	}
	if id == "-" {
		id = "0"
	}
	numericID, err := strconv.ParseInt(id, 10, 64)
	assert.NilError(t, err, "line %d", lineNum)
	return bot.ChatIdentity{ID: numericID, Login: name}
}

func parseAccessLevel(t testing.TB, value string, lineNum int) bot.AccessLevel {
	switch value {
	case "broadcaster":
		return bot.AccessLevelBroadcaster
	case "moderator":
		return bot.AccessLevelModerator
	case "vip":
		return bot.AccessLevelVIP
	case "subscriber":
		return bot.AccessLevelSubscriber
	case "admin":
		return bot.AccessLevelAdmin
	case "super-admin":
		return bot.AccessLevelSuperAdmin
	default:
		t.Fatalf("line %d: unknown access level %s", lineNum, value)
		return bot.AccessLevelUnknown
	}
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

	tt := dbsql.TwitchToken{
		TwitchID:     int64(botID),
		BotName:      dbsql.TextFrom(botName),
		AccessToken:  "some-access-token",
		TokenType:    "bearer",
		RefreshToken: "some-refresh-token",
		Expiry:       dbsql.TimestamptzFrom(expiry),
		Scopes:       twitch.BotScopes,
	}
	assert.NilError(t, st.queries.SaveTwitchToken(t.Context(), &tt), "line %d", lineNum)

	tt = dbsql.TwitchToken{
		TwitchID:     int64(userID),
		AccessToken:  "some-access-token",
		TokenType:    "bearer",
		RefreshToken: "some-refresh-token",
		Expiry:       dbsql.TimestamptzFrom(expiry),
		Scopes:       twitch.UserScopes,
	}
	assert.NilError(t, st.queries.SaveTwitchToken(t.Context(), &tt), "line %d", lineNum)

	m := &testChatMessage{
		botLogin: botName,
		id:       rand.Text(),
		broadcaster: bot.ChatIdentity{
			ID:    int64(botID),
			Login: botName,
		},
		chatter: bot.ChatIdentity{
			ID:    int64(userID),
			Login: userName,
		},
		text: "!join",
	}

	st.handleM(t, m)
	st.sendAny(t, "", "", lineNum)
	st.notifyEventsubUpdatesCalls(t, "", botName, lineNum)
}
