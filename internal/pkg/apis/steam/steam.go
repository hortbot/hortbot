// Package steam provides a Steam API client.
package steam

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"golang.org/x/net/context/ctxhttp"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// Steam API errors.
//
//     - 200 -> nil
//     - 404 -> ErrNotFound
//     - 401 or 403 -> ErrNotAuthorized
//     - 5xx -> ErrServerError
//     - Otherwise -> ErrUnknown
var (
	ErrNotFound      = errors.New("steam: not found")
	ErrNotAuthorized = errors.New("steam: not authorized")
	ErrServerError   = errors.New("steam: server error")
	ErrUnknown       = errors.New("steam: unknown error")
)

//counterfeiter:generate . API

// API represents the supported API functions. It's defined for fake generation.
type API interface {
	GetPlayerSummary(ctx context.Context, id string) (*Summary, error)
	GetOwnedGames(ctx context.Context, id string) ([]*Game, error)
}

// Steam is a Steam API client.
type Steam struct {
	apiKey string
	cli    *http.Client
}

var _ API = (*Steam)(nil)

// Option controls client functionality.
type Option func(*Steam)

// New creates a new Steam API client.
func New(apiKey string, opts ...Option) *Steam {
	if apiKey == "" {
		panic("empty apiKey")
	}

	s := &Steam{
		apiKey: apiKey,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// HTTPClient sets the Steam client's underlying http.Client.
// If nil (or if this option wasn't used), http.DefaultClient will be used.
func HTTPClient(cli *http.Client) Option {
	return func(s *Steam) {
		s.cli = cli
	}
}

// Summary is a Steam player summary.
type Summary struct {
	Name       string `json:"personaname"`
	ProfileURL string `json:"profileurl"`
	Game       string `json:"gameextrainfo"`
	GameID     string `json:"gameid"`
	GameServer string `json:"gameserverip"`
}

// GetPlayerSummary gets a Steam user's summary.
//
// GET http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/
func (s *Steam) GetPlayerSummary(ctx context.Context, id string) (*Summary, error) {
	url := "http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=" + s.apiKey + "&format=json&steamids=" + url.QueryEscape(id)

	resp, err := ctxhttp.Get(ctx, s.cli, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	body := struct {
		Response struct {
			Players []*Summary `json:"players"`
		} `json:"response"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, &body); err != nil {
		return nil, ErrServerError
	}

	p := body.Response.Players

	if len(p) == 0 {
		return nil, ErrNotFound
	}

	return p[0], nil
}

// Game is a Steam owned game.
type Game struct {
	ID   int64  `json:"appid"`
	Name string `json:"name"`
}

// GetOwnedGames gets a Steam user's owned games.
//
// GET http://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/
func (s *Steam) GetOwnedGames(ctx context.Context, id string) ([]*Game, error) {
	url := "http://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?key=" + s.apiKey + "&format=json&steamid=" + url.QueryEscape(id) + "&include_appinfo=1"

	resp, err := ctxhttp.Get(ctx, s.cli, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	body := struct {
		Response struct {
			Games []*Game `json:"games"`
		} `json:"response"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, &body); err != nil {
		return nil, ErrServerError
	}

	return body.Response.Games, nil
}

func statusToError(code int) error {
	if code >= 200 && code < 300 {
		return nil
	}

	switch code {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrNotAuthorized
	}

	if code >= 500 {
		return ErrServerError
	}

	return ErrUnknown
}
