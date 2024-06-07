package useragent_test

import (
	"strings"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/useragent"
	"gotest.tools/v3/assert"
)

func TestBot(t *testing.T) {
	t.Parallel()
	agent := useragent.Bot()
	assert.Assert(t, strings.HasPrefix(agent, "HortBot"))
}

func TestBrowser(t *testing.T) {
	t.Parallel()
	agent := useragent.Browser()
	assert.Assert(t, strings.HasPrefix(agent, "Mozilla"))
}
