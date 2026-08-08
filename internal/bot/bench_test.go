package bot_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/hltb/hltbmocks"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/simple/simplemocks"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/twitchmocks"
	"github.com/hortbot/hortbot/internal/pkg/testutil/miniredistest"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func BenchmarkHandleNop(b *testing.B) {
	const botName = "hortbot"

	db := pool.FreshDB(b)
	defer db.Close()

	ctx := b.Context()

	rServer, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(b, err)
	defer rCleanup()

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:                     db,
		Redis:                  redis.New(rClient),
		EventsubUpdateNotifier: nopNotifier{},
		Twitch: &twitchmocks.APIMock{
			SendChatMessageFunc: func(ctx context.Context, broadcasterID, modID int64, modToken *oauth2.Token, message string) (*oauth2.Token, error) {
				return nil, nil //nolint:nilnil
			},
		},
		Simple:     &simplemocks.APIMock{},
		HLTB:       &hltbmocks.APIMock{},
		NoDedupe:   true,
		PublicJoin: true,
	}

	bb := bot.New(config)
	assert.NilError(b, bb.Init(ctx))

	bb.Handle(ctx, chatMessage(botName, botName, 1, name, userID, "!join"))

	m := chatMessage(botName, name, userID, name, userID, "test")

	for b.Loop() {
		bb.Handle(ctx, m)
		rServer.FastForward(time.Minute)
	}
	b.StopTimer()
}

func BenchmarkHandleNopParallel(b *testing.B) {
	const botName = "hortbot"

	db := pool.FreshDB(b)
	defer db.Close()

	ctx := b.Context()

	_, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(b, err)
	defer rCleanup()

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:                     db,
		Redis:                  redis.New(rClient),
		EventsubUpdateNotifier: nopNotifier{},
		Twitch: &twitchmocks.APIMock{
			SendChatMessageFunc: func(ctx context.Context, broadcasterID, modID int64, modToken *oauth2.Token, message string) (*oauth2.Token, error) {
				return nil, nil //nolint:nilnil
			},
		},
		Simple:     &simplemocks.APIMock{},
		HLTB:       &hltbmocks.APIMock{},
		NoDedupe:   true,
		PublicJoin: true,
	}

	bb := bot.New(config)
	assert.NilError(b, bb.Init(ctx))

	bb.Handle(ctx, chatMessage(botName, botName, 1, name, userID, "!join"))

	m := chatMessage(botName, name, userID, name, userID, "test")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bb.Handle(ctx, m)
		}
	})
	b.StopTimer()
}

func BenchmarkHandleCustomCommand(b *testing.B) {
	const botName = "hortbot"

	rServer, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(b, err)
	defer rCleanup()

	db := pool.FreshDB(b)
	defer db.Close()

	ctx := b.Context()

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:                     db,
		Redis:                  redis.New(rClient),
		EventsubUpdateNotifier: nopNotifier{},
		Twitch: &twitchmocks.APIMock{
			SendChatMessageFunc: func(ctx context.Context, broadcasterID, modID int64, modToken *oauth2.Token, message string) (*oauth2.Token, error) {
				return nil, nil //nolint:nilnil
			},
		},
		Simple:     &simplemocks.APIMock{},
		HLTB:       &hltbmocks.APIMock{},
		NoDedupe:   true,
		PublicJoin: true,
	}

	bb := bot.New(config)
	assert.NilError(b, bb.Init(ctx))

	bb.Handle(ctx, chatMessage(botName, botName, 1, name, userID, "!join"))
	bb.Handle(ctx, chatMessage(botName, name, userID, name, userID, "!command add pan FOUND THE (_PARAMETER_CAPS_), HAVE YE?"))

	m := chatMessage(botName, name, userID, name, userID, "!pan working command")

	for b.Loop() {
		bb.Handle(ctx, m)
		rServer.FastForward(time.Minute)
	}
	b.StopTimer()
}

func BenchmarkHandleMixed(b *testing.B) {
	const botName = "hortbot"

	rServer, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(b, err)
	defer rCleanup()

	db := pool.FreshDB(b)
	defer db.Close()

	ctx := b.Context()

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:                     db,
		Redis:                  redis.New(rClient),
		EventsubUpdateNotifier: nopNotifier{},
		Twitch: &twitchmocks.APIMock{
			SendChatMessageFunc: func(ctx context.Context, broadcasterID, modID int64, modToken *oauth2.Token, message string) (*oauth2.Token, error) {
				return nil, nil //nolint:nilnil
			},
		},
		Simple:     &simplemocks.APIMock{},
		HLTB:       &hltbmocks.APIMock{},
		NoDedupe:   true,
		PublicJoin: true,
	}

	bb := bot.New(config)
	assert.NilError(b, bb.Init(ctx))

	bb.Handle(ctx, chatMessage(botName, botName, 1, name, userID, "!join"))
	bb.Handle(ctx, chatMessage(botName, name, userID, name, userID, "!command add pan FOUND THE (_PARAMETER_CAPS_), HAVE YE?"))
	bb.Handle(ctx, chatMessage(botName, name, userID, name, userID, "!autoreply add *who_is_zik* Nobody important."))
	bb.Handle(ctx, chatMessage(botName, name, userID, name, userID, `!autoreply add REGEX:(^|\b)wowee($|\b) Wowee`))

	ms := make([]bot.Message, 95, 96)

	for i := range ms {
		ms[i] = chatMessage(botName, name, userID, "someone", 9999999, "nothing interesting")
	}

	ms = append(ms,
		chatMessage(botName, name, userID, name, userID, "!pan working command"),
		chatMessage(botName, name, userID, name, userID, "who is zik"),
		chatMessage(botName, name, userID, name, userID, "!who knows"),
		chatMessage(botName, name, userID, name, userID, "!admin"),
		chatMessage(botName, name, userID, name, userID, "!set prefix"),
	)

	l := len(ms)

	b.ResetTimer()
	for i := range b.N {
		m := ms[i%l]
		bb.Handle(ctx, m)
		rServer.FastForward(time.Minute)
	}
	b.StopTimer()
}

func BenchmarkHandleManyBannedPhrases(b *testing.B) {
	const botName = "hortbot"

	rServer, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(b, err)
	defer rCleanup()

	db := pool.FreshDB(b)
	defer db.Close()

	ctx := b.Context()

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:                     db,
		Redis:                  redis.New(rClient),
		EventsubUpdateNotifier: nopNotifier{},
		Twitch: &twitchmocks.APIMock{
			SendChatMessageFunc: func(ctx context.Context, broadcasterID, modID int64, modToken *oauth2.Token, message string) (*oauth2.Token, error) {
				return nil, nil //nolint:nilnil
			},
		},
		Simple:     &simplemocks.APIMock{},
		HLTB:       &hltbmocks.APIMock{},
		NoDedupe:   true,
		PublicJoin: true,
	}

	bb := bot.New(config)
	assert.NilError(b, bb.Init(ctx))

	bb.Handle(ctx, chatMessage(botName, botName, 1, name, userID, "!join"))
	bb.Handle(ctx, chatMessage(botName, name, userID, name, userID, "!filter on"))
	bb.Handle(ctx, chatMessage(botName, name, userID, name, userID, "!filter banphrase on"))

	for range 300 {
		bb.Handle(ctx, chatMessage(botName, name, userID, name, userID, "!filter banphrase add "+randomString(10)))
	}

	for b.Loop() {
		bb.Handle(ctx, chatMessage(botName, name, userID, "someone", 9999999, "nothing interesting"))
		rServer.FastForward(time.Minute)
	}
	b.StopTimer()
}

var nextUserID atomic.Int64

func init() {
	nextUserID.Store(1) // The bot is canonically user 1; start at user 2.
}

func getNextUserID() (int64, string) {
	id := nextUserID.Add(1)
	return id, fmt.Sprintf("user%d", id)
}

type nopNotifier struct{}

func (nopNotifier) NotifyChannelUpdates(ctx context.Context, botName string) error { return nil }
func (nopNotifier) NotifyEventsubUpdates(ctx context.Context) error                { return nil }

func chatMessage(botLogin, broadcasterLogin string, broadcasterID int64, chatterLogin string, chatterID int64, text string) bot.Message {
	return benchmarkMessage{
		botLogin: botLogin,
		id:       uuid.Must(uuid.NewV4()).String(),
		broadcaster: bot.ChatIdentity{
			ID:    broadcasterID,
			Login: broadcasterLogin,
		},
		chatter: bot.ChatIdentity{
			ID:    chatterID,
			Login: chatterLogin,
		},
		text: text,
	}
}

type benchmarkMessage struct {
	botLogin    string
	id          string
	broadcaster bot.ChatIdentity
	chatter     bot.ChatIdentity
	text        string
}

func (m benchmarkMessage) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(m.text)
	if err != nil {
		return nil, fmt.Errorf("marshal benchmark message: %w", err)
	}
	return data, nil
}

func (m benchmarkMessage) Bot() string                   { return m.botLogin }
func (m benchmarkMessage) MessageID() string             { return m.id }
func (benchmarkMessage) MessageTimestamp() time.Time     { return time.Time{} }
func (m benchmarkMessage) Broadcaster() bot.ChatIdentity { return m.broadcaster }
func (m benchmarkMessage) Chatter() bot.ChatIdentity     { return m.chatter }
func (m benchmarkMessage) Text() string                  { return m.text }
func (benchmarkMessage) IsAction() bool                  { return false }
func (benchmarkMessage) CountEmotes() int                { return 0 }
func (m benchmarkMessage) ChatterAccessLevel() bot.AccessLevel {
	if m.broadcaster.ID == m.chatter.ID {
		return bot.AccessLevelBroadcaster
	}
	return bot.AccessLevelUnknown
}

func randomString(n int) string {
	const characters = "qwertyuiopasdfghjklzxcvbnm"

	var builder strings.Builder
	builder.Grow(n)

	for range n {
		x := rand.N(len(characters)) //nolint:gosec
		builder.WriteByte(characters[x])
	}

	return builder.String()
}
