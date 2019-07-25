// Package oauth2x contains helpers for the oauth2 package.
package oauth2x

import (
	"sync"

	"golang.org/x/oauth2"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 golang.org/x/oauth2.TokenSource

type overrideSource struct {
	ts  oauth2.TokenSource
	typ string
}

// NewTypeOverride creates a token source which overrides the tokens returned
// by ts with a different auth type. If typ is empty, then the original
// token source is returned.
func NewTypeOverride(ts oauth2.TokenSource, typ string) oauth2.TokenSource {
	if typ == "" {
		return ts
	}

	return &overrideSource{
		ts:  ts,
		typ: typ,
	}
}

func (ts *overrideSource) Token() (*oauth2.Token, error) {
	tok, err := ts.ts.Token()
	if tok == nil || err != nil {
		return tok, err
	}

	tok2 := *tok
	tok2.TokenType = ts.typ
	return &tok2, nil
}

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
