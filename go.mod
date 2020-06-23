module github.com/hortbot/hortbot

go 1.13

require (
	contrib.go.opencensus.io/exporter/jaeger v0.2.0
	contrib.go.opencensus.io/integrations/ocsql v0.1.6
	github.com/99designs/gqlgen v0.11.3
	github.com/alicebob/miniredis/v2 v2.12.0
	github.com/antchfx/htmlquery v1.2.3
	github.com/araddon/dateparse v0.0.0-20200409225146-d820a6159ab1
	github.com/bmatcuk/doublestar v1.3.1
	github.com/dghubble/trie v0.0.0-20200219060618-c42a287caf69
	github.com/dustin/go-humanize v1.0.0
	github.com/ericlagergren/decimal v0.0.0-20191206042408-88212e6cfca9 // indirect
	github.com/felixge/httpsnoop v1.0.1
	github.com/fortytw2/leaktest v1.3.0
	github.com/friendsofgo/errors v0.9.2
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/go-redis/redis/v8 v8.0.0-beta.5
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gobuffalo/flect v0.2.1
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/golang-migrate/migrate/v4 v4.11.0
	github.com/google/go-cmp v0.5.0
	github.com/gorilla/sessions v1.2.0
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/goware/urlx v0.3.1
	github.com/hako/durafmt v0.0.0-20200605151348-3a43fc422dd9
	github.com/jackc/pgconn v1.6.0
	github.com/jackc/pgx/v4 v4.6.0
	github.com/jakebailey/irc v0.0.0-20190904051515-2d11e69506b0
	github.com/jarcoal/httpmock v1.0.5
	github.com/jessevdk/go-flags v1.4.1-0.20181221193153-c0795c8afcf4
	github.com/jmoiron/sqlx v1.2.0
	github.com/joho/godotenv v1.3.0
	github.com/leononame/clock v0.1.6
	github.com/markbates/pkger v0.17.0
	github.com/maxbrunsfeld/counterfeiter/v6 v6.2.3
	github.com/mitchellh/mapstructure v1.3.2 // indirect
	github.com/nsqio/go-nsq v1.0.8
	github.com/ory/dockertest/v3 v3.6.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/posener/ctxutil v1.0.0
	github.com/prometheus/client_golang v1.7.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/xid v1.2.1
	github.com/tomwright/queryparam/v4 v4.1.0
	github.com/valyala/quicktemplate v1.5.0
	github.com/vektah/gqlparser/v2 v2.0.1
	github.com/volatiletech/null/v8 v8.1.0
	github.com/volatiletech/sqlboiler/v4 v4.1.2
	github.com/volatiletech/strmangle v0.0.1
	github.com/wader/filtertransport v0.0.0-20200316221534-bdd9e61eee78
	github.com/zikaeroh/ctxjoin v0.0.0-20200613235025-e3d47af29310
	github.com/zikaeroh/ctxlog v0.0.0-20200613043947-8791c8613223
	go.opencensus.io v0.22.4
	go.uber.org/atomic v1.6.0
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.0.0-20200604202706-70a84ac30bf9 // indirect
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	golang.org/x/tools v0.0.0-20200619210111-0f592d2728bb
	gotest.tools/v3 v3.0.2
	mvdan.cc/xurls/v2 v2.2.0
)

replace github.com/markbates/pkger => github.com/zikaeroh/pkger v0.17.1-0.20200604025301-dceb832975ba
