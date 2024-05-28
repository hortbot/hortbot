// Package recache provides a regex compilation cache.
package recache

import (
	"regexp"

	lru "github.com/hashicorp/golang-lru/v2"
)

type value struct {
	re  *regexp.Regexp
	err error
}

// RegexpCache caches compiled regular expressions.
type RegexpCache struct {
	c *lru.TwoQueueCache[string, value]
}

// New creates a new RegexpCache.
func New() *RegexpCache {
	c, err := lru.New2Q[string, value](1000)
	if err != nil {
		panic(err)
	}

	return &RegexpCache{
		c: c,
	}
}

// Compile compiles a pattern, forcing case-insensitive matching. If this
// pattern is cached, the existing Regexp/error will be returned.
func (r *RegexpCache) Compile(pattern string) (*regexp.Regexp, error) {
	got, ok := r.c.Get(pattern)
	if ok {
		return got.re, got.err
	}

	re, err := regexp.Compile(`(?i)` + pattern)

	r.c.Add(pattern, value{
		re:  re,
		err: err,
	})

	return re, err //nolint:wrapcheck
}
