package confimport

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/db/migrations"
	"github.com/hortbot/hortbot/internal/pkg/testpostgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"gotest.tools/v3/assert"
)

func TestExportImport(t *testing.T) {
	t.Parallel()
	pdb, err := testpostgres.New()
	assert.NilError(t, err)
	defer pdb.Cleanup()
	assert.NilError(t, migrations.Up(pdb.ConnStr(), nil))

	db, err := pdb.Open(t.Context())
	assert.NilError(t, err)
	defer db.Close()

	ctx := t.Context()
	queries := dbsql.New(db)
	_, err = queries.InsertDefaultChannel(ctx, dbsql.InsertDefaultChannelParams{
		TwitchID: 1, Name: "source", DisplayName: "Source", BotName: "hortbot",
	})
	assert.NilError(t, err)

	config, err := ExportByName(ctx, queries, "source")
	assert.NilError(t, err)
	config.Channel.TwitchID = 2
	config.Channel.Name = "imported"
	config.Channel.DisplayName = "Imported"
	assert.NilError(t, config.Insert(ctx, queries))

	imported, err := queries.GetChannelByName(ctx, "imported")
	assert.NilError(t, err)
	assert.Equal(t, imported.TwitchID, int64(2))
	assert.Equal(t, imported.DisplayName, "Imported")
	assert.Equal(t, imported.Prefix, config.Channel.Prefix)
	assert.Equal(t, imported.CreatedAt, config.Channel.CreatedAt)
	assert.Equal(t, imported.UpdatedAt, config.Channel.UpdatedAt)

	legacyConfig, err := ExportByName(ctx, queries, "source")
	assert.NilError(t, err)
	data, err := json.Marshal(legacyConfig)
	assert.NilError(t, err)

	var document map[string]any
	assert.NilError(t, json.Unmarshal(data, &document))
	channel := document["channel"].(map[string]any)
	delete(channel, "last_seen")
	delete(channel, "filter_exempt_level")
	channel["ignored"] = nil
	channel["custom_owners"] = nil
	channel["custom_mods"] = nil
	channel["custom_regulars"] = nil
	channel["permitted_links"] = nil
	channel["filter_banned_phrases_patterns"] = nil

	data, err = json.Marshal(document)
	assert.NilError(t, err)
	var legacy Config
	assert.NilError(t, json.Unmarshal(data, &legacy))
	legacy.Channel.TwitchID = 3
	legacy.Channel.Name = "legacy"
	legacy.Channel.DisplayName = "Legacy"
	legacy.Commands = []*Command{{
		Info: &CommandInfo{
			Name:        "legacy-list",
			AccessLevel: dbsql.AccessLevelEveryone,
			Creator:     "test",
			Editor:      "test",
		},
		CommandList: &CommandList{},
		Repeat: &RepeatedCommand{
			Enabled: true,
			Delay:   60,
			Creator: "test",
			Editor:  "test",
		},
	}, {
		Info: &CommandInfo{
			Name:        "legacy-command",
			AccessLevel: dbsql.AccessLevelModerator,
			Creator:     "test",
			Editor:      "test",
		},
		CustomCommand: &CustomCommand{
			Message: "hello",
		},
		Schedule: &ScheduledCommand{
			Enabled:        true,
			CronExpression: "0 * * * *",
			Creator:        "test",
			Editor:         "test",
		},
	}}
	legacy.Quotes = []*Quote{{
		Num:     1,
		Quote:   "legacy quote",
		Creator: "test",
		Editor:  "test",
	}}
	legacy.Autoreplies = []*Autoreply{{
		Num:      1,
		Trigger:  "legacy",
		Response: "reply",
		Creator:  "test",
		Editor:   "test",
	}}
	legacy.Variables = []*Variable{{
		Name:  "legacy",
		Value: "value",
	}}
	assert.NilError(t, legacy.Insert(ctx, queries))

	importedLegacy, err := queries.GetChannelByName(ctx, "legacy")
	assert.NilError(t, err)
	assert.Equal(t, importedLegacy.FilterExemptLevel, dbsql.AccessLevelSubscriber)
	assert.Assert(t, importedLegacy.LastSeen.Valid)
	assert.Assert(t, !importedLegacy.LastSeen.Time.IsZero())
	assert.DeepEqual(t, importedLegacy.Ignored, []string{})
	assert.DeepEqual(t, importedLegacy.CustomOwners, []string{})
	assert.DeepEqual(t, importedLegacy.CustomMods, []string{})
	assert.DeepEqual(t, importedLegacy.CustomRegulars, []string{})
	assert.DeepEqual(t, importedLegacy.PermittedLinks, []string{})
	assert.DeepEqual(t, importedLegacy.FilterBannedPhrasesPatterns, []string{})

	info, err := queries.GetCommandInfo(ctx, dbsql.GetCommandInfoParams{
		ChannelID: importedLegacy.ID,
		Name:      "legacy-list",
	})
	assert.NilError(t, err)
	list, err := queries.GetCommandList(ctx, info.CommandListID.Int64)
	assert.NilError(t, err)
	assert.DeepEqual(t, list.Items, []string{})
	assert.Assert(t, list.CreatedAt.Valid)
	assert.Assert(t, list.UpdatedAt.Valid)
	repeated, err := queries.GetRepeatedCommandByInfo(ctx, info.ID)
	assert.NilError(t, err)
	assert.Equal(t, repeated.MessageDiff, int64(1))
	assert.Assert(t, repeated.CreatedAt.Valid)
	assert.Assert(t, repeated.UpdatedAt.Valid)

	exportedLegacy, err := ExportByName(ctx, queries, "legacy")
	assert.NilError(t, err)
	assert.Equal(t, len(exportedLegacy.Commands), 2)
	assert.Equal(t, len(exportedLegacy.Quotes), 1)
	assert.Equal(t, len(exportedLegacy.Autoreplies), 1)
	assert.Equal(t, len(exportedLegacy.Variables), 1)

	data, err = json.Marshal(exportedLegacy)
	assert.NilError(t, err)
	assert.NilError(t, json.Unmarshal(data, &document))
	removeTableDefaults(t, ctx, db, "channels", document["channel"].(map[string]any))
	delete(document["channel"].(map[string]any), "filter_exempt_level")
	for _, value := range document["quotes"].([]any) {
		removeTableDefaults(t, ctx, db, "quotes", value.(map[string]any))
	}
	for _, value := range document["commands"].([]any) {
		command := value.(map[string]any)
		removeTableDefaults(t, ctx, db, "command_infos", command["info"].(map[string]any))
		if value := command["custom_command"]; value != nil {
			removeTableDefaults(t, ctx, db, "custom_commands", value.(map[string]any))
		}
		if value := command["command_list"]; value != nil {
			removeTableDefaults(t, ctx, db, "command_lists", value.(map[string]any))
		}
		if value := command["repeat"]; value != nil {
			removeTableDefaults(t, ctx, db, "repeated_commands", value.(map[string]any))
		}
		if value := command["schedule"]; value != nil {
			removeTableDefaults(t, ctx, db, "scheduled_commands", value.(map[string]any))
		}
	}
	for _, value := range document["autoreplies"].([]any) {
		removeTableDefaults(t, ctx, db, "autoreplies", value.(map[string]any))
	}
	for _, value := range document["variables"].([]any) {
		removeTableDefaults(t, ctx, db, "variables", value.(map[string]any))
	}

	data, err = json.Marshal(document)
	assert.NilError(t, err)
	var roundtripConfig Config
	assert.NilError(t, json.Unmarshal(data, &roundtripConfig))
	roundtripConfig.Channel.TwitchID = 4
	roundtripConfig.Channel.Name = "roundtrip"
	roundtripConfig.Channel.DisplayName = "Roundtrip"
	assert.NilError(t, roundtripConfig.Insert(ctx, queries))

	roundtrip, err := ExportByName(ctx, queries, "roundtrip")
	assert.NilError(t, err)
	assert.Equal(t, len(roundtrip.Commands), 2)
	assert.Equal(t, len(roundtrip.Quotes), 1)
	assert.Equal(t, roundtrip.Quotes[0].Quote, "legacy quote")
	assert.Equal(t, len(roundtrip.Autoreplies), 1)
	assert.Equal(t, roundtrip.Autoreplies[0].Response, "reply")
	assert.Equal(t, len(roundtrip.Variables), 1)
	assert.Equal(t, roundtrip.Variables[0].Value, "value")
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
		err    string
	}{
		{name: "channel", config: Config{}, err: "config has no channel"},
		{name: "quote", config: Config{Channel: &Channel{}, Quotes: []*Quote{nil}}, err: "quote 0 is null"},
		{name: "command", config: Config{Channel: &Channel{}, Commands: []*Command{nil}}, err: "command 0 is null"},
		{name: "command info", config: Config{Channel: &Channel{}, Commands: []*Command{{}}}, err: "command 0 has no info"},
		{
			name: "command implementation",
			config: Config{
				Channel:  &Channel{},
				Commands: []*Command{{Info: &CommandInfo{}}},
			},
			err: "command 0 must have exactly one implementation",
		},
		{
			name: "two command implementations",
			config: Config{
				Channel: &Channel{},
				Commands: []*Command{{
					Info:          &CommandInfo{},
					CustomCommand: &CustomCommand{},
					CommandList:   &CommandList{},
				}},
			},
			err: "command 0 must have exactly one implementation",
		},
		{name: "autoreply", config: Config{Channel: &Channel{}, Autoreplies: []*Autoreply{nil}}, err: "autoreply 0 is null"},
		{name: "variable", config: Config{Channel: &Channel{}, Variables: []*Variable{nil}}, err: "variable 0 is null"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorContains(t, test.config.validate(), test.err)
		})
	}
}

func removeTableDefaults(t *testing.T, ctx context.Context, db *pgxpool.Pool, table string, object map[string]any) {
	t.Helper()

	rows, err := db.Query(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = $1
		  AND (column_default IS NOT NULL OR is_identity = 'YES')
	`, table)
	assert.NilError(t, err)
	defer rows.Close()

	for rows.Next() {
		var column string
		assert.NilError(t, rows.Scan(&column))
		delete(object, column)
	}
	assert.NilError(t, rows.Err())
}
