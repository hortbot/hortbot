package linkmatch_test

import (
	"testing"

	"github.com/goware/urlx"
	"github.com/hortbot/hortbot/internal/pkg/linkmatch"
	"gotest.tools/v3/assert"
)

func TestHostAndPath(t *testing.T) {
	tests := []struct {
		pattern string
		link    string
		match   bool
	}{
		{
			pattern: "google.com",
			link:    "google.com",
			match:   true,
		},
		{
			pattern: "google.com",
			link:    "goo.gl",
			match:   false,
		},
		{
			pattern: "clips.twitch.tv",
			link:    "https://clips.twitch.tv/ModernDaintyZebraMikeHogu",
			match:   true,
		},
		{
			pattern: "twitch.tv",
			link:    "https://clips.twitch.tv/ModernDaintyZebraMikeHogu",
			match:   true,
		},
		{
			pattern: "clips.twitch.tv/",
			link:    "https://clips.twitch.tv/ModernDaintyZebraMikeHogu",
			match:   true,
		},
		{
			pattern: "twitch.tv/coestar",
			link:    "https://www.twitch.tv/coestar/clip/UglyBashfulEggnogLitFam?filter=clips&range=7d&sort=time",
			match:   true,
		},
		{
			pattern: "twitch.tv/*/clip",
			link:    "https://www.twitch.tv/coestar/clip/UglyBashfulEggnogLitFam?filter=clips&range=7d&sort=time",
			match:   true,
		},
		{
			pattern: "twitch.tv/*",
			link:    "https://www.twitch.tv/some/random/page",
			match:   true,
		},
		{
			pattern: "twitch.tv/**",
			link:    "https://www.twitch.tv/some/random/page",
			match:   true,
		},
		{
			pattern: "twitch.tv/some/*",
			link:    "https://www.twitch.tv/some/random/page",
			match:   true,
		},
		{
			pattern: "twitch.tv/*/random/*",
			link:    "https://www.twitch.tv/some/random/page",
			match:   true,
		},
		{
			pattern: "twitch.tv/*/random/*",
			link:    "twitch.tv/unrelated?what",
			match:   false,
		},
		{
			pattern: "twitch.tv/*/clip*",
			link:    "https://www.twitch.tv/coestar/clip/UglyBashfulEggnogLitFam?filter=clips&range=7d&sort=time",
			match:   true,
		},
		{
			pattern: "*:::::",
			link:    "https://www.twitch.tv/coestar/clip/UglyBashfulEggnogLitFam?filter=clips&range=7d&sort=time",
			match:   false,
		},
		{
			pattern: "twitch.tv/foo*",
			link:    "twitch.tv/foobar/",
			match:   true,
		},
		{
			pattern: "twitch.tv/foo*",
			link:    "almost1.tv/foobar/",
			match:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.pattern+" "+test.link, func(t *testing.T) {
			u, err := urlx.ParseWithDefaultScheme(test.link, "https")
			assert.NilError(t, err)

			match := linkmatch.HostAndPath(test.pattern, u)
			assert.Equal(t, test.match, match)
		})
	}
}

func BenchmarkHostAndPath(b *testing.B) {
	u, err := urlx.ParseWithDefaultScheme("https://www.twitch.tv/coestar/clip/UglyBashfulEggnogLitFam?filter=clips&range=7d&sort=time", "https")
	assert.NilError(b, err)

	b.Run("Wrong domain", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			linkmatch.HostAndPath("example.com/*/clip", u)
		}
	})

	b.Run("Match no path", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			linkmatch.HostAndPath("twitch.tv", u)
		}
	})

	b.Run("Match path prefix", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			linkmatch.HostAndPath("twitch.tv/coestar", u)
		}
	})

	b.Run("Middle glob", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			linkmatch.HostAndPath("twitch.tv/*/clip", u)
		}
	})
}

func TestIsBadPattern(t *testing.T) {
	tests := []struct {
		pattern string
		want    bool
	}{
		{"twitch.tv", false},
		{"", true},
		{"*", true},
		{"twitch.tv/*", false},
		{"**", true},
		{"*/*", true},
		{"https://*/*", true},
	}

	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			assert.Equal(t, linkmatch.IsBadPattern(test.pattern), test.want)
		})
	}
}
