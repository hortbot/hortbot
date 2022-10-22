// genmodels generates hortbot's database models package. It spawns a postgres
// container, migrates it up, then runs sqlboiler's internal generation code
// directly to write the models.
//
// If the migrations have changed, ensure that the migrations package has been
// re-generated via go generate.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/friendsofgo/errors"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/docker/dpostgres"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/volatiletech/sqlboiler/v4/boilingcore"
	_ "github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-psql/driver" // For the SQLBoiler psql driver.
	"github.com/volatiletech/sqlboiler/v4/importers"
)

func main() {
	if err := mainErr(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func mainErr() error {
	fmt.Println("Creating postgres database")
	db, connStr, cleanup, err := dpostgres.NewNoMigrate()
	if err != nil {
		return errors.Wrap(err, "creating database")
	}
	defer cleanup()

	if err := db.Close(); err != nil {
		return errors.Wrap(err, "closing initial database connection")
	}

	fmt.Println("Migrating database up")
	if err := migrations.Up(connStr, migrateLogf); err != nil {
		return errors.Wrap(err, "migrating database")
	}

	pgConf, err := pgconn.ParseConfig(connStr)
	if err != nil {
		return errors.Wrap(err, "parsing database config")
	}

	wd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "getting working directory")
	}

	dbPath := filepath.Join(wd, "internal", "db")
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist", dbPath)
		}
		return errors.Wrap(err, "stat-ing db path")
	}

	modelsPath := filepath.Join(dbPath, "models")
	fmt.Println("Generating models in", modelsPath)

	bConf := &boilingcore.Config{
		DriverName:      "psql",
		OutFolder:       modelsPath,
		PkgName:         "models",
		NoTests:         true,
		NoHooks:         true,
		NoRowsAffected:  true,
		Wipe:            true,
		StructTagCasing: "snake",
		RelationTag:     "-",
		DriverConfig: map[string]interface{}{
			"dbname":    pgConf.Database,
			"host":      pgConf.Host,
			"port":      int(pgConf.Port),
			"user":      pgConf.User,
			"pass":      pgConf.Password,
			"sslmode":   "disable",
			"blacklist": []string{"schema_migrations"},
		},
		Imports: importers.NewDefaultImports(),
		Version: sqlboilerVersion(),
	}

	state, err := boilingcore.New(bConf)
	if err != nil {
		return errors.Wrap(err, "processing sqlboiler config")
	}

	if err := state.Run(); err != nil {
		return errors.Wrap(err, "running sqlboiler")
	}

	if err := state.Cleanup(); err != nil {
		return errors.Wrap(err, "cleaning up sqlboiler")
	}

	// Create is fine, since the above code wipes the models directory.
	docFile, err := os.Create(filepath.Join(modelsPath, "doc.go"))
	if err != nil {
		return errors.Wrap(err, "creating doc.go")
	}

	fmt.Fprintln(docFile, "// Package models implements an ORM generated from the HortBot Postgres database.")
	fmt.Fprintln(docFile, "package", bConf.PkgName)

	if err := docFile.Close(); err != nil {
		return errors.Wrap(err, "closing doc.go")
	}

	return nil
}

func migrateLogf(format string, v ...interface{}) {
	fmt.Printf("\t"+format, v...)
}

func sqlboilerVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	for _, mod := range info.Deps {
		if mod.Path == "github.com/volatiletech/sqlboiler/v4" {
			if mod.Replace != nil {
				return ""
			}
			return mod.Version
		}
	}

	return ""
}
