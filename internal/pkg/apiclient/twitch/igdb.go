package twitch

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
)

type externalGameType uint8

//go:generate go run golang.org/x/tools/cmd/stringer -type=GameLinkType -trimprefix=GameLink

type GameLinkType uint8

const (
	_ GameLinkType = iota
	GameLinkSteam
	GameLinkEpic
	GameLinkGOG
	GameLinkItch
	GameLinkOfficial
)

type GameLink struct {
	Type GameLinkType `json:"category"`
	URL  string       `json:"url"`
}

// Most of these are undocumented, but discoverable by doing something like:
// fields external_games.category, external_games.url;
// where external_games.category = 6;
// limit 10;

const (
	_ externalGameType = iota
	externalGameSteam
	externalGameGamesDB   //nolint
	externalGameGiantBomb //nolint
	externalGameGoOptimuz //nolint
	externalGameGOG
	externalGamePushSquare     //nolint
	externalGameIsThereAnyDeal //nolint
	externalGameGamersGate     //nolint
	_
	externalGameYouTube      //nolint
	externalGameMicrosoft    //nolint
	externalGameNintendoLife //nolint
	externalGameApple        //nolint
	externalGameTwitch
	externalGameAndroid     //nolint
	externalGamePlaystation //nolint
	_
	externalGameXbox        //nolint
	externalGameGamersPress //nolint
	externalGameAmazon      //nolint
	externalGameNintendo    //nolint
	externalGamePlayAmazon  //nolint
	_                       // play.amazon.com
	externalGamePlayAsia    //nolint
	externalGameTapTap      //nolint
	externalGameEpic
	externalGameTouchArcade //nolint
	externalGameOculus      //nolint
)

func externalGameToLink(g externalGameType) GameLinkType {
	switch g {
	case externalGameSteam:
		return GameLinkSteam
	case externalGameGOG:
		return GameLinkGOG
	case externalGameEpic:
		return GameLinkEpic
	}
	return 0
}

type externalGame struct {
	Type externalGameType `json:"category"`
	URL  string           `json:"url"`
}

type gameWebsiteType uint8

const (
	_ gameWebsiteType = iota
	gameWebsiteOfficial
	gameWebsiteFandom    //nolint
	gameWebsiteWikipedia //nolint
	gameWebsiteFacebook  //nolint
	gameWebsiteTwitter   //nolint
	gameWebsiteTwitch    //nolint
	_
	gameWebsiteInstagram   //nolint
	gameWebsiteYouTube     //nolint
	gameWebsiteAppleIPhone //nolint
	gameWebsiteAppleIPad   //nolint
	gameWebsiteAndroid     //nolint
	gameWebsiteSteam
	gameWebsiteReddit //nolint
	gameWebsiteItch
	gameWebsiteEpic
	gameWebsiteGOG
	gameWebsiteDiscord //nolint
)

type gameWebsite struct {
	Type gameWebsiteType `json:"category"`
	URL  string          `json:"url"`
}

func websiteToLink(g gameWebsiteType) GameLinkType {
	switch g {
	case gameWebsiteOfficial:
		return GameLinkOfficial
	case gameWebsiteSteam:
		return GameLinkSteam
	case gameWebsiteItch:
		return GameLinkItch
	case gameWebsiteEpic:
		return GameLinkEpic
	case gameWebsiteGOG:
		return GameLinkGOG
	}
	return 0
}

const gameLinkQuery = `fields websites.category, websites.url, external_games.category, external_games.url; where external_games.category = %d & external_games.uid = "%d"; limit 1;`

// GetGameLinks gets a Twitch game's links to other services. Results are returned in this order,
// with unknown matches removed:
//
//   - Steam
//   - Epic
//   - GOG
func (t *Twitch) GetGameLinks(ctx context.Context, twitchCategory int64) ([]GameLink, error) {
	query := fmt.Sprintf(gameLinkQuery, externalGameTwitch, twitchCategory)

	resp, err := t.helixCli.PostRaw(ctx, `https://api.igdb.com/v4/games`, strings.NewReader(query))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	var body []struct {
		Websites      []gameWebsite  `json:"websites"`
		ExternalGames []externalGame `json:"external_games"`
	}

	if err := jsonx.DecodeSingle(resp.Body, &body); err != nil {
		return nil, ErrServerError
	}

	if len(body) == 0 {
		return nil, ErrNotFound
	}

	linkMap := make(map[GameLinkType]string)

	for _, w := range body[0].Websites {
		typ := websiteToLink(w.Type)
		if w.URL != "" && typ != 0 {
			if _, ok := linkMap[typ]; !ok {
				linkMap[typ] = w.URL
			}
		}
	}

	for _, e := range body[0].ExternalGames {
		typ := externalGameToLink(e.Type)
		if e.URL != "" && typ != 0 {
			if _, ok := linkMap[typ]; !ok {
				linkMap[typ] = e.URL
			}
		}
	}

	links := make([]GameLink, 0, len(linkMap))

	for typ, url := range linkMap {
		links = append(links, GameLink{
			Type: typ,
			URL:  url,
		})
	}

	if len(links) == 0 {
		return nil, ErrNotFound
	}

	sort.Slice(links, func(i, j int) bool { return links[i].Type < links[j].Type })

	return links, nil
}
