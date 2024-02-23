package bot

import (
	"fmt"
	"sync"
	"testing"
)

type testingHelper struct {
	mu      sync.Mutex
	userIDs map[string]int64
	names   map[int64]string
}

func (t *testingHelper) checkUserNameID(name string, id int64) {
	if !testing.Testing() {
		panic("checkUserNameID called outside of test")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.userIDs == nil {
		t.userIDs = make(map[string]int64)
	}

	if t.names == nil {
		t.names = make(map[int64]string)
	}

	if expectedID, ok := t.userIDs[name]; ok && id != expectedID {
		panic(fmt.Sprintf("%v previously had id %v, now %v", name, expectedID, id))
	}

	if expectedName, ok := t.names[id]; ok && name != expectedName {
		panic(fmt.Sprintf("%v previously had name %v, now %v", id, expectedName, name))
	}

	t.userIDs[name] = id
	t.names[id] = name
}
