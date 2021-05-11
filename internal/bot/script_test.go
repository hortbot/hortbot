package bot_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/hortbot/hortbot/internal/bot/btest"
	"gotest.tools/v3/assert"
)

func TestScripts(t *testing.T) {
	t.Parallel()

	scriptDir := filepath.Join("testdata", "script")

	files, err := doublestar.Glob(os.DirFS(scriptDir), "**/*.txt")
	assert.NilError(t, err)
	assert.Assert(t, len(files) != 0)

	for _, file := range files {
		name := strings.TrimSuffix(file, ".txt")
		file := filepath.Join(scriptDir, filepath.FromSlash(file))
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			btest.RunScript(t, file, pool.FreshDB)
		})
	}
}
