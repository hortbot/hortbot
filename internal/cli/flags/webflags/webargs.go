// Package webflags proceses web server related flags.
package webflags

import (
	"database/sql"

	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/web"
)

type Web struct {
	Addr       string            `long:"web-addr" env:"HB_WEB_ADDR" description:"Server address for the web server"`
	SessionKey string            `long:"web-session-key" env:"HB_WEB_SESSION_KEY" description:"Session cookie auth key"`
	Brand      string            `long:"web-brand" env:"HB_WEB_BRAND" description:"Web server default branding"`
	BrandMap   map[string]string `long:"web-brand-map" env:"HB_WEB_BRAND_MAP" env-delim:"," description:"Web server brand mapping from domains to branding (ex: 'example.com:SomeBot,other.net:WhoAmI')"`
}

var DefaultWeb = Web{
	Addr:       ":5000",
	SessionKey: "this-is-insecure-do-not-use-this",
	Brand:      "HortBot",
}

func (args *Web) New(debug bool, rdb *redis.DB, db *sql.DB, tw *twitch.Twitch) *web.App {
	return &web.App{
		Addr:       args.Addr,
		SessionKey: []byte(args.SessionKey),
		Brand:      args.Brand,
		BrandMap:   args.BrandMap,
		Debug:      debug,
		Redis:      rdb,
		DB:         db,
		Twitch:     tw,
	}
}
