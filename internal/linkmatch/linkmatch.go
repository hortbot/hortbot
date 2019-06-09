package linkmatch

import (
	"net/url"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/goware/urlx"
)

func HostAndPath(pattern string, u *url.URL) bool {
	p, err := urlx.ParseWithDefaultScheme(pattern, "https")
	if err != nil {
		return false
	}

	switch {
	case u.Host == p.Host:
	case strings.HasSuffix(u.Host, "."+p.Host):
		// Host matches, continue.
	default:
		return false
	}

	pPath := strings.ToLower(strings.Trim(p.Path, "/"))
	uPath := strings.ToLower(strings.Trim(u.Path, "/"))

	if pPath == "" {
		return true
	}

	if !strings.ContainsRune(pPath, '*') && strings.HasPrefix(uPath, pPath) {
		return true
	}

	ppPath := pPath

	switch {
	case strings.HasSuffix(ppPath, "/**"):
		// Do nothing.
	case strings.HasSuffix(ppPath, "/*"):
		ppPath += "*"
	case strings.HasSuffix(ppPath, "**"):
		// Do nothing.
	case strings.HasSuffix(ppPath, "*"):
		ppPath += "*"
	default:
		ppPath += "/**"
	}

	if matched, err := doublestar.Match(ppPath, uPath); matched || err != nil {
		return matched
	}

	return false
}
