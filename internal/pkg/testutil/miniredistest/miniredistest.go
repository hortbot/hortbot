// Package miniredistest provides a test redis server.
package miniredistest

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// New creates a new miniredis server and returns a pre-prepared client.
func New() (s *miniredis.Miniredis, c *redis.Client, cleanup func(), retErr error) {
	s, err := miniredis.Run()
	if err != nil {
		return nil, nil, func() {}, err
	}

	defer func() {
		if retErr != nil {
			s.Close()
		}
	}()

	c = redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	return s, c, func() {
		defer s.Close()
		defer c.Close()
	}, nil
}
