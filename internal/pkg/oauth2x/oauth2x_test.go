package oauth2x_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hortbot/hortbot/internal/pkg/oauth2x"
	"github.com/hortbot/hortbot/internal/pkg/oauth2x/oauth2xfakes"
	"golang.org/x/oauth2"
	"gotest.tools/assert"
)

var (
	ignoreUnexportedInToken = cmpopts.IgnoreUnexported(oauth2.Token{})

	tokGood = &oauth2.Token{
		AccessToken:  "access-token",
		TokenType:    "TYPE",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}

	tokExpired = &oauth2.Token{
		AccessToken:  "access-token",
		TokenType:    "TYPE",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(-time.Hour),
	}

	errTest = errors.New("testing error")
)

func TestOverrideEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		override    string
		tok         *oauth2.Token
		err         error
		expectedTok *oauth2.Token
		expectedErr error
	}{
		{
			name:        "Empty",
			override:    "",
			tok:         tokGood,
			err:         nil,
			expectedTok: tokGood,
			expectedErr: nil,
		},
		{
			name:        "Modified",
			override:    "OAuth",
			tok:         tokGood,
			err:         nil,
			expectedTok: tokenWithType(tokGood, "OAuth"),
			expectedErr: nil,
		},
		{
			name:        "Error",
			override:    "OAuth",
			tok:         tokGood,
			err:         errTest,
			expectedTok: nil,
			expectedErr: errTest,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			fake := &oauth2xfakes.FakeTokenSource{}
			fake.TokenReturns(test.tok, test.err)

			ts := oauth2x.NewTypeOverrideSource(fake, test.override)

			tok, err := ts.Token()
			assert.Assert(t, err == test.expectedErr)
			assert.DeepEqual(t, tok, test.expectedTok, ignoreUnexportedInToken)
		})
	}
}

func tokenWithType(t *oauth2.Token, typ string) *oauth2.Token {
	t2 := *t
	t2.TokenType = typ
	return &t2
}

func TestOnNew(t *testing.T) {
	t.Parallel()

	first := true
	fake := &oauth2xfakes.FakeTokenSource{}
	fake.TokenCalls(func() (*oauth2.Token, error) {
		if first {
			first = false
			return tokExpired, nil
		}
		return tokGood, nil
	})

	var results []*oauth2.Token

	ts := oauth2x.NewOnNewTokenSource(fake, func(tok *oauth2.Token, err error) {
		assert.NilError(t, err)
		results = append(results, tok)
	})

	tok, err := ts.Token()
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, tokExpired, ignoreUnexportedInToken)

	tok, err = ts.Token()
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, tokGood, ignoreUnexportedInToken)

	tok, err = ts.Token()
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, tokGood, ignoreUnexportedInToken)

	assert.DeepEqual(t, results, []*oauth2.Token{
		tokExpired,
		tokGood,
	}, ignoreUnexportedInToken)

	assert.Equal(t, fake.TokenCallCount(), 2)
}

func TestOnNewError(t *testing.T) {
	t.Parallel()

	first := true
	fake := &oauth2xfakes.FakeTokenSource{}
	fake.TokenCalls(func() (*oauth2.Token, error) {
		if first {
			first = false
			return nil, errTest
		}
		return tokGood, nil
	})

	var results []*oauth2.Token

	onFirst := true
	ts := oauth2x.NewOnNewTokenSource(fake, func(tok *oauth2.Token, err error) {
		if onFirst {
			onFirst = false
			assert.Assert(t, tok == nil)
			assert.Equal(t, err, errTest)
		} else {
			assert.NilError(t, err)
		}

		results = append(results, tok)
	})

	tok, err := ts.Token()
	assert.Equal(t, err, errTest)
	assert.Assert(t, tok == nil)

	tok, err = ts.Token()
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, tokGood, ignoreUnexportedInToken)

	tok, err = ts.Token()
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, tokGood, ignoreUnexportedInToken)

	assert.DeepEqual(t, results, []*oauth2.Token{
		nil,
		tokGood,
	}, ignoreUnexportedInToken)

	assert.Equal(t, fake.TokenCallCount(), 2)
}
