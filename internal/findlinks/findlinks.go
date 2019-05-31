package findlinks

import (
	"net/url"

	"github.com/goware/urlx"
	"mvdan.cc/xurls/v2"
)

var linkRegex = xurls.Relaxed()

func Find(message string) []*url.URL {
	matches := linkRegex.FindAllString(message, -1)

	if len(matches) == 0 {
		return nil
	}

	urls := make([]*url.URL, 0, len(matches))

	for _, m := range matches {
		u, err := urlx.ParseWithDefaultScheme(m, "https")
		if err == nil {
			urls = append(urls, u)
		}
	}

	return urls
}
