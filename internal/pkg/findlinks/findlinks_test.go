package findlinks_test

import (
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hortbot/hortbot/internal/pkg/findlinks"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestFind(t *testing.T) {
	tests := []struct {
		message   string
		whitelist []string
		want      []*url.URL
	}{
		{},
		{
			message: "Check out my cool website google.com",
			want: []*url.URL{
				{
					Scheme: "https",
					Host:   "google.com",
				},
			},
		},
		{
			message: "https://clips.twitch.tv/ModernDaintyZebraMikeHogu",
			want: []*url.URL{
				{
					Scheme: "https",
					Host:   "clips.twitch.tv",
					Path:   "/ModernDaintyZebraMikeHogu",
				},
			},
		},
		{
			message: "look https://www.twitch.tv/coestar/clip/UglyBashfulEggnogLitFam?filter=clips&range=7d&sort=time",
			want: []*url.URL{
				{
					Scheme:   "https",
					Host:     "www.twitch.tv",
					Path:     "/coestar/clip/UglyBashfulEggnogLitFam",
					RawQuery: "filter=clips&range=7d&sort=time",
				},
			},
		},
		{
			message: "twitch.tv is cool and http://github.com/hortbot/hortbot",
			want: []*url.URL{
				{
					Scheme: "https",
					Host:   "twitch.tv",
				},
				{
					Scheme: "http",
					Host:   "github.com",
					Path:   "/hortbot/hortbot",
				},
			},
		},
		{
			message: "yo look at https://youtu.be/dQw4w9WgXcQ",
			want: []*url.URL{
				{
					Scheme: "https",
					Host:   "youtu.be",
					Path:   "/dQw4w9WgXcQ",
				},
			},
		},
		{
			message:   "yo look at explicitly https://youtu.be/dQw4w9WgXcQ",
			whitelist: []string{"http", "https", "ftp"},
			want: []*url.URL{
				{
					Scheme: "https",
					Host:   "youtu.be",
					Path:   "/dQw4w9WgXcQ",
				},
			},
		},
		{
			message:   "this isn't here https://youtu.be/dQw4w9WgXcQ",
			whitelist: []string{"http"},
		},
		{
			message: "yo look at git://github.com/what.git",
			want: []*url.URL{
				{
					Scheme: "git",
					Host:   "github.com",
					Path:   "/what.git",
				},
			},
		},
		{
			message:   "this does not match git://github.com/what.git",
			whitelist: []string{"http", "https", "ftp"},
		},
		{
			message:   "don't delete config.ini as that'd be bad",
			whitelist: []string{"http", "https", "ftp"},
		},
		{
			message:   "My favorite site is example.com; it's really the place to be.",
			whitelist: []string{"http", "https", "ftp"},
			want: []*url.URL{
				{
					Scheme: "https",
					Host:   "example.com",
				},
			},
		},
		{
			message:   "thisisn'tgoogle.comoranything",
			whitelist: []string{"http", "https", "ftp"},
		},
		{
			message:   "中国.中国/中国",
			whitelist: []string{"http", "https", "ftp"},
			want: []*url.URL{
				{
					Scheme:  "https",
					Host:    "中国.中国",
					Path:    "/中国",
					RawPath: "/中国",
				},
			},
		},
	}

	for _, test := range tests {
		got := findlinks.Find(test.message, test.whitelist...)
		assert.Check(t, cmp.DeepEqual(test.want, got, cmpopts.EquateEmpty()), "%s", test.message)
	}
}

func BenchmarkFind(b *testing.B) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "long with links",
			message: "twitch.tv is cool and http://github.com/hortbot/hortbot look https://www.twitch.tv/coestar/clip/UglyBashfulEggnogLitFam?filter=clips&range=7d&sort=time huh",
		},
		{
			name:    "short no links",
			message: "LUL",
		},
		{
			name:    "medium no links",
			message: "This is a message, and it doesn't have any links. Isn't that interesting? I sure think so.",
		},
		{
			name:    "long no links",
			message: "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
		},
		{
			name:    "custom command",
			message: "!pan working command",
		},
	}

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				findlinks.Find(test.message)
			}
		})
	}
}
