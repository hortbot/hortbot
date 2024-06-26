// Package hltb implements a HowLongToBeat client.
package hltb

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
)

//go:generate go run github.com/matryer/moq -fmt goimports -out hltbmocks/mocks.go -pkg hltbmocks . API

// API represents the supported API functions. It's defined for fake generation.
type API interface {
	SearchGame(ctx context.Context, query string) (*Game, error)
}

// Game is a found game on HLTB.
type Game struct {
	Title         string
	URL           string
	MainStory     string
	MainPlusExtra string
	Completionist string
}

// HLTB is a HowLongToBeat client.
type HLTB struct {
	cli httpx.Client
}

var _ API = &HLTB{}

// New creates a new HLTB client.
func New(cli *http.Client) *HLTB {
	return &HLTB{
		cli: httpx.NewClient(cli, "hltb", httpx.WithBrowserUserAgent()),
	}
}

type requestBody struct {
	SearchType    string   `json:"searchType"`
	SearchTerms   []string `json:"searchTerms"`
	SearchPage    int      `json:"searchPage"`
	Size          int      `json:"size"`
	SearchOptions struct {
		Games struct {
			UserID        int    `json:"userId"`
			Platform      string `json:"platform"`
			SortCategory  string `json:"sortCategory"`
			RangeCategory string `json:"rangeCategory"`
			RangeTime     struct {
				Min int `json:"min"`
				Max int `json:"max"`
			} `json:"rangeTime"`
			Gameplay struct {
				Perspective string `json:"perspective"`
				Flow        string `json:"flow"`
				Genre       string `json:"genre"`
			} `json:"gameplay"`
			Modifier string `json:"modifier"`
		} `json:"games"`
		Users struct {
			SortCategory string `json:"sortCategory"`
		} `json:"users"`
		Filter     string `json:"filter"`
		Sort       int    `json:"sort"`
		Randomizer int    `json:"randomizer"`
	} `json:"searchOptions"`
}

// SearchGame performs a search on HLTB and returns the first result.
func (h *HLTB) SearchGame(ctx context.Context, query string) (*Game, error) {
	requestBody := &requestBody{
		SearchType:  "games",
		SearchTerms: strings.Fields(query),
		SearchPage:  1,
		Size:        20,
	}
	requestBody.SearchOptions.Games.SortCategory = "popular"
	requestBody.SearchOptions.Games.RangeCategory = "main"
	requestBody.SearchOptions.Users.SortCategory = "postcount"

	var body struct {
		Data []struct {
			GameName string `json:"game_name"`
			GameID   int    `json:"game_id"`
			CompMain int    `json:"comp_main"`
			CompPlus int    `json:"comp_plus"`
			Comp100  int    `json:"comp_100"`
		} `json:"data"`
	}

	req := h.cli.NewRequestToJSON("https://howlongtobeat.com/api/search", &body).
		Header("Origin", "https://howlongtobeat.com").
		Header("Referer", "https://howlongtobeat.com/?q=").
		BodyJSON(requestBody).
		Post()

	if err := req.Fetch(ctx); err != nil {
		return nil, apiclient.WrapRequestErr("hltb", err, nil)
	}

	if len(body.Data) == 0 {
		return nil, apiclient.NewStatusError("hltb", 404)
	}

	first := body.Data[0]

	return &Game{
		Title:         first.GameName,
		URL:           fmt.Sprintf("https://howlongtobeat.com/game/%d", first.GameID),
		MainStory:     timeToString(first.CompMain),
		MainPlusExtra: timeToString(first.CompPlus),
		Completionist: timeToString(first.Comp100),
	}, nil
}

func timeToString(t int) string {
	if t == 0 {
		return ""
	}
	hours := strconv.FormatFloat(round(float64(t)/3600, 0.5), 'f', 1, 64)
	hours = strings.TrimRight(hours, ".0")
	return hours + " hours"
}

func round(x, to float64) float64 {
	return math.Round(x/to) * to
}
