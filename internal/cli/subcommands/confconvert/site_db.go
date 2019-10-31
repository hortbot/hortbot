package confconvert

import (
	"context"
	"sort"
	"time"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/ory/dockertest"
)

var (
	siteDB  *sqlx.DB
	daySecs = uint((24 * time.Hour).Seconds())
)

func (cmd *cmd) prepareSiteDB(ctx context.Context) func() {
	logger := ctxlog.FromContext(ctx)

	pool, err := dockertest.NewPool("")
	if err != nil {
		logger.Fatal("error creating dockertest pool", PlainError(err))
	}

	const (
		password = "password"
		dbName   = "db"
	)

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mariadb",
		Tag:        "10.1",
		Env:        []string{"MYSQL_ROOT_PASSWORD=" + password, "MYSQL_DATABASE=" + dbName},
		Mounts:     []string{cmd.SiteDumps + ":/docker-entrypoint-initdb.d"},
	})
	if err != nil {
		logger.Fatal("error creating MariaDB container", PlainError(err))
	}

	if err := resource.Expire(daySecs); err != nil {
		logger.Fatal("error setting container expiration", PlainError(err))
	}

	connStr := "root:" + password + "@tcp(" + resource.GetHostPort("3306/tcp") + ")/" + dbName

	err = pool.Retry(func() error {
		var err error
		siteDB, err = sqlx.Open("mysql", connStr)
		if err != nil {
			return err
		}
		return siteDB.Ping()
	})
	if err != nil {
		logger.Fatal("error waiting for database to be ready", PlainError(err))
	}

	return func() {
		_ = siteDB.Close()
		_ = pool.Purge(resource)
	}
}

func getSiteInfo(ctx context.Context, name string) (string, bool, error) {
	row := &struct {
		BotName string        `db:"botChannel"`
		Active  types.BitBool `db:"isActive"`
	}{}

	if err := siteDB.Get(row, "SELECT botChannel, isActive FROM site_channels WHERE channel=?", name); err != nil {
		return "", false, err
	}

	return row.BotName, bool(row.Active), nil
}

func getVariables(ctx context.Context, channelName string) ([]*models.Variable, error) {
	var rows []struct {
		Name         string `db:"var"`
		Value        string `db:"value"`
		LastModified int64  `db:"lastModified"`
	}

	if err := siteDB.Select(&rows, "SELECT var, value, lastModified FROM site_vars WHERE channel=?", channelName); err != nil {
		return nil, err
	}

	variables := make([]*models.Variable, len(rows))

	for i, row := range rows {
		t := time.Unix(row.LastModified, 0)

		variables[i] = &models.Variable{
			Name:      row.Name,
			Value:     row.Value,
			CreatedAt: t,
			UpdatedAt: t,
		}
	}

	sort.Slice(variables, func(i, j int) bool {
		return variables[i].Name < variables[j].Name
	})

	return variables, nil
}
