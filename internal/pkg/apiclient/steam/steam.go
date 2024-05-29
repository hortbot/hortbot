// Package steam provides a Steam API client.
package steam

import (
	"context"
	"net/http"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
)

//go:generate go run github.com/matryer/moq -fmt goimports -out steammocks/mocks.go -pkg steammocks . API

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

// New creates a new Steam API client.
func New(apiKey string, cli *http.Client) *Steam {
	if apiKey == "" {
		panic("empty apiKey")
	}

	return &Steam{
		apiKey: apiKey,
		cli:    httpx.NewClient(cli, "steam", false),
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
	var body struct {
		Response struct {
			Players []*Summary `json:"players"`
		} `json:"response"`
	}

	req := s.cli.NewRequestToJSON("https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/", &body).
		Param("key", s.apiKey).
		Param("format", "json").
		Param("steamids", id)

	if err := req.Fetch(ctx); err != nil {
		return nil, apiclient.WrapRequestErr("steam", err, []string{s.apiKey})
	}

	p := body.Response.Players

	if len(p) == 0 {
		return nil, apiclient.NewStatusError("steam", 404)
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
	var body struct {
		Response struct {
			Games []*Game `json:"games"`
		} `json:"response"`
	}

	req := s.cli.NewRequestToJSON("https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/", &body).
		Param("key", s.apiKey).
		Param("format", "json").
		Param("steamid", id).
		Param("include_appinfo", "1")

	if err := req.Fetch(ctx); err != nil {
		return nil, apiclient.WrapRequestErr("steam", err, []string{s.apiKey})
	}

	return body.Response.Games, nil
}
