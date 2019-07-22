package oauth2x

import (
	"sync"

	"golang.org/x/oauth2"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 golang.org/x/oauth2.TokenSource

// TypeOverrideSource is an TokenSource which overrides the TokenType of the returned token.
type TypeOverrideSource struct {
	ts  oauth2.TokenSource
	typ string
}

var _ oauth2.TokenSource = (*TypeOverrideSource)(nil)

// NewTypeOverrideSource creates a new TypeOverrideSource based on the given
// TokenSource. If tokenType is non-empty, then tokens returned by Token will
// have their TokenType replaced.
func NewTypeOverrideSource(ts oauth2.TokenSource, tokenType string) *TypeOverrideSource {
	return &TypeOverrideSource{
		ts:  ts,
		typ: tokenType,
	}
}

func (ts *TypeOverrideSource) Token() (*oauth2.Token, error) {
	tok, err := ts.ts.Token()
	if err != nil {
		return nil, err
	}

	if ts.typ == "" || tok == nil {
		return tok, nil
	}

	tok2 := *tok
	tok2.TokenType = ts.typ
	return &tok2, nil
}

// OnNewTokenSource is a TokenSource which calls a function when a new token is created,
// allowing for the token to be persisted or otherwise used outside of the OAuth2 flow.
type OnNewTokenSource struct {
	ts    oauth2.TokenSource
	onNew func(*oauth2.Token, error)

	mu  sync.Mutex
	tok *oauth2.Token
}

// NewOnNewTokenSource creates a new OnNewTokenSource based on the given
// TokenSource. When a new token is created (either when no token has yet been
// fetched, or an old one expires), then onNew will be called with the new token
// as well as the error returned when getting it, if any. If onNew is nil, then
// it will not be called.
//
// onNew is called synchronously under lock and will block other uses of this
// TokenSource. If a non-blocking operation is desired, create a new goroutine
// in onNew.
func NewOnNewTokenSource(ts oauth2.TokenSource, onNew func(*oauth2.Token, error)) *OnNewTokenSource {
	return &OnNewTokenSource{
		ts:    ts,
		onNew: onNew,
	}
}

var _ oauth2.TokenSource = (*OnNewTokenSource)(nil)

func (ts *OnNewTokenSource) Token() (*oauth2.Token, error) {
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
