// Package webargs proceses web server arguments.
package webargs

import (
	"database/sql"

	"github.com/hortbot/hortbot/internal/db/redis"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/web"
)

type Web struct {
	WebAddr       string            `long:"web-addr" env:"HB_WEB_ADDR" description:"Server address for the web server"`
	WebSessionKey string            `long:"web-session-key" env:"HB_WEB_SESSION_KEY" description:"Session cookie auth key"`
	WebBrand      string            `long:"web-brand" env:"HB_WEB_BRAND" description:"Web server default branding"`
	WebBrandMap   map[string]string `long:"web-brand-map" env:"HB_WEB_BRAND_MAP" env-delim:"," description:"Web server brand mapping from domains to branding (ex: 'example.com:SomeBot,other.net:WhoAmI')"`
}

var DefaultWeb = Web{
	WebAddr:       ":5000",
	WebSessionKey: "this-is-insecure-do-not-use-this",
	WebBrand:      "HortBot",
}

func (args *Web) WebApp(debug bool, rdb *redis.DB, db *sql.DB, tw *twitch.Twitch) *web.App {
	return &web.App{
		Addr:       args.WebAddr,
		SessionKey: []byte(args.WebSessionKey),
		Brand:      args.WebBrand,
		BrandMap:   args.WebBrandMap,
		Debug:      debug,
		Redis:      rdb,
		DB:         db,
		Twitch:     tw,
	}
}
