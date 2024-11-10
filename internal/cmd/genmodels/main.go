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
	"strings"

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
		return fmt.Errorf("creating database: %w", err)
	}
	defer cleanup()

	if err := db.Close(); err != nil {
		return fmt.Errorf("closing initial database connection: %w", err)
	}

	fmt.Println("Migrating database up")
	if err := migrations.Up(connStr, migrateLogf); err != nil {
		return fmt.Errorf("migrating database: %w", err)
	}

	pgConf, err := pgconn.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("parsing database config: %w", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	dbPath := filepath.Join(wd, "internal", "db")
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist", dbPath)
		}
		return fmt.Errorf("stat-ing db path: %w", err)
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
		DriverConfig: map[string]any{
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
		return fmt.Errorf("processing sqlboiler config: %w", err)
	}

	if err := state.Run(); err != nil {
		return fmt.Errorf("running sqlboiler: %w", err)
	}

	if err := state.Cleanup(); err != nil {
		return fmt.Errorf("cleaning up sqlboiler: %w", err)
	}

	// Create is fine, since the above code wipes the models directory.
	docFile, err := os.Create(filepath.Join(modelsPath, "doc.go"))
	if err != nil {
		return fmt.Errorf("creating doc.go: %w", err)
	}

	fmt.Fprintln(docFile, "// Package models implements an ORM generated from the HortBot Postgres database.")
	fmt.Fprintln(docFile, "package", bConf.PkgName)

	if err := docFile.Close(); err != nil {
		return fmt.Errorf("closing doc.go: %w", err)
	}

	return nil
}

func migrateLogf(format string, v ...any) {
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
			return strings.TrimPrefix(mod.Version, "v")
		}
	}

	return ""
}
