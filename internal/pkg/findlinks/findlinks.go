// Package findlinks provides functions to find links in text.
package findlinks

import (
	"net/url"
	"regexp"

	"github.com/goware/urlx"
	"mvdan.cc/xurls/v2"
)

var linkRegex = func() *regexp.Regexp {
	re := regexp.MustCompile(`\b` + xurls.Relaxed().String() + `\b`)
	re.Longest()
	return re
}()

// Find searches for links in a given message. If schemeWhitelist items are
// included, then the results will only include links with the provided
// schemes.
func Find(message string, schemeWhitelist ...string) []*url.URL {
	matches := linkRegex.FindAllString(message, -1)

	if len(matches) == 0 {
		return nil
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
