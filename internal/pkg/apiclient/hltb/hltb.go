// Package hltb implements a HowLongToBeat client.
package hltb

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
	"golang.org/x/net/html"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . API

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
func New(opts ...Option) *HLTB {
	h := &HLTB{
		cli: httpx.Client{
			Name: "hltb",
		},
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Option controls client functionality.
type Option func(*HLTB)

// HTTPClient sets the HTTP client used when making requests to HLTB.
// If given a nil client (or not set), the client will use the default
// HTTP client in net/http.
func HTTPClient(cli *http.Client) Option {
	return func(e *HLTB) {
		e.cli.Client = cli
	}
}

var errNotFound = &apiclient.Error{API: "hltb", StatusCode: 404}

// SearchGame performs a search on HLTB and returns the first result.
func (h *HLTB) SearchGame(ctx context.Context, query string) (*Game, error) {
	extraHeaders := make(http.Header)
	extraHeaders.Set("origin", "https://howlongtobeat.com")
	extraHeaders.Set("referer", "https://howlongtobeat.com/")

	resp, err := h.cli.PostForm(ctx, "https://howlongtobeat.com/search_results?page=1", queryForm(query), extraHeaders)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !apiclient.IsOK(resp.StatusCode) {
		fmt.Println(resp.StatusCode)
		return nil, &apiclient.Error{API: "hltb", StatusCode: resp.StatusCode}
	}

	page, err := html.Parse(resp.Body)
	if err != nil {
		return nil, &apiclient.Error{API: "hltb", Err: err}
	}

	noResults, err := htmlquery.Query(page, "//li[contains(text(), 'No results for')]")
	if err != nil {
		return nil, &apiclient.Error{API: "hltb", Err: err}
	}
	if noResults != nil {
		return nil, errNotFound
	}

	title, err := htmlquery.Query(page, "//div[@class='search_list_details']/*/a")
	if err != nil {
		return nil, &apiclient.Error{API: "hltb", Err: err}
	}
	if title == nil {
		return nil, errNotFound
	}

	var game Game

	path, err := htmlquery.Query(page, "//div[@class='search_list_details']/*/a/@href")
	if err != nil {
		return nil, &apiclient.Error{API: "hltb", Err: err}
	}
	if p := trimmedInner(path); p != "" {
		game.URL = "https://howlongtobeat.com/" + p
	}

	times, err := htmlquery.QueryAll(page, "//div[contains(@class, 'search_list_tidbit')]")
	if err != nil {
		return nil, &apiclient.Error{API: "hltb", Err: err}
	}

Find:
	for i, node := range times {
		switch i {
		case 1:
			game.MainStory = cleanTime(node)
		case 3:
			game.MainPlusExtra = cleanTime(node)
		case 5:
			game.Completionist = cleanTime(node)
			break Find
		}
	}

	titleText := trimmedInner(title)

	if titleText == "" || game == (Game{}) {
		return nil, errNotFound
	}

	game.Title = titleText

	return &game, nil
}

func trimmedInner(node *html.Node) string {
	return strings.TrimSpace(htmlquery.InnerText(node))
}

func cleanTime(node *html.Node) string {
	s := trimmedInner(node)
	s = strings.Trim(s, "-")
	s = strings.ReplaceAll(s, "½", ".5")
	return strings.ToLower(s)
}

var formCommon = url.Values{
	"t":           []string{"games"},
	"sorthead":    []string{"popular"},
	"sortd":       []string{"0"},
	"plat":        []string{""},
	"length_type": []string{"main"},
	"length_min":  []string{""},
	"length_max":  []string{""},
	"v":           []string{""},
	"f":           []string{""},
	"g":           []string{""},
	"detail":      []string{""},
	"randomize":   []string{"0"},
}

func queryForm(query string) url.Values {
	form := make(url.Values, len(formCommon)+1)
	for k, v := range formCommon {
		form[k] = v
	}
	form["queryString"] = []string{query}
	return form
}
