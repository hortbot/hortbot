package bot_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmatcuk/doublestar/v2"
	"github.com/hortbot/hortbot/internal/bot/btest"
	"gotest.tools/v3/assert"
)

func TestScripts(t *testing.T) {
	t.Parallel()

	files, err := doublestar.Glob(filepath.Join("testdata", "script", "**", "*.txt"))
	assert.NilError(t, err)
	assert.Assert(t, len(files) != 0)

	prefix := filepath.Join("testdata", "script")

	for _, file := range files {
		file := file
		name := strings.TrimSuffix(strings.TrimPrefix(file, prefix)[1:], ".txt")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			btest.RunScript(t, file, pool.FreshDB)
		})
	}
}
