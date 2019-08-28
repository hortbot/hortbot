package bot

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hortbot/hortbot/internal/pkg/linkmatch"
	"github.com/opentracing/opentracing-go"
)

var filters = []func(context.Context, *session) (filtered bool, err error){
	filterMe,
	filterLength,
	filterLinks,
	filterEmotes,
	filterCaps,
	filterSymbols,
	filterBannedPhrases,
}

func tryFilter(ctx context.Context, s *session) (filtered bool, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "tryFilter")
	defer span.Finish()

	if !s.Channel.ShouldModerate || !s.Channel.EnableFilters {
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

func filterMe(ctx context.Context, s *session) (filtered bool, err error) {
	if !s.Channel.FilterMe || !s.Me {
		return false, nil
	}

	if s.UserLevel.CanAccess(levelSubscriber) {
		return false, nil
	}

	return true, filterDoPunish(ctx, s, "me", "/me is not allowed in this channel")
}

func filterLinks(ctx context.Context, s *session) (filtered bool, err error) {
	if !s.Channel.FilterLinks {
		return false, nil
	}

	minLevel := levelModerator
	if s.Channel.SubsMayLink {
		minLevel = levelSubscriber
	}

	if s.UserLevel.CanAccess(minLevel) {
		return false, nil
	}

	links := s.Links(ctx)

	if len(links) == 0 {
		return false, nil
	}

	if allLinksPermitted(s.Channel.PermittedLinks, links) {
		return false, nil
	}

	permitted, err := s.HasLinkPermit(ctx, s.User)
	if err != nil {
		return false, err
	}

	if permitted {
		return false, s.Replyf(ctx, "Link permitted. (%s)", s.UserDisplay)
	}

	return true, filterDoPunish(ctx, s, "links", "please ask a moderator before posting links")
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

	if s.UserLevel.CanAccess(levelSubscriber) {
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

	return true, filterDoPunish(ctx, s, "caps", "please don't shout or talk in all caps")
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

	if s.UserLevel.CanAccess(levelSubscriber) {
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

	return true, filterDoPunish(ctx, s, "symbols", "please don't spam symbols")
}

func filterLength(ctx context.Context, s *session) (filtered bool, err error) {
	if s.Channel.FilterMaxLength <= 0 {
		return false, nil
	}

	if s.UserLevel.CanAccess(levelSubscriber) {
		return false, nil
	}

	if utf8.RuneCountInString(s.Message) < s.Channel.FilterMaxLength {
		return false, nil
	}

	return true, filterDoPunish(ctx, s, "max_length", "please don't spam long messages")
}

func filterEmotes(ctx context.Context, s *session) (filtered bool, err error) {
	if !s.Channel.FilterEmotes {
		return false, nil
	}

	if s.UserLevel.CanAccess(levelSubscriber) {
		return false, nil
	}

	// TODO: BTTV/FFZ emotes.
	count := strings.Count(s.M.Tags["emotes"], "-")

	if count > s.Channel.FilterEmotesMax {
		return true, filterDoPunish(ctx, s, "emotes", "please don't spam emotes")
	}

	// If the count is 1 and the message has no whitespace, then it's a single emote message.
	if count == 1 && s.Channel.FilterEmotesSingle && !containsSpace(s.Message) {
		return true, filterDoPunish(ctx, s, "emotes", "single emote messages are not allowed")
	}

	return false, nil
}

func filterBannedPhrases(ctx context.Context, s *session) (filtered bool, err error) {
	if !s.Channel.FilterBannedPhrases {
		return false, nil
	}

	if s.UserLevel.CanAccess(levelSubscriber) {
		return false, nil
	}

	for _, pattern := range s.Channel.FilterBannedPhrasesPatterns {
		re, err := s.Deps.ReCache.Compile(pattern)
		if err != nil {
			continue
		}

		if re.MatchString(s.Message) {
			return true, filterDoPunish(ctx, s, "banned_phrase", "disallowed word or phrase")
		}
	}

	return false, nil
}

func withoutSpaces(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

func containsSpace(s string) bool {
	for _, r := range s {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func filterDoPunish(ctx context.Context, s *session, filter, message string) error {
	if s.Channel.EnableWarnings {
		warned, err := s.FilterWarned(ctx, s.User, filter)
		if err != nil {
			return err
		}

		if !warned {
			if err := s.DeleteMessage(ctx); err != nil {
				return err
			}

			if s.Channel.DisplayWarnings {
				return s.Replyf(ctx, "%s, %s - warning", s.UserDisplay, message)
			}

			return nil
		}
	}

	var err error

	if s.Channel.TimeoutDuration == 0 {
		err = s.SendCommand(ctx, "timeout", s.User)
	} else {
		err = s.SendCommand(ctx, "timeout", s.User, strconv.Itoa(s.Channel.TimeoutDuration))
	}

	if err != nil {
		return err
	}

	if s.Channel.DisplayWarnings {
		return s.Replyf(ctx, "%s, %s - timeout", s.UserDisplay, message)
	}

	return nil
}
