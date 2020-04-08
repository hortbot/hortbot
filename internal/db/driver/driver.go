// Package driver defines the default PostgreSQL driver to use, to make it
// convenient to change for all users at once.
package driver

import _ "github.com/jackc/pgx/v4/stdlib" // Postgres driver, pulled in for side effects.

// Name is the name of a postgres driver, passable to sql.Open.
const Name = "pgx"
