package templates

import (
	"fmt"

	"github.com/a-h/templ"
)

func showVarValueStyle(font, color string) templ.SafeCSS {
	return buildStyle(font, color)
}

func showVarLabelStyle(font, color string) templ.SafeCSS {
	return buildStyle(font, color)
}

func buildStyle(font, color string) templ.SafeCSS {
	var css string
	if font != "" {
		css += fmt.Sprintf(`font-family: %q, sans-serif !important;`, font)
	}
	if color != "" {
		css += fmt.Sprintf("color: %s !important;", templ.EscapeString(color))
	}
	return templ.SafeCSS(css)
}
