//go:build tools
// +build tools

package tools

import (
	_ "github.com/matryer/moq"
	_ "github.com/valyala/quicktemplate/qtc"
	_ "golang.org/x/tools/cmd/stringer"
)
