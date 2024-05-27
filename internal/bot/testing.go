package bot

import (
	"context"
	"sync"
	"testing"

	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

type testingHelper struct {
	mu      sync.Mutex
	userIDs map[string]int64
	names   map[int64]string
}

func (t *testingHelper) checkUserNameID(ctx context.Context, name string, id int64) {
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
		ctxlog.Warn(ctx, "user ID changed", zap.String("name", name), zap.Int64("expected", expectedID), zap.Int64("actual", id))
	}

	if expectedName, ok := t.names[id]; ok && name != expectedName {
		ctxlog.Warn(ctx, "user name changed", zap.Int64("id", id), zap.String("expected", expectedName), zap.String("actual", name))
	}

	t.userIDs[name] = id
	t.names[id] = name
}
