package twitch

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/goware/urlx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/petoem/cleanurl"
)

type externalGameType uint8

//go:generate go tool golang.org/x/tools/cmd/stringer -type=GameLinkType -trimprefix=GameLink

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
// fields external_games.external_game_source, external_games.url;
// where external_games.external_game_source = 6;
// limit 10;

const (
	_ externalGameType = iota
	externalGameSteam
	externalGameGamesDB
	externalGameGiantBomb
	externalGameGoOptimuz
	externalGameGOG
	externalGamePushSquare
	externalGameIsThereAnyDeal
	externalGameGamersGate
	_
	externalGameYouTube
	externalGameMicrosoft
	externalGameNintendoLife
	externalGameApple
	externalGameTwitch
	externalGameAndroid
	externalGamePlaystation
	_
	externalGameXbox
	externalGameGamersPress
	externalGameAmazon
	externalGameNintendo
	externalGamePlayAmazon
	_ // play.amazon.com
	externalGamePlayAsia
	externalGameTapTap
	externalGameEpic
	externalGameTouchArcade
	externalGameOculus
)

func externalGameToLink(g externalGameType) GameLinkType {
	switch g { //nolint:exhaustive
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
	ExternalGameSource externalGameType `json:"external_game_source"`
	URL                string           `json:"url"`
}

type gameWebsiteType uint8

const (
	_ gameWebsiteType = iota
	gameWebsiteOfficial
	gameWebsiteFandom
	gameWebsiteWikipedia
	gameWebsiteFacebook
	gameWebsiteTwitter
	gameWebsiteTwitch
	_
	gameWebsiteInstagram
	gameWebsiteYouTube
	gameWebsiteAppleIPhone
	gameWebsiteAppleIPad
	gameWebsiteAndroid
	gameWebsiteSteam
	gameWebsiteReddit
	gameWebsiteItch
	gameWebsiteEpic
	gameWebsiteGOG
	gameWebsiteDiscord
)

type gameWebsite struct {
	Type gameWebsiteType `json:"type"`
	URL  string          `json:"url"`
}

func websiteToLink(g gameWebsiteType) GameLinkType {
	switch g { //nolint:exhaustive
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

const gameLinkQuery = `fields websites.type, websites.url, external_games.external_game_source, external_games.url; where external_games.external_game_source = %d & external_games.uid = "%d"; limit 1;`

// GetGameLinks gets a Twitch game's links to other services. Results are returned in this order,
// with unknown matches removed:
//
//   - Steam
//   - Epic
//   - GOG
func (t *Twitch) GetGameLinks(ctx context.Context, twitchCategory int64) ([]GameLink, error) {
	query := fmt.Sprintf(gameLinkQuery, externalGameTwitch, twitchCategory)

	var body []struct {
		Websites      []gameWebsite  `json:"websites"`
		ExternalGames []externalGame `json:"external_games"`
	}

	req, err := t.helixCli.NewRequestToJSON(ctx, "https://api.igdb.com/v4/games", &body)
	if err != nil {
		return nil, err
	}

	if err := req.BodyBytes([]byte(query)).Fetch(ctx); err != nil {
		return nil, apiclient.WrapRequestErr("twitch", err, nil)
	}

	if len(body) == 0 {
		return nil, apiclient.NewStatusError("twitch", http.StatusNotFound)
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
		typ := externalGameToLink(e.ExternalGameSource)
		if e.URL != "" && typ != 0 {
			if _, ok := linkMap[typ]; !ok {
				linkMap[typ] = e.URL
			}
		}
	}

	links := make([]GameLink, 0, len(linkMap))

	for typ, u := range linkMap {
		parsed, err := urlx.Parse(u)
		if err == nil {
			cleanurl.CleanURL(parsed)
			u = parsed.String()
		}

		links = append(links, GameLink{
			Type: typ,
			URL:  u,
		})
	}

	if len(links) == 0 {
		return nil, apiclient.NewStatusError("twitch", http.StatusNotFound)
	}

	sort.Slice(links, func(i, j int) bool { return links[i].Type < links[j].Type })

	return links, nil
}
