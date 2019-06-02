package redis

import "github.com/go-redis/redis"

func ReplaceCheck(source string) func() {
	old := check
	check = redis.NewScript(source)
	return func() {
		check = old
	}
}

func ReplaceCheckAndMark(source string) func() {
	old := checkAndMark
	checkAndMark = redis.NewScript(source)
	return func() {
		checkAndMark = old
	}
}
