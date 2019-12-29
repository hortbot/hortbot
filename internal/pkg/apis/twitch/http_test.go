package twitch

import (
	"context"
	"errors"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/oauth2x/oauth2xfakes"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func TestHTTPClient(t *testing.T) {
	t.Run("BadRequest", func(t *testing.T) {
		c := &httpClient{}
		_, err := c.newRequest("not a method", "", nil)
		assert.ErrorContains(t, err, "invalid method")
	})

	t.Run("BadToken", func(t *testing.T) {
		testErr := errors.New("testing error")

		c := &httpClient{
			ts: &oauth2xfakes.FakeTokenSource{
				TokenStub: func() (*oauth2.Token, error) {
					return nil, testErr
				},
			},
		}

		_, err := c.newRequest("GET", "http://localhost", nil)
		assert.Equal(t, err, testErr)
	})

	t.Run("BadURL", func(t *testing.T) {
		const errMsg = "invalid control character in URL"

		c := &httpClient{}

		_, err := c.Get(context.Background(), "\n") //nolint:bodyclose
		assert.ErrorContains(t, err, errMsg)

		_, err = c.Put(context.Background(), "\n", 123) //nolint:bodyclose
		assert.ErrorContains(t, err, errMsg)
	})

	t.Run("PutBad", func(t *testing.T) {
		c := &httpClient{
			ts: &oauth2xfakes.FakeTokenSource{
				TokenStub: func() (*oauth2.Token, error) {
					return &oauth2.Token{}, nil
				},
			},
		}

		_, err := c.Put(context.Background(), "http://localhost", unmarshallable{}) //nolint:bodyclose
		assert.ErrorContains(t, err, "unmarshallable")
	})

	t.Run("DoError", func(t *testing.T) {
		c := &httpClient{
			ts: &oauth2xfakes.FakeTokenSource{
				TokenStub: func() (*oauth2.Token, error) {
					return &oauth2.Token{}, nil
				},
			},
		}

		_, err := c.Get(context.Background(), "http://localhost:24353") //nolint:bodyclose
		assert.ErrorContains(t, err, "connection refused")
	})
}

type unmarshallable struct{}

func (unmarshallable) MarshalJSON() ([]byte, error) {
	return nil, errors.New("unmarshallable")
}
