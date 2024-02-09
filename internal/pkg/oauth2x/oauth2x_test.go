package oauth2x_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hortbot/hortbot/internal/pkg/oauth2x"
	"github.com/hortbot/hortbot/internal/pkg/oauth2x/oauth2xfakes"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

var (
	ignoreUnexportedInToken = cmpopts.IgnoreUnexported(oauth2.Token{})

	tokGood = &oauth2.Token{
		AccessToken:  "access-token-good",
		TokenType:    "TYPE",
		RefreshToken: "refresh-token-good",
		Expiry:       time.Now().Add(time.Hour),
	}

	tokeGoodOAuth = tokenWithType(tokGood, "OAuth")

	tokExpired = &oauth2.Token{
		AccessToken:  "access-token-expired",
		TokenType:    "TYPE",
		RefreshToken: "refresh-token-exired",
		Expiry:       time.Now().Add(-time.Hour),
	}

	tokExpired2 = &oauth2.Token{
		AccessToken:  "access-token-expired2",
		TokenType:    "TYPE",
		RefreshToken: "refresh-token-expired2",
		Expiry:       time.Now().Add(-time.Hour / 2),
	}

	errTest = errors.New("testing error")
)

func cloneToken(t *oauth2.Token) *oauth2.Token {
	if t == nil {
		return nil
	}
	t2 := *t
	return &t2
}

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
			expectedTok: tokeGoodOAuth,
			expectedErr: nil,
		},
		{
			name:        "Error",
			override:    "OAuth",
			tok:         nil,
			err:         errTest,
			expectedTok: nil,
			expectedErr: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			fake := &oauth2xfakes.FakeTokenSource{}
			fake.TokenReturns(cloneToken(test.tok), test.err)

			ts := oauth2x.NewTypeOverride(fake, test.override)

			tok, err := ts.Token()
			assert.Equal(t, err, test.expectedErr)
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
			return cloneToken(tokExpired), nil
		}
		return cloneToken(tokGood), nil
	})

	var results []*oauth2.Token

	ts := oauth2x.NewOnNew(fake, func(tok *oauth2.Token, err error) {
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
		return cloneToken(tokGood), nil
	})

	var results []*oauth2.Token

	onFirst := true
	ts := oauth2x.NewOnNew(fake, func(tok *oauth2.Token, err error) {
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

func TestOnNewWithInit(t *testing.T) {
	t.Parallel()

	first := true
	fake := &oauth2xfakes.FakeTokenSource{}
	fake.TokenCalls(func() (*oauth2.Token, error) {
		if first {
			first = false
			return cloneToken(tokExpired2), nil
		}
		return cloneToken(tokGood), nil
	})

	var results []*oauth2.Token

	ts := oauth2x.NewOnNewWithToken(fake, func(tok *oauth2.Token, err error) {
		assert.NilError(t, err)
		results = append(results, tok)
	}, tokExpired)

	tok, err := ts.Token()
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, tokExpired2, ignoreUnexportedInToken)

	tok, err = ts.Token()
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, tokGood, ignoreUnexportedInToken)

	tok, err = ts.Token()
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, tokGood, ignoreUnexportedInToken)

	assert.DeepEqual(t, results, []*oauth2.Token{
		tokExpired2,
		tokGood,
	}, ignoreUnexportedInToken)

	assert.Equal(t, fake.TokenCallCount(), 2)
}

func TestOnNewNil(t *testing.T) {
	t.Parallel()

	first := true
	fake := &oauth2xfakes.FakeTokenSource{}
	fake.TokenCalls(func() (*oauth2.Token, error) {
		if first {
			first = false
			return cloneToken(tokExpired), nil
		}
		return cloneToken(tokGood), nil
	})

	ts := oauth2x.NewOnNew(fake, nil)

	tok, err := ts.Token()
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, tokExpired, ignoreUnexportedInToken)

	tok, err = ts.Token()
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, tokGood, ignoreUnexportedInToken)

	tok, err = ts.Token()
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, tokGood, ignoreUnexportedInToken)

	assert.Equal(t, fake.TokenCallCount(), 2)
}

func TestEquals(t *testing.T) {
	orig := &oauth2.Token{
		AccessToken:  "access-token",
		TokenType:    "bearer",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Round(time.Second),
	}

	cp := func(o *oauth2.Token) *oauth2.Token {
		o2 := *o
		return &o2
	}

	tests := []struct {
		other *oauth2.Token
		equal bool
	}{
		{
			other: nil,
			equal: false,
		},
		{
			other: orig,
			equal: true,
		},
		{
			other: cp(orig),
			equal: true,
		},
		{
			other: &oauth2.Token{
				AccessToken:  "what",
				TokenType:    orig.TokenType,
				RefreshToken: orig.RefreshToken,
				Expiry:       orig.Expiry,
			},
			equal: false,
		},
		{
			other: &oauth2.Token{
				AccessToken:  orig.AccessToken,
				TokenType:    "OAuth",
				RefreshToken: orig.RefreshToken,
				Expiry:       orig.Expiry,
			},
			equal: false,
		},
		{
			other: &oauth2.Token{
				AccessToken:  orig.AccessToken,
				TokenType:    orig.TokenType,
				RefreshToken: "what",
				Expiry:       orig.Expiry,
			},
			equal: false,
		},
		{
			other: &oauth2.Token{
				AccessToken:  orig.AccessToken,
				TokenType:    orig.TokenType,
				RefreshToken: orig.RefreshToken,
				Expiry:       orig.Expiry.Add(time.Hour),
			},
			equal: false,
		},
		{
			other: &oauth2.Token{
				AccessToken:  orig.AccessToken,
				TokenType:    orig.TokenType,
				RefreshToken: orig.RefreshToken,
				Expiry:       orig.Expiry.Add(-time.Hour),
			},
			equal: false,
		},
		{
			other: &oauth2.Token{
				AccessToken:  orig.AccessToken,
				TokenType:    orig.TokenType,
				RefreshToken: orig.RefreshToken,
			},
			equal: false,
		},
	}

	for i, test := range tests {
		assert.Assert(t, oauth2x.Equals(orig, test.other) == test.equal, "test %d", i)
		assert.Assert(t, oauth2x.Equals(test.other, orig) == test.equal, "test %d", i)
	}
}
