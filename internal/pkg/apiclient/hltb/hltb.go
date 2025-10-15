// Package hltb implements a HowLongToBeat client.
package hltb

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/antchfx/htmlquery"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
	"github.com/zikaeroh/ctxlog"
	"golang.org/x/net/html"
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
	cli httpx.Client

	mu       sync.Mutex
	apiToken string
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
	UseCache      bool     `json:"useCache"`
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
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.apiToken == "" {
		apiToken, err := h.getAPIToken(ctx)
		if err != nil {
			return nil, err
		}
		h.apiToken = apiToken

		return h.searchGame(ctx, query, apiToken)
	}

	g, err := h.searchGame(ctx, query, h.apiToken)
	if err == nil {
		return g, err
	}

	if apiErr, ok := apiclient.AsError(err); !ok || !apiErr.IsNotFound() {
		return nil, err
	}

	apiToken, err := h.getAPIToken(ctx)
	if err != nil {
		return nil, err
	}

	h.apiToken = apiToken

	return h.searchGame(ctx, query, apiToken)
}

func (h *HLTB) searchGame(ctx context.Context, query string, apiToken string) (*Game, error) {
	requestBody := &requestBody{
		SearchType:  "games",
		SearchTerms: strings.Fields(query),
		SearchPage:  1,
		Size:        20,
		UseCache:    true,
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

	req := h.cli.NewRequestToJSON("https://howlongtobeat.com/api/search/"+apiToken, &body).
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

var apiTokenRegex = regexp.MustCompile(`"/api/search/".concat\("([a-zA-Z0-9]+)"\)`)

func (h *HLTB) getAPIToken(ctx context.Context) (string, error) {
	ctxlog.Debug(ctx, "refreshing HLTB API token")

	var buf bytes.Buffer

	req := h.cli.NewRequest("https://howlongtobeat.com").
		Header("Origin", "https://howlongtobeat.com").
		Header("Referer", "https://howlongtobeat.com/?q=").
		ToBytesBuffer(&buf)

	if err := req.Fetch(ctx); err != nil {
		return "", apiclient.WrapRequestErr("hltb", err, nil)
	}

	page, err := html.Parse(&buf)
	if err != nil {
		return "", apiclient.WrapRequestErr("hltb", err, nil)
	}

	script, err := htmlquery.Query(page, "//script[contains(@src, '_app')]")
	if err != nil {
		ctxlog.Error(ctx, "Failed to find HLTB script tag")
		return "", apiclient.WrapRequestErr("hltb", err, nil)
	}

	if script == nil {
		return "", apiclient.NewStatusError("hltb", 404)
	}

	src := htmlquery.SelectAttr(script, "src")

	var scriptSource string

	req = h.cli.NewRequest("https://howlongtobeat.com"+src).
		Header("Origin", "https://howlongtobeat.com").
		Header("Referer", "https://howlongtobeat.com/?q=").
		ToString(&scriptSource)

	if err := req.Fetch(ctx); err != nil {
		return "", apiclient.WrapRequestErr("hltb", err, nil)
	}

	matches := apiTokenRegex.FindStringSubmatch(scriptSource)
	if len(matches) != 2 {
		return "", apiclient.NewStatusError("hltb", 404)
	}

	return matches[1], nil
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
