// Package recache provides a regex compilation cache.
package recache

import (
	"regexp"
	"time"

	"github.com/patrickmn/go-cache"
)

// Expiration intervals.
const (
	DefaultExpiration      = 5 * time.Minute
	DefaultCleanupInterval = 10 * time.Minute
)

// RegexpCache caches compiled regular expressions.
type RegexpCache struct {
	c *cache.Cache
}

// New creates a new RegexpCache.
func New() *RegexpCache {
	return &RegexpCache{
		c: cache.New(DefaultExpiration, DefaultCleanupInterval),
	}
}

// Compile compiles a pattern, forcing case-insensitive matching. If this
// pattern is cached, the existing Regexp/error will be returned.
func (r *RegexpCache) Compile(pattern string) (*regexp.Regexp, error) {
	type value struct {
		re  *regexp.Regexp
		err error
	}

	got, ok := r.c.Get(pattern)
	if ok {
		v := got.(*value)
		return v.re, v.err
	}

	re, err := regexp.Compile(`(?i)` + pattern)

	v := &value{
		re:  re,
		err: err,
	}

	r.c.SetDefault(pattern, v)

	return re, err
}
