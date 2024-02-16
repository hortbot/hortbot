// Package oauth2x contains helpers for the oauth2 package.
package oauth2x

import (
	"sync"
	"time"

	"golang.org/x/oauth2"
)

//go:generate go run github.com/matryer/moq -fmt goimports -out oauth2xmocks/mocks.go -pkg oauth2xmocks . TokenSource

type TokenSource = oauth2.TokenSource

type onNewSource struct {
	ts    oauth2.TokenSource
	onNew func(*oauth2.Token, error)

	mu  sync.Mutex
	tok *oauth2.Token
}

// NewOnNew creates a token source which calls onNew when a new token is created, passing
// the new token and an error (if any occurred while fetching it). If onNew is nil, then
// the token source returned is equivalent to one returned by oauth2.ReuseTokenSource.
//
// onNew is called synchronously under lock and will block other uses of this
// token source. If a non-blocking operation is desired, create a new goroutine
// in onNew.
func NewOnNew(ts oauth2.TokenSource, onNew func(*oauth2.Token, error)) oauth2.TokenSource {
	return NewOnNewWithToken(ts, onNew, nil)
}

// NewOnNewWithToken is the same as OnNew, but uses tok as the first token
// instead of calling out to the wrapped token source first. If this token
// is no longer valid, then the wrapped token source will be called, as
// would be done for any expiry.
func NewOnNewWithToken(ts oauth2.TokenSource, onNew func(*oauth2.Token, error), tok *oauth2.Token) oauth2.TokenSource {
	if onNew == nil {
		return oauth2.ReuseTokenSource(tok, ts)
	}

	return &onNewSource{
		ts:    ts,
		onNew: onNew,
		tok:   tok,
	}
}

func (ts *onNewSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.tok.Valid() {
		return ts.tok, nil
	}

	tok, err := ts.ts.Token()
	ts.onNew(tok, err)
	ts.tok = tok

	return tok, err
}

// Equals checks that two tokens are equal by the following rules:
//
// - AccessToken, TokenType, and RefreshToken must be identical.
// - Expiry must match within one second.
func Equals(x, y *oauth2.Token) bool {
	switch {
	case x == y:
		return true
	case x == nil || y == nil:
	case x.AccessToken != y.AccessToken:
	case x.TokenType != y.TokenType:
	case x.RefreshToken != y.RefreshToken:
	case x.Expiry.IsZero() != y.Expiry.IsZero():
	case absDur(x.Expiry.Sub(y.Expiry)) > time.Second:
	default:
		return true
	}
	return false
}

func absDur(d time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return -d
}
