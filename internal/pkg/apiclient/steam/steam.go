// Package steam provides a Steam API client.
package steam

import (
	"context"
	"net/http"
	"net/url"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . API

// API represents the supported API functions. It's defined for fake generation.
type API interface {
	GetPlayerSummary(ctx context.Context, id string) (*Summary, error)
	GetOwnedGames(ctx context.Context, id string) ([]*Game, error)
}

// Steam is a Steam API client.
type Steam struct {
	apiKey string
	cli    httpx.Client
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
		s.cli.Client = cli
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
// GET https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/
func (s *Steam) GetPlayerSummary(ctx context.Context, id string) (*Summary, error) {
	url := "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=" + url.QueryEscape(s.apiKey) + "&format=json&steamids=" + url.QueryEscape(id)

	resp, err := s.cli.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !apiclient.IsOK(resp.StatusCode) {
		return nil, &apiclient.Error{API: "steam", StatusCode: resp.StatusCode}
	}

	body := struct {
		Response struct {
			Players []*Summary `json:"players"`
		} `json:"response"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, &body); err != nil {
		return nil, &apiclient.Error{API: "steam", Err: err}
	}

	p := body.Response.Players

	if len(p) == 0 {
		return nil, &apiclient.Error{API: "steam", StatusCode: http.StatusNotFound}
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
// GET https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/
func (s *Steam) GetOwnedGames(ctx context.Context, id string) ([]*Game, error) {
	url := "https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?key=" + s.apiKey + "&format=json&steamid=" + url.QueryEscape(id) + "&include_appinfo=1"

	resp, err := s.cli.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !apiclient.IsOK(resp.StatusCode) {
		return nil, &apiclient.Error{API: "steam", StatusCode: resp.StatusCode}
	}

	body := struct {
		Response struct {
			Games []*Game `json:"games"`
		} `json:"response"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, &body); err != nil {
		return nil, &apiclient.Error{API: "steam", Err: err}
	}

	return body.Response.Games, nil
}
