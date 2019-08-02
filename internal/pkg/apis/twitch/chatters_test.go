package twitch_test

import (
	"context"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"gotest.tools/assert"
)

func TestGetChatters(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	tests := []struct {
		Channel  string
		Chatters int64
		Err      error
	}{
		{
			Channel:  "foobar",
			Chatters: 1234,
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
			assert.Equal(t, chatters, test.Chatters)
			assert.Equal(t, err, test.Err)
		})
	}
}
