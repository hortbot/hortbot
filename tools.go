//go:build tools
// +build tools

package tools

import (
	_ "github.com/99designs/gqlgen"
	_ "github.com/golang-migrate/migrate/v4/cmd/migrate"
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
	_ "github.com/valyala/quicktemplate/qtc"
	_ "golang.org/x/tools/cmd/stringer"
)
