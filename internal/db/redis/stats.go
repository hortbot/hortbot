package redis

import (
	"context"
)

const builtinUsageStatsHash = "stats_builtin_usage"

// IncrementBuiltinUsageStat increments a usage statistic by one.
func (db *DB) IncrementBuiltinUsageStat(ctx context.Context, name string) error {
	return db.client.HIncrBy(ctx, builtinUsageStatsHash, name, 1).Err()
}

// GetBuiltinUsageStats gets all usage statistics.
func (db *DB) GetBuiltinUsageStats(ctx context.Context) (map[string]string, error) {
	return db.client.HGetAll(ctx, builtinUsageStatsHash).Result()
}

const actionUsageStatsHash = "stats_action_usage"

// IncrementActionUsageStat increments a usage statistic by one.
func (db *DB) IncrementActionUsageStat(ctx context.Context, name string) error {
	return db.client.HIncrBy(ctx, actionUsageStatsHash, name, 1).Err()
}

// GetActionUsageStats gets all usage statistics.
func (db *DB) GetActionUsageStats(ctx context.Context) (map[string]string, error) {
	return db.client.HGetAll(ctx, actionUsageStatsHash).Result()
}
