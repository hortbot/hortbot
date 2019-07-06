package bot

import (
	"context"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hortbot/hortbot/internal/pkg/linkmatch"
)

var filters = []func(context.Context, *session) (filtered bool, err error){
	filterLinks,
	filterCaps,
	filterSymbols,
}

func tryFilter(ctx context.Context, s *session) (filtered bool, err error) {
	if !s.Channel.ShouldModerate || !s.Channel.EnableFilters {
		return false, nil
	}

	if s.UserLevel.CanAccess(levelSubscriber) {
		return false, nil
	}

	for _, fn := range filters {
		filtered, err := fn(ctx, s)
		if filtered || err != nil {
			return filtered, err
		}
	}

	return false, nil
}

func filterLinks(ctx context.Context, s *session) (filtered bool, err error) {
	if !s.Channel.FilterLinks {
		return false, nil
	}

	links := s.Links()

	if len(links) == 0 {
		return false, nil
	}

	if allLinksPermitted(s.Channel.PermittedLinks, links) {
		return false, nil
	}

	permitted, err := s.HasLinkPermit(s.User)
	if err != nil {
		return false, err
	}

	if permitted {
		return false, s.Replyf("Link permitted. (%s)", s.UserDisplay)
	}

	if err := s.DeleteMessage(); err != nil {
		return true, err
	}

	return true, s.Replyf("%s, please ask a moderator before posting links.", s.UserDisplay)
}

func allLinksPermitted(permitted []string, urls []*url.URL) bool {
	// Fast path for single links.
	if len(urls) == 1 {
		u := urls[0]

		for _, pd := range permitted {
			if linkmatch.HostAndPath(pd, u) {
				return true
			}
		}

		return false
	}

	urls = append(urls[:0:0], urls...)

	for _, pd := range permitted {
		allNil := true

		for i, u := range urls {
			if u == nil {
				continue
			}

			allNil = false

			if linkmatch.HostAndPath(pd, u) {
				urls[i] = nil
			}
		}

		if allNil {
			return true
		}
	}

	for _, u := range urls {
		if u != nil {
			return false
		}
	}

	return true
}

func filterCaps(ctx context.Context, s *session) (filtered bool, err error) {
	if !s.Channel.FilterCaps {
		return false, nil
	}

	message := s.Message

	if utf8.RuneCountInString(message) < s.Channel.FilterCapsMinChars {
		return false, nil
	}

	message = withoutSpaces(message)

	messageLen := 0
	count := 0

	for _, r := range message {
		messageLen++
		if unicode.IsUpper(r) {
			count++
		}
	}

	if count < s.Channel.FilterCapsMinCaps {
		return false, nil
	}

	percent := float64(count) / float64(messageLen)

	if int(percent*100) < s.Channel.FilterCapsPercentage {
		return false, nil
	}

	if err := s.DeleteMessage(); err != nil {
		return true, err
	}

	return true, s.Replyf("%s, please don't shout or talk in all caps.", s.UserDisplay)
}

var symbolFuncs = []func(rune) bool{
	unicode.IsControl,
	unicode.IsMark,
	unicode.IsPunct,
	unicode.IsSymbol,
}

func filterSymbols(ctx context.Context, s *session) (filtered bool, err error) {
	if !s.Channel.FilterSymbols {
		return false, nil
	}

	message := withoutSpaces(s.Message)

	messageLen := 0
	count := 0

	for _, r := range message {
		messageLen++

		for _, fn := range symbolFuncs {
			if fn(r) {
				count++
				break
			}
		}
	}

	if count < s.Channel.FilterSymbolsMinSymbols {
		return false, nil
	}

	percent := float64(count) / float64(messageLen)

	if int(percent*100) < s.Channel.FilterSymbolsPercentage {
		return false, nil
	}

	if err := s.DeleteMessage(); err != nil {
		return true, err
	}

	return true, s.Replyf("%s, please don't spam symbols.", s.UserDisplay)
}

func withoutSpaces(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}
