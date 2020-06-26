package twitch_test

import (
	"context"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func TestDeadToken(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)

		const id = 1
		tok := invalidRefreshTokFor(ctx, t, tw, ft, id, "invalid")
		_, err := tw.ModifyChannel(ctx, id, tok, "some new title", 0)
		assert.Equal(t, err, twitch.ErrDeadToken)
	})

	t.Run("Unknown message", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)

		const id = 1
		tok := invalidRefreshTokFor(ctx, t, tw, ft, id, "unknown")
		_, err := tw.ModifyChannel(ctx, id, tok, "some new title", 0)
		assert.Equal(t, err, twitch.ErrDeadToken)
	})

	t.Run("Decode error", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)

		const id = 1
		tok := invalidRefreshTokFor(ctx, t, tw, ft, id, "decodeerror")
		_, err := tw.ModifyChannel(ctx, id, tok, "some new title", 0)
		assert.Equal(t, err, twitch.ErrServerError)
	})
}

func invalidRefreshTokFor(ctx context.Context, t *testing.T, tw *twitch.Twitch, ft *fakeTwitch, id int64, refresh string) *oauth2.Token {
	code := ft.codeForUserInvalidRefresh(id, refresh)

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	// Awful hack to make sure the oauth library tries to refresh the token.
	assert.Assert(t, tok.Valid())
	tok.Expiry = tok.Expiry.Add(-time.Hour)
	assert.Assert(t, !tok.Valid())

	return tok
}
