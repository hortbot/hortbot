package bot_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/bot"
	"github.com/hortbot/hortbot/internal/bot/irctobot"
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

	bb.Handle(ctx, privMSG(botName, botName, 1, name, userID, "!join"))

	m := privMSG(botName, name, userID, name, userID, "test")

	b.ResetTimer()
	for range b.N {
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

	bb.Handle(ctx, privMSG(botName, botName, 1, name, userID, "!join"))

	m := privMSG(botName, name, userID, name, userID, "test")

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

	bb.Handle(ctx, privMSG(botName, botName, 1, name, userID, "!join"))
	bb.Handle(ctx, privMSG(botName, name, userID, name, userID, "!command add pan FOUND THE (_PARAMETER_CAPS_), HAVE YE?"))

	m := privMSG(botName, name, userID, name, userID, "!pan working command")

	b.ResetTimer()
	for range b.N {
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

	bb.Handle(ctx, privMSG(botName, botName, 1, name, userID, "!join"))
	bb.Handle(ctx, privMSG(botName, name, userID, name, userID, "!command add pan FOUND THE (_PARAMETER_CAPS_), HAVE YE?"))
	bb.Handle(ctx, privMSG(botName, name, userID, name, userID, "!autoreply add *who_is_zik* Nobody important."))
	bb.Handle(ctx, privMSG(botName, name, userID, name, userID, `!autoreply add REGEX:(^|\b)wowee($|\b) Wowee`))

	ms := make([]bot.Message, 95, 96)

	for i := range ms {
		ms[i] = privMSG(botName, name, userID, "someone", 9999999, "nothing interesting")
	}

	ms = append(ms,
		privMSG(botName, name, userID, name, userID, "!pan working command"),
		privMSG(botName, name, userID, name, userID, "who is zik"),
		privMSG(botName, name, userID, name, userID, "!who knows"),
		privMSG(botName, name, userID, name, userID, "!admin"),
		privMSG(botName, name, userID, name, userID, "!set prefix"),
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

	bb.Handle(ctx, privMSG(botName, botName, 1, name, userID, "!join"))
	bb.Handle(ctx, privMSG(botName, name, userID, name, userID, "!filter on"))
	bb.Handle(ctx, privMSG(botName, name, userID, name, userID, "!filter banphrase on"))

	for range 300 {
		bb.Handle(ctx, privMSG(botName, name, userID, name, userID, "!filter banphrase add "+randomString(10)))
	}

	b.ResetTimer()
	for range b.N {
		bb.Handle(ctx, privMSG(botName, name, userID, "someone", 9999999, "nothing interesting"))
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

func privMSG(origin string, ch string, roomID int64, user string, userID int64, msg string) bot.Message {
	return irctobot.ToMessage(origin, &irc.Message{
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
	})
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
