package dbsql_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"gotest.tools/v3/assert"
)

func TestDeleteStaleModeratedChannels(t *testing.T) {
	t.Parallel()

	pdb, err := testpostgres.New()
	assert.NilError(t, err)
	t.Cleanup(pdb.Cleanup)
	assert.NilError(t, migrations.Up(pdb.ConnStr(), nil))

	db, err := pdb.Open(t.Context())
	assert.NilError(t, err)
	t.Cleanup(db.Close)

	ctx := t.Context()
	queries := dbsql.New(db)
	cutoff := time.Now()

	assert.NilError(t, queries.UpsertModeratedChannel(ctx, dbsql.UpsertModeratedChannelParams{
		BotName:          "hortbot",
		BroadcasterID:    1,
		BroadcasterLogin: "current",
		BroadcasterName:  "Current",
		UpdatedAt:        dbsql.TimestamptzFrom(cutoff),
	}))
	assert.NilError(t, queries.UpsertModeratedChannel(ctx, dbsql.UpsertModeratedChannelParams{
		BotName:          "hortbot",
		BroadcasterID:    2,
		BroadcasterLogin: "stale",
		BroadcasterName:  "Stale",
		UpdatedAt:        dbsql.TimestamptzFrom(cutoff.Add(-time.Second)),
	}))
	assert.NilError(t, queries.DeleteStaleModeratedChannels(ctx, dbsql.DeleteStaleModeratedChannelsParams{
		BotName:       "hortbot",
		UpdatedBefore: dbsql.TimestamptzFrom(cutoff),
	}))

	current, err := queries.IsModeratedChannel(ctx, dbsql.IsModeratedChannelParams{
		BroadcasterID: 1,
		BotName:       "hortbot",
	})
	assert.NilError(t, err)
	assert.Assert(t, current)

	stale, err := queries.IsModeratedChannel(ctx, dbsql.IsModeratedChannelParams{
		BroadcasterID: 2,
		BotName:       "hortbot",
	})
	assert.NilError(t, err)
	assert.Assert(t, !stale)

	channel, err := queries.InsertDefaultChannel(ctx, dbsql.InsertDefaultChannelParams{
		TwitchID:    1,
		Name:        "timestamp-test",
		DisplayName: "Timestamp Test",
		BotName:     "hortbot",
	})
	assert.NilError(t, err)
	assert.NilError(t, queries.UpsertVariable(ctx, dbsql.UpsertVariableParams{
		ChannelID: channel.ID,
		Name:      "timestamp-test",
		Value:     "before",
	}))
	variable, err := queries.GetVariable(ctx, dbsql.GetVariableParams{
		ChannelID: channel.ID,
		Name:      "timestamp-test",
	})
	assert.NilError(t, err)

	tx, err := db.Begin(ctx)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	var transactionStart time.Time
	assert.NilError(t, tx.QueryRow(ctx, "SELECT transaction_timestamp()").Scan(&transactionStart))
	time.Sleep(10 * time.Millisecond)

	txQueries := dbsql.New(tx)
	assert.NilError(t, txQueries.UpdateVariableValue(ctx, dbsql.UpdateVariableValueParams{
		Value: "after",
		ID:    variable.ID,
	}))
	variable, err = txQueries.GetVariable(ctx, dbsql.GetVariableParams{
		ChannelID: channel.ID,
		Name:      "timestamp-test",
	})
	assert.NilError(t, err)
	assert.Assert(t, variable.UpdatedAt.Time.After(transactionStart))
}

func TestQueryUpdatedAtPolicy(t *testing.T) {
	t.Parallel()

	updatedTables := map[string]bool{
		"autoreplies":        true,
		"channels":           true,
		"command_infos":      true,
		"command_lists":      true,
		"custom_commands":    true,
		"moderated_channels": true,
		"quotes":             true,
		"repeated_commands":  true,
		"scheduled_commands": true,
		"twitch_tokens":      true,
		"variables":          true,
	}
	doNotTouch := map[string]bool{
		"CompactAutoreplies":              true,
		"CompactQuotes":                   true,
		"UpdateAutoreplyCount":            true,
		"UpdateChannelActivity":           true,
		"UpdateCommandInfoCount":          true,
		"UpdateCommandInfoUsage":          true,
		"UpdateRepeatedCommandLastCount":  true,
		"UpdateScheduledCommandLastCount": true,
	}

	_, filename, _, ok := runtime.Caller(0)
	assert.Assert(t, ok)
	files, err := filepath.Glob(filepath.Join(filepath.Dir(filename), "..", "queries", "*.sql"))
	assert.NilError(t, err)

	headerPattern := regexp.MustCompile(`(?m)^-- name: ([A-Za-z0-9_]+) :[a-z]+\n`)
	updatePattern := regexp.MustCompile(`(?i)\bUPDATE\s+([a-z_]+)`)
	insertPattern := regexp.MustCompile(`(?i)\bINSERT\s+INTO\s+([a-z_]+)`)
	touchPattern := regexp.MustCompile(`(?i)\bupdated_at\s*=`)
	transactionTimePattern := regexp.MustCompile(`(?i)\bupdated_at\s*=\s*NOW\(\)`)
	forUpdatePattern := regexp.MustCompile(`(?i)\bFOR\s+UPDATE\b`)
	foundExceptions := make(map[string]bool, len(doNotTouch))

	for _, file := range files {
		data, err := os.ReadFile(file)
		assert.NilError(t, err)
		headers := headerPattern.FindAllSubmatchIndex(data, -1)
		for i, header := range headers {
			end := len(data)
			if i+1 < len(headers) {
				end = headers[i+1][0]
			}
			name := string(data[header[2]:header[3]])
			query := string(data[header[1]:end])
			if strings.Contains(name, "ForUpdate") {
				assert.Assert(t, forUpdatePattern.MatchString(query), "%s does not lock its selected rows", name)
			}

			table := ""
			if match := updatePattern.FindStringSubmatch(query); match != nil {
				table = strings.ToLower(match[1])
			} else if strings.Contains(strings.ToUpper(query), "ON CONFLICT") {
				if match := insertPattern.FindStringSubmatch(query); match != nil {
					table = strings.ToLower(match[1])
				}
			}
			if !updatedTables[table] {
				continue
			}

			touchesUpdatedAt := touchPattern.MatchString(query)
			if doNotTouch[name] {
				foundExceptions[name] = true
				assert.Assert(t, !touchesUpdatedAt, "%s unexpectedly changes updated_at", name)
				continue
			}
			assert.Assert(t, touchesUpdatedAt, "%s mutates %s without changing updated_at", name, table)
			assert.Assert(t, !transactionTimePattern.MatchString(query), "%s uses transaction-start NOW() for updated_at", name)
		}
	}

	for name := range doNotTouch {
		assert.Assert(t, foundExceptions[name], "updated_at policy exception %s does not match a query", name)
	}
}
