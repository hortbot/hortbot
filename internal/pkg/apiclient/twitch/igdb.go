package twitch

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
)

type GameLinkType uint8

//go:generate go run golang.org/x/tools/cmd/stringer -type=GameLinkType -trimprefix=GameLink

// Most of these are undocumented, but discoverable by doing something like:
// fields external_games.category, external_games.url;
// where external_games.category = 6;
// limit 10;

const (
	_ GameLinkType = iota
	GameLinkSteam
	GameLinkGamesDB
	GameLinkGiantBomb
	GameLinkGoOptimuz
	GameLinkGOG
	GameLinkPushSquare
	GameLinkIsThereAnyDeal
	GameLinkGamersGate
	_ // No 9.
	GameLinkYouTube
	GameLinkMicrosoft
	GameLinkNintendoLife
	GameLinkApple
	GameLinkTwitch
	GameLinkAndroid
	GameLinkPlaystation
	_ // No 17.
	GameLinkXbox
	GameLinkGamersPress
	GameLinkAmazon
	GameLinkNintendo
	GameLinkPlayAmazon
	_ // play.amazon.com
	GameLinkPlayAsia
	GameLinkTapTap
	GameLinkEpic
	GameLinkTouchArcade
	GameLinkOculus
)

var gameLinkOrdering = [256]uint8{
	GameLinkSteam: 1,
	GameLinkEpic:  2,
	GameLinkGOG:   3,
}

type GameLink struct {
	Type GameLinkType `json:"category"`
	URL  string       `json:"url"`
}

// GetGameLinks gets a Twitch game's links to other services. Results are returned in this order,
// with unknown matches removed:
//
//     - Steam
//     - Epic
//     - GOG
func (t *Twitch) GetGameLinks(ctx context.Context, twitchCategory int64) ([]GameLink, error) {
	// TODO: should this use the "websites" instead?
	query := fmt.Sprintf(`fields external_games.category, external_games.url; where external_games.category = %d & external_games.uid = "%d"; limit 1;`, GameLinkTwitch, twitchCategory)

	resp, err := t.helixCli.PostRaw(ctx, `https://api.igdb.com/v4/games`, strings.NewReader(query))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	var body []struct {
		ExternalGames []GameLink `json:"external_games"`
	}

	if err := jsonx.DecodeSingle(resp.Body, &body); err != nil {
		return nil, ErrServerError
	}

	if len(body) == 0 {
		return nil, ErrNotFound
	}

	links := body[0].ExternalGames

	// Filter from https://github.com/golang/go/wiki/SliceTricks.
	n := 0
	for _, x := range links {
		if x.URL != "" {
			links[n] = x
			n++
		}
	}
	links = links[:n]

	if len(links) == 0 {
		return nil, ErrNotFound
	}

	sort.SliceStable(links, func(i, j int) bool {
		io := gameLinkOrdering[links[i].Type]
		jo := gameLinkOrdering[links[j].Type]

		switch {
		case io == jo:
			return false
		case io == 0:
			return false
		case jo == 0:
			return true
		default:
			return io < jo
		}
	})

	// Unwanted links are at the end; slice them off.
	end := sort.Search(len(links), func(i int) bool {
		return gameLinkOrdering[links[i].Type] == 0
	})

	links = links[:end]

	if len(links) == 0 {
		return nil, ErrNotFound
	}

	return links, nil
}
