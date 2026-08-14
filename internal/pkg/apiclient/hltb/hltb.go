// Package hltb implements a HowLongToBeat client.
package hltb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
)

const (
	baseURL        = "https://howlongtobeat.com"
	initURL        = baseURL + "/api/bleed/init"
	searchURL      = baseURL + "/api/bleed"
	searchPageSize = 1
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

type authData struct {
	Token string `json:"token"`
	HPKey string `json:"hpKey"`
	HPVal string `json:"hpVal"`
}

func (a authData) secrets() []string {
	secrets := make([]string, 0, 3)
	for _, secret := range []string{a.Token, a.HPKey, a.HPVal} {
		if secret != "" {
			secrets = append(secrets, secret)
		}
	}
	return secrets
}

type searchRequest struct {
	SearchType    string               `json:"searchType"`
	SearchTerms   []string             `json:"searchTerms"`
	SearchPage    int                  `json:"searchPage"`
	Size          int                  `json:"size"`
	SearchOptions searchRequestOptions `json:"searchOptions"`
	UseCache      bool                 `json:"useCache"`
	hpKey         string
	hpVal         string
}

type searchRequestOptions struct {
	Games      searchRequestGames `json:"games"`
	Users      searchRequestUsers `json:"users"`
	Lists      searchRequestLists `json:"lists"`
	Filter     string             `json:"filter"`
	Sort       int                `json:"sort"`
	Randomizer int                `json:"randomizer"`
}

type searchRequestGames struct {
	UserID        int                   `json:"userId"`
	Platform      string                `json:"platform"`
	SortCategory  string                `json:"sortCategory"`
	RangeCategory string                `json:"rangeCategory"`
	RangeTime     searchRequestRange    `json:"rangeTime"`
	Gameplay      searchRequestGameplay `json:"gameplay"`
	RangeYear     searchRequestRange    `json:"rangeYear"`
	Modifier      string                `json:"modifier"`
}

type searchRequestRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type searchRequestGameplay struct {
	Perspective string `json:"perspective"`
	Flow        string `json:"flow"`
	Genre       string `json:"genre"`
	Difficulty  string `json:"difficulty"`
}

type searchRequestUsers struct {
	SortCategory string `json:"sortCategory"`
}

type searchRequestLists struct {
	SortCategory string `json:"sortCategory"`
}

func (r searchRequest) MarshalJSON() ([]byte, error) {
	type plainSearchRequest searchRequest

	body, err := json.Marshal(plainSearchRequest(r))
	if err != nil {
		return nil, fmt.Errorf("marshaling search request: %w", err)
	}
	if r.hpKey == "" {
		return body, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("unmarshaling search request: %w", err)
	}

	hpVal, err := json.Marshal(r.hpVal)
	if err != nil {
		return nil, fmt.Errorf("marshaling honeypot value: %w", err)
	}
	fields[r.hpKey] = hpVal

	body, err = json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("marshaling search request fields: %w", err)
	}
	return body, nil
}

type searchResponse struct {
	Data []struct {
		GameID   int    `json:"game_id"`
		GameName string `json:"game_name"`
		CompMain int    `json:"comp_main"`
		CompPlus int    `json:"comp_plus"`
		Comp100  int    `json:"comp_100"`
	} `json:"data"`
}

// HLTB is a HowLongToBeat client.
type HLTB struct {
	cli    httpx.Client
	authMu sync.Mutex
	auth   authData
}

var _ API = (*HLTB)(nil)

// New creates a new HLTB client.
func New(cli *http.Client) *HLTB {
	return &HLTB{
		cli: httpx.NewClient(cli, "hltb", httpx.WithBrowserUserAgent()),
	}
}

// SearchGame performs a search on HLTB and returns the first result.
func (h *HLTB) SearchGame(ctx context.Context, query string) (*Game, error) {
	if len(strings.Fields(query)) == 0 {
		return nil, apiclient.NewNonStatusError("hltb", errors.New("search term cannot be empty"))
	}

	auth, err := h.getAuth(ctx, "")
	if err != nil {
		return nil, err
	}

	search, err := h.search(ctx, query, auth)
	if apiErr, ok := apiclient.AsError(err); ok && apiErr.StatusCode == http.StatusForbidden {
		auth, err = h.getAuth(ctx, auth.Token)
		if err != nil {
			return nil, err
		}
		search, err = h.search(ctx, query, auth)
	}
	if err != nil {
		return nil, err
	}

	if len(search.Data) == 0 {
		return nil, apiclient.NewStatusError("hltb", http.StatusNotFound)
	}

	first := search.Data[0]
	return &Game{
		Title:         first.GameName,
		URL:           fmt.Sprintf("%s/game/%d", baseURL, first.GameID),
		MainStory:     timeToString(first.CompMain),
		MainPlusExtra: timeToString(first.CompPlus),
		Completionist: timeToString(first.Comp100),
	}, nil
}

func (h *HLTB) getAuth(ctx context.Context, staleToken string) (authData, error) {
	h.authMu.Lock()
	defer h.authMu.Unlock()

	if h.auth.Token != "" && h.auth.Token != staleToken {
		return h.auth, nil
	}

	var auth authData
	req := h.cli.NewRequestToJSON(initURL, &auth).
		Param("t", strconv.FormatInt(time.Now().UnixMilli(), 10)).
		Header("Origin", baseURL+"/").
		Header("Referer", baseURL+"/")

	if err := req.Fetch(ctx); err != nil {
		return authData{}, apiclient.WrapRequestErr("hltb", err, nil)
	}
	if auth.Token == "" {
		return authData{}, apiclient.NewNonStatusError("hltb", errors.New("empty search token"))
	}

	h.auth = auth
	return auth, nil
}

func (h *HLTB) search(ctx context.Context, query string, auth authData) (*searchResponse, error) {
	body := searchRequest{
		SearchType:  "games",
		SearchTerms: strings.Fields(query),
		SearchPage:  1,
		Size:        searchPageSize,
		SearchOptions: searchRequestOptions{
			Games: searchRequestGames{
				SortCategory:  "popular",
				RangeCategory: "main",
				Modifier:      "hide_dlc",
			},
			Users: searchRequestUsers{
				SortCategory: "postcount",
			},
			Lists: searchRequestLists{
				SortCategory: "follows",
			},
		},
		UseCache: true,
		hpKey:    auth.HPKey,
		hpVal:    auth.HPVal,
	}

	var search searchResponse
	req := h.cli.NewRequestToJSON(searchURL, &search).
		Header("Origin", baseURL+"/").
		Header("Referer", baseURL+"/").
		Header("Content-Type", "application/json").
		Header("x-auth-token", auth.Token).
		Header("x-hp-key", auth.HPKey).
		Header("x-hp-val", auth.HPVal).
		BodyJSON(body).
		Post()

	if err := req.Fetch(ctx); err != nil {
		return nil, apiclient.WrapRequestErr("hltb", err, auth.secrets())
	}

	return &search, nil
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
