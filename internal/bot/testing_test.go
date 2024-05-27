package bot

import (
	"context"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gotest.tools/v3/assert"
)

func TestTestingHelper(t *testing.T) {
	t.Parallel()

	logs := []string{}

	logger := testutil.Logger(t).WithOptions(zap.Hooks(func(e zapcore.Entry) error {
		logs = append(logs, e.Message)
		return nil
	}))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	helper := &testingHelper{}

	// Normally fails, but if nil should be ignored.
	helper.checkUserNameID(ctx, "foo", 1)
	assert.DeepEqual(t, logs, []string{})

	helper.checkUserNameID(ctx, "foo", 1)
	assert.DeepEqual(t, logs, []string{})

	helper.checkUserNameID(ctx, "foo", 2)
	assert.DeepEqual(t, logs, []string{"user ID changed"})
	logs = logs[:0]

	helper.checkUserNameID(ctx, "bar", 1)
	assert.DeepEqual(t, logs, []string{"user name changed"})
}
