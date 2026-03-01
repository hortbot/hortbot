package templates

import (
	"regexp"
	"strings"

	"github.com/a-h/templ"
)

var validCSSFont = regexp.MustCompile(`^[a-zA-Z0-9 _\-',]+$`)
var validCSSColor = regexp.MustCompile(`^[a-zA-Z0-9#(), .%]+$`)

func showVarValueStyle(font, color string) templ.SafeCSS {
	return buildStyle(font, color)
}

func showVarLabelStyle(font, color string) templ.SafeCSS {
	return buildStyle(font, color)
}

func buildStyle(font, color string) templ.SafeCSS {
	var b strings.Builder
	if font != "" && validCSSFont.MatchString(font) {
		b.WriteString(`font-family: "`)
		b.WriteString(font)
		b.WriteString(`", sans-serif !important;`)
	}
	if color != "" && validCSSColor.MatchString(color) {
		b.WriteString("color: ")
		b.WriteString(color)
		b.WriteString(" !important;")
	}
	return templ.SafeCSS(b.String())
}
