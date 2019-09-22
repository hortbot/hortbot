package cmdargs

import "time"

type Common struct {
	Debug bool `long:"debug" env:"HB_DEBUG" description:"Enables debug mode and the debug log level"`
}

var DefaultCommon = Common{}

type IRC struct {
	Nick         string        `long:"nick" env:"HB_NICK" description:"IRC nick" required:"true"`
	Pass         string        `long:"pass" env:"HB_PASS" description:"IRC pass" required:"true"`
	PingInterval time.Duration `long:"ping-interval" env:"HB_PING_INTERVAL" description:"How often to ping the IRC server"`
	PingDeadline time.Duration `long:"ping-deadline" env:"HB_PING_DEADLINE" description:"How long to wait for a PONG before disconnecting"`
}

var DefaultIRC = IRC{
	PingInterval: 5 * time.Minute,
	PingDeadline: 5 * time.Second,
}

type SQL struct {
	DB        string `long:"db" env:"HB_DB" description:"PostgresSQL connection string" required:"true"`
	MigrateUp bool   `long:"migrate-up" env:"HB_MIGRATE_UP" description:"Migrates the postgres database up"`
}

var DefaultSQL = SQL{}

type Redis struct {
	RedisAddr string `long:"redis-addr" env:"HB_REDIS_ADDR" description:"Redis address" required:"true"`
}

var DefaultRedis = Redis{}

type Bot struct {
	Admins []string `long:"admin" env:"HB_ADMINS" env-delim:"," description:"Bot admins"`

	WhitelistEnabled bool     `long:"whitelist-enabled" env:"HB_WHITELIST_ENABLED" description:"Enable the user whitelist"`
	Whitelist        []string `long:"whitelist" env:"HB_WHITELIST" env-delim:"," description:"User whitelist"`

	DefaultCooldown int `long:"default-cooldown" env:"HB_DEFAULT_COOLDOWN" description:"default command cooldown"`

	BotWebAddr    string            `long:"bot-web-addr" env:"HB_BOT_WEB_ADDR" description:"Default address for the bot website"`
	BotWebAddrMap map[string]string `long:"bot-web-addr-map" env:"HB_BOT_WEB_ADDR_MAP" description:"Bot name to web address mapping"`
}

var DefaultBot = Bot{
	DefaultCooldown: 5,
	BotWebAddr:      "http://localhost:5000",
}

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

type RateLimit struct {
	RateLimitSlow   int           `long:"rate-limit-slow" env:"HB_RATE_LIMIT_RATE" description:"Message allowed per rate limit period (slow)"`
	RateLimitFast   int           `long:"rate-limit-fast" env:"HB_RATE_LIMIT_RATE" description:"Message allowed per rate limit period (fast)"`
	RateLimitPeriod time.Duration `long:"rate-limit-period" env:"HB_RATE_LIMIT_PERIOD" description:"Rate limit period"`
}

var DefaultRateLimit = RateLimit{
	RateLimitSlow:   15,
	RateLimitFast:   80,
	RateLimitPeriod: 30 * time.Second,
}

type LastFM struct {
	LastFMKey string `long:"lastfm-key" env:"HB_LASTFM_KEY" description:"LastFM API key"`
}

var DefaultLastFM = LastFM{}

type Twitch struct {
	TwitchClientID     string `long:"twitch-client-id" env:"HB_TWITCH_CLIENT_ID" description:"Twitch OAuth client ID" required:"true"`
	TwitchClientSecret string `long:"twitch-client-secret" env:"HB_TWITCH_CLIENT_SECRET" description:"Twitch OAuth client secret" required:"true"`
	TwitchRedirectURL  string `long:"twitch-redirect-url" env:"HB_TWITCH_REDIRECT_URL" description:"Twitch OAuth redirect URL" required:"true"`
}

var DefaultTwitch = Twitch{}

type Steam struct {
	SteamKey string `long:"steam-key" env:"HB_STEAM_KEY" description:"Steam API key"`
}

var DefaultSteam = Steam{}

type NSQ struct {
	NSQAddr    string `long:"nsq-addr" env:"HB_NSQ_ADDR" description:"NSQD address" required:"true"`
	NSQChannel string `long:"nsq-channel" env:"HB_NSQ_CHANNEL" description:"NSQ subscription channel"`
}

var DefaultNSQ = NSQ{
	NSQChannel: "queue",
}

type Jaeger struct {
	JaegerAgent string `long:"jaeger-agent" env:"HB_JAEGER_AGENT" description:"jaeger agent address"`
}

var DefaultJaeger = Jaeger{}
