// Package linkmatch implements an algorithm for matching URLs against patterns.
package linkmatch

import (
	"net/url"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/hortbot/hortbot/internal/pkg/stringsx"
)

// HostAndPath checks that the given URL matches the given pattern.
//
// The host and path are treated differently. For hosts, following rules
// are applied:
//
//   - If the host is the same as the pattern, then it's a match.
//   - If the host has the suffix ".<pattern>", then it's a match.
//
// For example, "twitch.tv" would match "twitch.tv", "www.twitch.tv", and
// "clips.twitch.tv". In the future, globs may be accepted here.
//
// Paths are matched "permissibly" by doublestar globs. Given the pattern
// "/foo", the path must match one of the patterns "/foo" or "/foo/**". Given
// the pattern "/foo*", the patch must match one of the patterns "/foo*",
// "/foo**", or "/foo*/**".
func HostAndPath(pattern string, u *url.URL) bool {
	hostPattern, pathPattern := splitPattern(pattern)

	if !hostMatches(hostPattern, u.Host) {
		return false
	}

	pathPattern = normalizePath(pathPattern)

	if pathPattern == "/" {
		return true
	}

	path := normalizePath(u.Path)

	return isMatchPermissive(pathPattern, path)
}

func hostMatches(pattern, host string) bool {
	host = strings.ToLower(host)
	pattern = strings.ToLower(pattern)

	if host == pattern {
		return true
	}

	if len(host) <= len(pattern) {
		return false
	}

	if !strings.HasSuffix(host, pattern) {
		return false
	}

	idx := len(host) - len(pattern) - 1
	return host[idx] == '.'
}

func isMatchPermissive(pattern, s string) bool {
	if isMatch(pattern, s) {
		return true
	}

	if strings.HasSuffix(pattern, "*") && isMatch(pattern+"*", s) {
		return true
	}

	if isMatch(pattern+"/**", s) {
		return true
	}

	return false
}

func isMatch(pattern, s string) bool {
	match, _ := doublestar.Match(pattern, s)
	return match
}

func normalizePath(p string) string {
	if p == "" || p == "/" {
		return "/"
	}

	if p[0] != '/' {
		p = "/" + p
	}

	p = strings.ToLower(p)
	return strings.TrimRight(p, "/")
}

func splitPattern(pattern string) (host, path string) {
	scheme, rest := stringsx.Split(pattern, "://")
	if rest == "" && scheme != "" {
		rest = scheme
	}

	return stringsx.SplitByte(rest, '/')
}

// IsBadPattern returns true if the pattern is too permissive.
func IsBadPattern(pattern string) bool {
	host, path := splitPattern(pattern)
	host = strings.Trim(host, "*")
	path = strings.Trim(path, "*")
	return host == "" && path == ""
}
