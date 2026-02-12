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
	t.Parallel()

	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)

		const id = 1
		tok := invalidRefreshTokFor(ctx, t, tw, ft, id, "invalid")
		_, err := tw.ModifyChannel(ctx, id, tok, new("some new title"), nil)
		assert.ErrorIs(t, err, twitch.ErrDeadToken)
	})

	t.Run("Unknown message", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)

		const id = 1
		tok := invalidRefreshTokFor(ctx, t, tw, ft, id, "unknown")
		_, err := tw.ModifyChannel(ctx, id, tok, new("some new title"), nil)
		assert.ErrorIs(t, err, twitch.ErrDeadToken)
	})

	t.Run("Decode error", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)

		const id = 1
		tok := invalidRefreshTokFor(ctx, t, tw, ft, id, "decodeerror")
		_, err := tw.ModifyChannel(ctx, id, tok, new("some new title"), nil)
		assert.Error(t, err, "twitch: invalid character '}' looking for beginning of value")
	})
}

func invalidRefreshTokFor(ctx context.Context, t *testing.T, tw *twitch.Twitch, ft *fakeTwitch, id int64, refresh string) *oauth2.Token { //nolint:thelper
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
