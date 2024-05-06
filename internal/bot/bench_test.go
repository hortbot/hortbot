package bot_test

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
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
	"github.com/jakebailey/irc"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func BenchmarkHandleNop(b *testing.B) {
	const botName = "hortbot"

	db := pool.FreshDB(b)
	defer db.Close()

	ctx := context.Background()

	rServer, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(b, err)
	defer rCleanup()

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:       db,
		Redis:    redis.New(rClient),
		Notifier: nopNotifier{},
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

	bb.Handle(ctx, botName, privMSG(botName, 1, name, userID, "!join"))

	m := privMSG(name, userID, name, userID, "test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb.Handle(ctx, botName, m)
		rServer.FastForward(time.Minute)
	}
	b.StopTimer()
}

func BenchmarkHandleNopParallel(b *testing.B) {
	const botName = "hortbot"

	db := pool.FreshDB(b)
	defer db.Close()

	ctx := context.Background()

	_, rClient, rCleanup, err := miniredistest.New()
	assert.NilError(b, err)
	defer rCleanup()

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:       db,
		Redis:    redis.New(rClient),
		Notifier: nopNotifier{},
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

	bb.Handle(ctx, botName, privMSG(botName, 1, name, userID, "!join"))

	m := privMSG(name, userID, name, userID, "test")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bb.Handle(ctx, botName, m)
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

	ctx := context.Background()

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:       db,
		Redis:    redis.New(rClient),
		Notifier: nopNotifier{},
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

	bb.Handle(ctx, botName, privMSG(botName, 1, name, userID, "!join"))
	bb.Handle(ctx, botName, privMSG(name, userID, name, userID, "!command add pan FOUND THE (_PARAMETER_CAPS_), HAVE YE?"))

	m := privMSG(name, userID, name, userID, "!pan working command")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb.Handle(ctx, botName, m)
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

	ctx := context.Background()

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:       db,
		Redis:    redis.New(rClient),
		Notifier: nopNotifier{},
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

	bb.Handle(ctx, botName, privMSG(botName, 1, name, userID, "!join"))
	bb.Handle(ctx, botName, privMSG(name, userID, name, userID, "!command add pan FOUND THE (_PARAMETER_CAPS_), HAVE YE?"))
	bb.Handle(ctx, botName, privMSG(name, userID, name, userID, "!autoreply add *who_is_zik* Nobody important."))
	bb.Handle(ctx, botName, privMSG(name, userID, name, userID, `!autoreply add REGEX:(^|\b)wowee($|\b) Wowee`))

	ms := make([]*irc.Message, 95, 96)

	for i := range ms {
		ms[i] = privMSG(name, userID, "someone", 9999999, "nothing interesting")
	}

	ms = append(ms,
		privMSG(name, userID, name, userID, "!pan working command"),
		privMSG(name, userID, name, userID, "who is zik"),
		privMSG(name, userID, name, userID, "!who knows"),
		privMSG(name, userID, name, userID, "!admin"),
		privMSG(name, userID, name, userID, "!set prefix"),
	)

	l := len(ms)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := ms[i%l]
		bb.Handle(ctx, botName, m)
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

	ctx := context.Background()

	userID, name := getNextUserID()

	config := &bot.Config{
		DB:       db,
		Redis:    redis.New(rClient),
		Notifier: nopNotifier{},
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

	bb.Handle(ctx, botName, privMSG(botName, 1, name, userID, "!join"))
	bb.Handle(ctx, botName, privMSG(name, userID, name, userID, "!filter on"))
	bb.Handle(ctx, botName, privMSG(name, userID, name, userID, "!filter banphrase on"))

	for range 300 {
		bb.Handle(ctx, botName, privMSG(name, userID, name, userID, "!filter banphrase add "+randomString(10)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb.Handle(ctx, botName, privMSG(name, userID, "someone", 9999999, "nothing interesting"))
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

func privMSG(ch string, roomID int64, user string, userID int64, msg string) *irc.Message {
	return &irc.Message{
		Tags: map[string]string{
			"id":      uuid.Must(uuid.NewV4()).String(),
			"room-id": strconv.FormatInt(roomID, 10),
			"user-id": strconv.FormatInt(userID, 10),
		},
		Prefix: irc.Prefix{
			Name: user,
			User: user,
			Host: user + "@tmi.twitch.tv",
		},
		Command:  "PRIVMSG",
		Params:   []string{"#" + ch},
		Trailing: msg,
	}
}

func randomString(n int) string {
	const characters = "qwertyuiopasdfghjklzxcvbnm"

	var builder strings.Builder
	builder.Grow(n)

	for range n {
		x := rand.Intn(len(characters)) //nolint:gosec
		builder.WriteByte(characters[x])
	}

	return builder.String()
}
