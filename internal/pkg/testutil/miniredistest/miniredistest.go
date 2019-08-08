package miniredistest

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v7"
)

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
