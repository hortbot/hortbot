//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/friendsofgo/errors"
)

func currentDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

func main() {
	if err := mainErr(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

const migrationTemplate = `BEGIN;

-- %s

COMMIT;
`

func mainErr() error {
	migrationsPath := filepath.Join(currentDir(), "static")

	now := time.Now().Format("20060102150405")
	name := os.Args[1]

	upPath := filepath.Join(migrationsPath, now+"_"+name+"_up.sql")
	downPath := filepath.Join(migrationsPath, now+"_"+name+"_down.sql")

	upFile, err := os.Create(upPath)
	if err != nil {
		return errors.Wrap(err, "creating up migration")
	}
	defer upFile.Close()

	downFile, err := os.Create(downPath)
	if err != nil {
		return errors.Wrap(err, "creating down migration")
	}
	defer downFile.Close()

	if _, err = fmt.Fprintf(upFile, migrationTemplate, "Up migration"); err != nil {
		return errors.Wrap(err, "writing up migration")
	}

	if _, err = fmt.Fprintf(downFile, migrationTemplate, "Down migration"); err != nil {
		return errors.Wrap(err, "writing down migration")
	}

	fmt.Println("Created migrations:")
	fmt.Println(upPath)
	fmt.Println(downPath)

	return nil
}
