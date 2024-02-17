package confconvert

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/hortbot/hortbot/internal/confimport/sitedb"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"github.com/zikaeroh/ctxlog"
)

var (
	siteChannels map[string]*sitedb.Channel
	siteVars     map[string][]*sitedb.Var
)

func loadSiteDB(ctx context.Context, dir string) {
	decodeInto(ctx, filepath.Join(dir, "site_channels.json"), &siteChannels)
	decodeInto(ctx, filepath.Join(dir, "site_vars.json"), &siteVars)
}

func decodeInto(ctx context.Context, file string, v any) {
	f, err := os.Open(file)
	if err != nil {
		ctxlog.Fatal(ctx, "error opening file", ctxlog.PlainError(err))
	}
	defer f.Close()

	if err := jsonx.DecodeSingle(f, v); err != nil {
		ctxlog.Fatal(ctx, "error decoding file", ctxlog.PlainError(err))
	}
}

func getSiteInfo(name string) (botName string, active bool, found bool) {
	ch, found := siteChannels[name]
	if !found {
		return "", false, false
	}

	return ch.BotName, ch.Active, true
}

func getVariables(channelName string) []*models.Variable {
	vars := siteVars[channelName]
	if len(vars) == 0 {
		return []*models.Variable{}
	}

	variables := make([]*models.Variable, len(vars))

	for i, v := range vars {
		variables[i] = &models.Variable{
			Name:      v.Name,
			Value:     v.Value,
			CreatedAt: v.LastModified,
			UpdatedAt: v.LastModified,
		}
	}

	sort.Slice(variables, func(i, j int) bool {
		return variables[i].Name < variables[j].Name
	})

	return variables
}
