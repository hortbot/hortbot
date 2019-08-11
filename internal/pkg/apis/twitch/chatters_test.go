package twitch_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

func TestGetChatters(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	chatters := &twitch.Chatters{
		Count: 1234,
	}
	chatters.Chatters.Broadcaster = []string{"foobar"}
	chatters.Chatters.Viewers = []string{"foo", "bar"}

	tests := []struct {
		Channel  string
		Chatters *twitch.Chatters
		Err      error
	}{
		{
			Channel:  "foobar",
			Chatters: chatters,
		},
		{
			Channel: "notfound",
			Err:     twitch.ErrNotFound,
		},
		{
			Channel: "servererror",
			Err:     twitch.ErrServerError,
		},
		{
			Channel: "badbody",
			Err:     twitch.ErrServerError,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.Channel, func(t *testing.T) {
			chatters, err := tw.GetChatters(ctx, test.Channel)
			assert.Equal(t, err, test.Err)
			assert.Assert(t, cmp.DeepEqual(chatters, test.Chatters, cmpopts.EquateEmpty()))
		})
	}
}
