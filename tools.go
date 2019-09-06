// +build tools

package tools

import (
	_ "github.com/golang-migrate/migrate/v4/cmd/migrate"
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
	_ "github.com/mjibson/esc"
	_ "github.com/valyala/quicktemplate/qtc"
	_ "github.com/volatiletech/sqlboiler"
	_ "github.com/volatiletech/sqlboiler/drivers/sqlboiler-psql"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/cmd/stringer"
)
