package redis

import (
	"context"

	"go.opencensus.io/trace"
)

const builtinUsageStatsHash = "stats_builtin_usage"

// IncrementBuiltinUsageStat increments a usage statistic by one.
func (db *DB) IncrementBuiltinUsageStat(ctx context.Context, name string) error {
	ctx, span := trace.StartSpan(ctx, "IncrementBuiltinUsageStat")
	defer span.End()
	return db.client.HIncrBy(ctx, builtinUsageStatsHash, name, 1).Err()
}

// GetBuiltinUsageStats gets all usage statistics.
func (db *DB) GetBuiltinUsageStats(ctx context.Context) (map[string]string, error) {
	ctx, span := trace.StartSpan(ctx, "GetBuiltinUsageStats")
	defer span.End()
	return db.client.HGetAll(ctx, builtinUsageStatsHash).Result()
}
