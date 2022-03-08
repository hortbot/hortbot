// Package findlinks provides functions to find links in text.
package findlinks

import (
	"net/url"
	"regexp"

	"github.com/goware/urlx"
	"mvdan.cc/xurls/v2"
)

var linkRegex = func() *regexp.Regexp {
	reStr := xurls.Relaxed().String()

	// This used to be required before xurls v2.3.0 to ensure that we don't
	// match domains within other strings, but not doing this seems to slow
	// down Find by anywhere from 2-3x.
	reStr = `\b(?:` + reStr + `)\b`

	re := regexp.MustCompile(reStr)
	re.Longest()
	return re
}()

// Find searches for links in a given message. If schemeWhitelist items are
// included, then the results will only include links with the provided
// schemes.
func Find(message string, schemeWhitelist ...string) []*url.URL {
	if !linkRegex.MatchString(message) {
		// Fast path to check for a message without links. Increases
		// overall runtime, but most messages do not contain links and
		// FindAllString still does more work than this precheck.
		return nil
	}

	matches := linkRegex.FindAllString(message, -1)

	if len(matches) == 0 {
		panic("findlinks: no matches, but precheck matched")
	}

	urls := make([]*url.URL, 0, len(matches))

	for _, m := range matches {
		u, err := urlx.ParseWithDefaultScheme(m, "https")
		if err == nil {
			if len(schemeWhitelist) == 0 || inWhitelist(u.Scheme, schemeWhitelist) {
				urls = append(urls, u)
			}
		}
	}

	return urls
}

func inWhitelist(s string, whitelist []string) bool {
	for _, w := range whitelist {
		if s == w {
			return true
		}
	}
	return false
}
