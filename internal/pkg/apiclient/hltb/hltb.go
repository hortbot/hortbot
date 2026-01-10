// Package hltb implements a HowLongToBeat client.
package hltb

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/forbiddencoding/howlongtobeat"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
)

//go:generate go tool github.com/matryer/moq -fmt goimports -out hltbmocks/mocks.go -pkg hltbmocks . API

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
	hltb *howlongtobeat.Client
}

var _ API = &HLTB{}

// New creates a new HLTB client.
func New(cli *http.Client) *HLTB {
	hltb, err := howlongtobeat.New(howlongtobeat.WithHTTPClient(cli))
	if err != nil {
		panic(fmt.Sprintf("failed to create HLTB client: %v", err))
	}
	return &HLTB{
		hltb: hltb,
	}
}

// SearchGame performs a search on HLTB and returns the first result.
func (h *HLTB) SearchGame(ctx context.Context, query string) (*Game, error) {
	search, err := h.hltb.Search(ctx, query, howlongtobeat.SearchModifierHideDLC, &howlongtobeat.SearchOptions{
		Pagination: &howlongtobeat.SearchGamePagination{
			Page:     1,
			PageSize: 1,
		},
	})
	if err != nil {
		return nil, apiclient.WrapRequestErr("hltb", err, nil)
	}

	games := search.Data
	if len(games) == 0 {
		return nil, apiclient.NewStatusError("hltb", 404)
	}

	first := games[0]
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
