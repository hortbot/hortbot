package botstate

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/aarondl/sqlboiler/v4/boil"
)

// IncrementBuiltinUsageStat atomically adds one to the named builtin
// usage counter, creating it if needed.
func (*Store) IncrementBuiltinUsageStat(ctx context.Context, exec boil.ContextExecutor, name string) error {
	return addBuiltinUsageStat(ctx, exec, name, 1)
}

func addBuiltinUsageStat(ctx context.Context, exec boil.ContextExecutor, name string, count int64) error {
	_, err := exec.ExecContext(ctx, `
		INSERT INTO bot_builtin_usage_stats (name, count) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET count = bot_builtin_usage_stats.count + EXCLUDED.count
	`, name, count)
	if err != nil {
		return fmt.Errorf("increment builtin usage stat: %w", err)
	}
	return nil
}

// GetBuiltinUsageStats returns all builtin usage counters.
func (*Store) GetBuiltinUsageStats(ctx context.Context, exec boil.ContextExecutor) (map[string]string, error) {
	return getUsageStats(ctx, exec, `SELECT name, count FROM bot_builtin_usage_stats`, "builtin")
}

// IncrementActionUsageStat atomically adds one to the named action
// usage counter.
func (*Store) IncrementActionUsageStat(ctx context.Context, exec boil.ContextExecutor, name string) error {
	return addActionUsageStat(ctx, exec, name, 1)
}

func addActionUsageStat(ctx context.Context, exec boil.ContextExecutor, name string, count int64) error {
	_, err := exec.ExecContext(ctx, `
		INSERT INTO bot_action_usage_stats (name, count) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET count = bot_action_usage_stats.count + EXCLUDED.count
	`, name, count)
	if err != nil {
		return fmt.Errorf("increment action usage stat: %w", err)
	}
	return nil
}

// MergeUsageStats imports usage counters without reducing any existing count.
func (*Store) MergeUsageStats(
	ctx context.Context,
	exec boil.ContextExecutor,
	builtins map[string]int64,
	actions map[string]int64,
) error {
	for _, name := range slices.Sorted(maps.Keys(builtins)) {
		if _, err := exec.ExecContext(ctx, `
			INSERT INTO bot_builtin_usage_stats (name, count) VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE
				SET count = GREATEST(bot_builtin_usage_stats.count, EXCLUDED.count)
		`, name, builtins[name]); err != nil {
			return fmt.Errorf("merge builtin usage stat: %w", err)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(actions)) {
		if _, err := exec.ExecContext(ctx, `
			INSERT INTO bot_action_usage_stats (name, count) VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE
				SET count = GREATEST(bot_action_usage_stats.count, EXCLUDED.count)
		`, name, actions[name]); err != nil {
			return fmt.Errorf("merge action usage stat: %w", err)
		}
	}
	return nil
}

// AddUsageStats adds a message's accumulated usage counters.
func (*Store) AddUsageStats(
	ctx context.Context,
	exec boil.ContextExecutor,
	builtins map[string]int64,
	actions map[string]int64,
) error {
	for _, name := range slices.Sorted(maps.Keys(builtins)) {
		count := builtins[name]
		if err := addBuiltinUsageStat(ctx, exec, name, count); err != nil {
			return err
		}
	}
	for _, name := range slices.Sorted(maps.Keys(actions)) {
		count := actions[name]
		if err := addActionUsageStat(ctx, exec, name, count); err != nil {
			return err
		}
	}
	return nil
}

// GetActionUsageStats returns all action usage counters.
func (*Store) GetActionUsageStats(ctx context.Context, exec boil.ContextExecutor) (map[string]string, error) {
	return getUsageStats(ctx, exec, `SELECT name, count FROM bot_action_usage_stats`, "action")
}

func getUsageStats(ctx context.Context, exec boil.ContextExecutor, query, kind string) (map[string]string, error) {
	rows, err := exec.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get %s usage stats: %w", kind, err)
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var name string
		var count int64
		if err := rows.Scan(&name, &count); err != nil {
			return nil, fmt.Errorf("scan %s usage stat: %w", kind, err)
		}
		out[name] = strconv.FormatInt(count, 10)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s usage stat rows: %w", kind, err)
	}
	return out, nil
}
