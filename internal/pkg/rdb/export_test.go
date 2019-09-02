package rdb

import "github.com/go-redis/redis/v7"

func ReplaceCheckAndMark(source string) func() {
	old := checkAndMark
	checkAndMark = redis.NewScript(source)
	return func() {
		checkAndMark = old
	}
}

func ReplaceCheckAndRefresh(source string) func() {
	old := checkAndRefresh
	checkAndRefresh = redis.NewScript(source)
	return func() {
		checkAndRefresh = old
	}
}

func ReplaceMarkOrDelete(source string) func() {
	old := markOrDelete
	markOrDelete = redis.NewScript(source)
	return func() {
		markOrDelete = old
	}
}

func ReplaceRateLimit(source string) func() {
	old := rateLimit
	rateLimit = redis.NewScript(source)
	return func() {
		rateLimit = old
	}
}
