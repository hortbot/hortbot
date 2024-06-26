module github.com/hortbot/hortbot

go 1.22

toolchain go1.22.0

require (
	github.com/alicebob/miniredis/v2 v2.33.0
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de
	github.com/bmatcuk/doublestar/v4 v4.6.1
	github.com/carlmjohnson/requests v0.23.5
	github.com/dghubble/trie v0.1.0
	github.com/dustin/go-humanize v1.0.1
	github.com/felixge/httpsnoop v1.0.4
	github.com/fortytw2/leaktest v1.3.0
	github.com/friendsofgo/errors v0.9.2
	github.com/go-chi/chi/v5 v5.0.14
	github.com/gobuffalo/flect v1.0.2
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang-migrate/migrate/v4 v4.17.1
	github.com/google/go-cmp v0.6.0
	github.com/gorilla/sessions v1.3.0
	github.com/goware/urlx v0.3.2
	github.com/hako/durafmt v0.0.0-20210608085754-5c1018a4e16b
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/jackc/pgx/v5 v5.6.0
	github.com/jakebailey/irc v0.0.0-20230110182717-9144941a856a
	github.com/jarcoal/httpmock v1.3.1
	github.com/jessevdk/go-flags v1.6.1
	github.com/joho/godotenv v1.5.1
	github.com/leononame/clock v0.1.6
	github.com/matryer/moq v0.3.4
	github.com/mroth/weightedrand/v2 v2.1.0
	github.com/nsqio/go-nsq v1.1.0
	github.com/ory/dockertest/v3 v3.10.0
	github.com/prometheus/client_golang v1.19.1
	github.com/redis/go-redis/v9 v9.5.3
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/xid v1.5.0
	github.com/tomwright/queryparam/v4 v4.1.0
	github.com/valyala/quicktemplate v1.7.0
	github.com/volatiletech/null/v8 v8.1.2
	github.com/volatiletech/sqlboiler/v4 v4.16.2
	github.com/volatiletech/strmangle v0.0.6
	github.com/wader/filtertransport v0.0.0-20200316221534-bdd9e61eee78
	github.com/ybbus/httpretry v1.0.2
	github.com/zikaeroh/ctxjoin v0.0.0-20240505042038-e54b3fa07c64
	github.com/zikaeroh/ctxlog v0.0.0-20210526021226-f475ac537d51
	go.deanishe.net/fuzzy v1.0.0
	go.uber.org/zap v1.27.0
	golang.org/x/net v0.26.0
	golang.org/x/oauth2 v0.21.0
	golang.org/x/sync v0.7.0
	golang.org/x/tools v0.22.0
	gotest.tools/v3 v3.5.1
	mvdan.cc/xurls/v2 v2.5.0
	nhooyr.io/websocket v1.8.11
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.2 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/containerd/continuity v0.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/docker/cli v20.10.17+incompatible // indirect
	github.com/docker/docker v24.0.9+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/ericlagergren/decimal v0.0.0-20211103172832-aca2edc11f73 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/magefile/mage v1.10.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.3-0.20211202183452-c5a74bcca799 // indirect
	github.com/opencontainers/runc v1.1.12 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/volatiletech/inflect v0.0.1 // indirect
	github.com/volatiletech/randomize v0.0.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/mod v0.18.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
