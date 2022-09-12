package bot_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/hortbot/hortbot/internal/bot/btest"
	"gotest.tools/v3/assert"
)

var scriptTest = flag.String("script_test", "", "Specific script test to run")

func TestScripts(t *testing.T) {
	t.Parallel()

	scriptDir := filepath.Join("testdata", "script")

	files, err := doublestar.Glob(os.DirFS(scriptDir), "**/*.txt")
	assert.NilError(t, err)
	assert.Assert(t, len(files) != 0)

	run := false

	for _, file := range files {
		name := strings.TrimSuffix(file, ".txt")
		file := filepath.Join(scriptDir, filepath.FromSlash(file))
		absFile, err := filepath.Abs(file)
		assert.NilError(t, err)

		if *scriptTest != "" {
			if *scriptTest != absFile {
				continue
			}
		}

		run = true
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			btest.RunScript(t, absFile, pool.FreshDB)
		})
	}

	if !run {
		t.Error("No tests were run.")
	}
}
