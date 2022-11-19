package useragent

// https://github.com/intoli/user-agents

import (
	"bytes"
	"compress/gzip"
	_ "embed" // For go:embed.

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"github.com/hortbot/hortbot/internal/version"
	"github.com/mroth/weightedrand/v2"
)

//go:embed user-agents.json.gz
var agents []byte

type entry struct {
	AppName    string `json:"appName"`
	Connection struct {
		Downlink      float64 `json:"downlink"`
		EffectiveType string  `json:"effectiveType"`
		RTT           int     `json:"rtt"`
	} `json:"connection"`
	Platform       string  `json:"platform"`
	PluginsLength  int     `json:"pluginsLength"`
	Vendor         string  `json:"vendor"`
	UserAgent      string  `json:"userAgent"`
	ViewportHeight int     `json:"viewportHeight"`
	ViewportWidth  int     `json:"viewportWidth"`
	DeviceCategory string  `json:"deviceCategory"`
	ScreenHeight   int     `json:"screenHeight"`
	ScreenWidth    int     `json:"screenWidth"`
	Weight         float64 `json:"weight"`
}

const minThreshold = 100

var chooser *weightedrand.Chooser[string, int]

func init() {
	r, err := gzip.NewReader(bytes.NewReader(agents))
	if err != nil {
		panic(err)
	}

	var entries []*entry

	if err := jsonx.DecodeSingle(r, &entries); err != nil {
		panic(err)
	}

	minWeight := entries[0].Weight
	for _, e := range entries {
		if e.Weight < minWeight {
			minWeight = e.Weight
		}
	}

	factor := 1.0
	for {
		if minWeight*factor > minThreshold {
			break
		}
		factor *= 10
	}

	choices := make([]weightedrand.Choice[string, int], 0, len(entries))
	for _, e := range entries {
		if e.DeviceCategory != "desktop" {
			continue
		}

		choice := weightedrand.NewChoice(e.UserAgent, int(factor*e.Weight))
		choices = append(choices, choice)
	}

	chooser, err = weightedrand.NewChooser(choices...)
	if err != nil {
		panic(err)
	}
}

var botUserAgent = "HortBot/" + version.Version()

// Bot returns the bot's user agent string.
func Bot() string {
	return botUserAgent
}

// Browser returns a random browser user agent.
func Browser() string {
	return chooser.Pick()
}
