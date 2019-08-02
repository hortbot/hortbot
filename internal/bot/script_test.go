package bot_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmatcuk/doublestar"
	"github.com/hortbot/hortbot/internal/bot/btest"
	"gotest.tools/assert"
)

func TestScripts(t *testing.T) {
	files, err := doublestar.Glob(filepath.Join("testdata", "script", "**", "*.txt"))
	assert.NilError(t, err)
	assert.Assert(t, len(files) != 0)

	prefix := filepath.Join("testdata", "script")

	for _, file := range files {
		file := file
		name := strings.TrimSuffix(strings.TrimPrefix(file, prefix)[1:], ".txt")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			btest.RunScript(t, file, freshDB)
		})
	}
}
