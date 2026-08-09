package eventsubsync_test

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	status := 1
	defer func() {
		if r := recover(); r != nil {
			panic(r)
		}
		os.Exit(status)
	}()

	defer pool.Cleanup()
	status = m.Run()
}
