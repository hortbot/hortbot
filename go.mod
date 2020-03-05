module github.com/hortbot/hortbot

go 1.13

require (
	contrib.go.opencensus.io/exporter/jaeger v0.2.0
	contrib.go.opencensus.io/integrations/ocsql v0.1.5
	github.com/DATA-DOG/go-sqlmock v1.4.1 // indirect
	github.com/alicebob/miniredis/v2 v2.11.3
	github.com/araddon/dateparse v0.0.0-20190622164848-0fb0a474d195
	github.com/bmatcuk/doublestar v1.2.2
	github.com/dghubble/trie v0.0.0-20200219060618-c42a287caf69
	github.com/dustin/go-humanize v1.0.0
	github.com/ericlagergren/decimal v0.0.0-20191206042408-88212e6cfca9 // indirect
	github.com/felixge/httpsnoop v1.0.1
	github.com/fortytw2/leaktest v1.3.0
	github.com/friendsofgo/errors v0.9.2
	github.com/go-chi/chi v4.0.3+incompatible
	github.com/go-redis/redis/v7 v7.2.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gobuffalo/flect v0.2.1
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang-migrate/migrate/v4 v4.9.1
	github.com/google/go-cmp v0.4.0
	github.com/gorilla/sessions v1.2.0
	github.com/goware/urlx v0.3.1
	github.com/hako/durafmt v0.0.0-20191009132224-3f39dc1ed9f4
	github.com/jackc/pgconn v1.3.2
	github.com/jackc/pgx/v4 v4.4.1
	github.com/jakebailey/irc v0.0.0-20190904051515-2d11e69506b0
	github.com/jarcoal/httpmock v1.0.4
	github.com/jessevdk/go-flags v1.4.1-0.20181221193153-c0795c8afcf4
	github.com/jmoiron/sqlx v1.2.0
	github.com/joho/godotenv v1.3.0
	github.com/leononame/clock v0.1.6
	github.com/maxbrunsfeld/counterfeiter/v6 v6.2.2
	github.com/mjibson/esc v0.2.0
	github.com/nsqio/go-nsq v1.0.8
	github.com/ory/dockertest/v3 v3.5.4
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/posener/ctxutil v1.0.0
	github.com/prometheus/client_golang v1.5.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/xid v1.2.1
	github.com/spf13/cast v1.3.1 // indirect
	github.com/tomwright/queryparam/v4 v4.1.0
	github.com/valyala/quicktemplate v1.4.2-0.20200112192417-6e4d18193077
	github.com/volatiletech/inflect v0.0.0-20170731032912-e7201282ae8d // indirect
	github.com/volatiletech/null v8.0.0+incompatible
	github.com/volatiletech/sqlboiler v3.6.1+incompatible
	go.opencensus.io v0.22.3
	go.uber.org/atomic v1.6.0
	go.uber.org/zap v1.14.0
	golang.org/x/crypto v0.0.0-20200302210943-78000ba7a073 // indirect
	golang.org/x/net v0.0.0-20200301022130-244492dfa37a
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/tools v0.0.0-20200304193943-95d2e580d8eb
	gotest.tools/v3 v3.0.2
	mvdan.cc/xurls/v2 v2.1.0
)
