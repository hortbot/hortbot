package twitch_test

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestGetChatters(t *testing.T) {
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
			ctx, cancel := testContext(t)
			defer cancel()

			chatters, err := tw.GetChatters(ctx, test.Channel)
			assert.Equal(t, err, test.Err)
			assert.Assert(t, cmp.DeepEqual(chatters, test.Chatters, cmpopts.EquateEmpty()))
		})
	}
}

func TestGetChattersError(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	_, err := tw.GetChatters(ctx, "geterr")
	assert.ErrorContains(t, err, errTestBadRequest.Error())
}
