package bnsq

import "time"

func TestingSleep(d time.Duration) {
	testingSleep.Store(d)
}
