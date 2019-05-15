module github.com/hortbot/hortbot

go 1.12

require (
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/cenkalti/backoff v2.1.1+incompatible // indirect
	github.com/containerd/continuity v0.0.0-20190426062206-aaeac12a7ffc // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/fortytw2/leaktest v1.3.0
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang-migrate/migrate/v4 v4.3.1
	github.com/google/go-cmp v0.3.0
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/jakebailey/irc v0.0.0-20190407213833-8d2a5d226230
	github.com/joho/godotenv v1.3.0
	github.com/lib/pq v1.1.1
	github.com/maxbrunsfeld/counterfeiter/v6 v6.0.2
	github.com/mjibson/esc v0.2.0
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/ory/dockertest v3.3.4+incompatible
	github.com/pkg/errors v0.8.1
	github.com/spf13/cast v1.3.0 // indirect
	github.com/volatiletech/inflect v0.0.0-20170731032912-e7201282ae8d // indirect
	github.com/volatiletech/null v8.0.0+incompatible
	github.com/volatiletech/sqlboiler v3.2.0+incompatible
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/net v0.0.0-20190514140710-3ec191127204 // indirect
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20190514135907-3a4b5fb9f71f // indirect
	golang.org/x/tools v0.0.0-20190515035509-2196cb7019cc // indirect
	google.golang.org/genproto v0.0.0-20190513181449-d00d292a067c // indirect
	gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.3 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
	gotest.tools v2.2.0+incompatible
)

replace gopkg.in/DATA-DOG/go-sqlmock.v1 => github.com/DATA-DOG/go-sqlmock v1.3.3

replace github.com/maxbrunsfeld/counterfeiter => github.com/maxbrunsfeld/counterfeiter/v6 v6.0.2
