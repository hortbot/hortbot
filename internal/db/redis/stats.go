package redis

import "context"

const usageStatsHash = "statistics"

// IncrementUsageStatistic increments a usage statistic by one.
func (db *DB) IncrementUsageStatistic(ctx context.Context, name string) error {
	return db.client.HIncrBy(ctx, usageStatsHash, name, 1).Err()
}

// GetUsageStatistics gets all usage statistics.
func (db *DB) GetUsageStatistics(ctx context.Context) (map[string]string, error) {
	return db.client.HGetAll(ctx, usageStatsHash).Result()
}
