package twitch_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func TestGetStream(t *testing.T) {
	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	want := &twitch.Stream{
		ID:          512301723123,
		GameID:      847362,
		Title:       "This is the title.",
		ViewerCount: 4321,
		UserID:      1234,
	}

	assert.NilError(t, json.Unmarshal([]byte(`"2017-08-14T16:08:32Z"`), &want.StartedAt))

	t.Run("Success by ID", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		stream, err := tw.GetStreamByUserID(ctx, 1234)
		assert.NilError(t, err)

		assert.DeepEqual(t, stream, want)
	})

	t.Run("Success by username", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		stream, err := tw.GetStreamByUsername(ctx, "foobar")
		assert.NilError(t, err)

		assert.DeepEqual(t, stream, want)
	})

	t.Run("Empty", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetStreamByUsername(ctx, "notfound")
		assert.Error(t, err, "twitch: ErrValidator: response error for https://api.twitch.tv/helix/streams?user_login=notfound: unexpected status: 404")
	})

	t.Run("Empty 404", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetStreamByUsername(ctx, "notfound2")
		assert.Error(t, err, "twitch: unexpected status: 404")
	})

	t.Run("Server error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetStreamByUsername(ctx, "servererror")
		assert.Error(t, err, "twitch: ErrValidator: response error for https://api.twitch.tv/helix/streams?user_login=servererror: unexpected status: 500")
	})

	t.Run("Decode error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetStreamByUsername(ctx, "decodeerror")
		assert.Error(t, err, "twitch: ErrHandler: invalid character '}' looking for beginning of value")
	})

	t.Run("Request error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetStreamByUsername(ctx, "requesterror")
		assert.ErrorContains(t, err, errTestBadRequest.Error())
	})
}
