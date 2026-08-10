package botstate

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/hortbot/hortbot/internal/db/dbsql"
)

// IncrementBuiltinUsageStat atomically adds one to the named builtin
// usage counter, creating it if needed.
func (*Store) IncrementBuiltinUsageStat(ctx context.Context, queries *dbsql.Queries, name string) error {
	return addBuiltinUsageStat(ctx, queries, name, 1)
}

func addBuiltinUsageStat(ctx context.Context, queries *dbsql.Queries, name string, count int64) error {
	err := queries.BotStateAddBuiltinUsageStat(ctx, dbsql.BotStateAddBuiltinUsageStatParams{
		Name:  name,
		Count: count,
	})
	if err != nil {
		return fmt.Errorf("increment builtin usage stat: %w", err)
	}
	return nil
}

// GetBuiltinUsageStats returns all builtin usage counters.
func (*Store) GetBuiltinUsageStats(ctx context.Context, queries *dbsql.Queries) (map[string]string, error) {
	rows, err := queries.BotStateListBuiltinUsageStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get builtin usage stats: %w", err)
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.Name] = strconv.FormatInt(row.Count, 10)
	}
	return out, nil
}

// IncrementActionUsageStat atomically adds one to the named action
// usage counter.
func (*Store) IncrementActionUsageStat(ctx context.Context, queries *dbsql.Queries, name string) error {
	return addActionUsageStat(ctx, queries, name, 1)
}

func addActionUsageStat(ctx context.Context, queries *dbsql.Queries, name string, count int64) error {
	err := queries.BotStateAddActionUsageStat(ctx, dbsql.BotStateAddActionUsageStatParams{
		Name:  name,
		Count: count,
	})
	if err != nil {
		return fmt.Errorf("increment action usage stat: %w", err)
	}
	return nil
}

// MergeUsageStats imports usage counters without reducing any existing count.
func (*Store) MergeUsageStats(
	ctx context.Context,
	queries *dbsql.Queries,
	builtins map[string]int64,
	actions map[string]int64,
) error {
	for _, name := range slices.Sorted(maps.Keys(builtins)) {
		if err := queries.BotStateMergeBuiltinUsageStat(ctx, dbsql.BotStateMergeBuiltinUsageStatParams{
			Name:  name,
			Count: builtins[name],
		}); err != nil {
			return fmt.Errorf("merge builtin usage stat: %w", err)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(actions)) {
		if err := queries.BotStateMergeActionUsageStat(ctx, dbsql.BotStateMergeActionUsageStatParams{
			Name:  name,
			Count: actions[name],
		}); err != nil {
			return fmt.Errorf("merge action usage stat: %w", err)
		}
	}
	return nil
}

// AddUsageStats adds a message's accumulated usage counters.
func (*Store) AddUsageStats(
	ctx context.Context,
	queries *dbsql.Queries,
	builtins map[string]int64,
	actions map[string]int64,
) error {
	for _, name := range slices.Sorted(maps.Keys(builtins)) {
		count := builtins[name]
		if err := addBuiltinUsageStat(ctx, queries, name, count); err != nil {
			return err
		}
	}
	for _, name := range slices.Sorted(maps.Keys(actions)) {
		count := actions[name]
		if err := addActionUsageStat(ctx, queries, name, count); err != nil {
			return err
		}
	}
	return nil
}

// GetActionUsageStats returns all action usage counters.
func (*Store) GetActionUsageStats(ctx context.Context, queries *dbsql.Queries) (map[string]string, error) {
	rows, err := queries.BotStateListActionUsageStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get action usage stats: %w", err)
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.Name] = strconv.FormatInt(row.Count, 10)
	}
	return out, nil
}
