package twitch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"github.com/hortbot/hortbot/internal/pkg/oauth2x/oauth2xfakes"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/zikaeroh/ctxlog"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func TestHTTPClient(t *testing.T) {
	t.Run("BadRequest", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		c := &httpClient{}
		_, err := c.newRequest(ctx, "not a method", "", nil)
		assert.ErrorContains(t, err, "invalid method")
	})

	t.Run("BadToken", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		testErr := errors.New("testing error")

		c := &httpClient{
			ts: &oauth2xfakes.FakeTokenSource{
				TokenStub: func() (*oauth2.Token, error) {
					return nil, testErr
				},
			},
		}

		_, err := c.newRequest(ctx, "GET", "http://localhost", nil)
		assert.Equal(t, err, testErr)
	})

	t.Run("BadURL", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		const errMsg = "invalid control character in URL"

		c := &httpClient{}

		_, err := c.Get(ctx, "\n") //nolint:bodyclose
		assert.ErrorContains(t, err, errMsg)

		_, err = c.Put(ctx, "\n", 123) //nolint:bodyclose
		assert.ErrorContains(t, err, errMsg)

		_, err = c.Post(ctx, "\n", 123) //nolint:bodyclose
		assert.ErrorContains(t, err, errMsg)
	})

	t.Run("Unmarshallable body", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		c := &httpClient{
			ts: &oauth2xfakes.FakeTokenSource{
				TokenStub: func() (*oauth2.Token, error) {
					return &oauth2.Token{}, nil
				},
			},
		}

		_, err := c.Put(ctx, "http://localhost", jsonx.Unmarshallable()) //nolint:bodyclose
		assert.ErrorContains(t, err, jsonx.ErrUnmarshallable.Error())

		_, err = c.Post(ctx, "http://localhost", jsonx.Unmarshallable()) //nolint:bodyclose
		assert.ErrorContains(t, err, jsonx.ErrUnmarshallable.Error())
	})

	t.Run("DoError", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		c := &httpClient{
			ts: &oauth2xfakes.FakeTokenSource{
				TokenStub: func() (*oauth2.Token, error) {
					return &oauth2.Token{}, nil
				},
			},
		}

		_, err := c.Get(ctx, "http://localhost:24353") //nolint:bodyclose
		assert.ErrorContains(t, err, "illegal attempt to use default HTTP transport")
	})
}

func testContext(t testing.TB) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(t))
	return context.WithTimeout(ctx, 10*time.Minute)
}
