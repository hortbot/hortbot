package bot

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/db/botstate"
	"github.com/hortbot/hortbot/internal/pkg/dbx"
	"github.com/redis/go-redis/v9"
)

const (
	legacyBuiltinUsageStats = "stats_builtin_usage"
	legacyActionUsageStats  = "stats_action_usage"
	legacyRafflePattern     = "channel:*:raffle:"
)

type legacyRedisState interface {
	Hash(context.Context, string) (map[string]string, error)
	SetMembers(context.Context, string) ([]string, error)
	Scan(context.Context, uint64, string, int64) ([]string, uint64, error)
	Delete(context.Context, ...string) error
}

type redisStateClient struct {
	client *redis.Client
}

func (r redisStateClient) Hash(ctx context.Context, key string) (map[string]string, error) {
	value, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis HGETALL: %w", err)
	}
	return value, nil
}

func (r redisStateClient) SetMembers(ctx context.Context, key string) ([]string, error) {
	value, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis SMEMBERS: %w", err)
	}
	return value, nil
}

func (r redisStateClient) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	keys, next, err := r.client.Scan(ctx, cursor, match, count).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("redis SCAN: %w", err)
	}
	return keys, next, nil
}

func (r redisStateClient) Delete(ctx context.Context, keys ...string) error {
	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("redis DEL: %w", err)
	}
	return nil
}

type redisImportResult struct {
	BuiltinStats int
	ActionStats  int
	Raffles      int
	RaffleUsers  int
}

func importLegacyRedisState(
	ctx context.Context,
	db *sql.DB,
	state *botstate.Store,
	source legacyRedisState,
) (redisImportResult, error) {
	builtins, err := readLegacyUsageStats(ctx, source, legacyBuiltinUsageStats)
	if err != nil {
		return redisImportResult{}, err
	}
	actions, err := readLegacyUsageStats(ctx, source, legacyActionUsageStats)
	if err != nil {
		return redisImportResult{}, err
	}

	if len(builtins) != 0 || len(actions) != 0 {
		err := dbx.Transact(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
			return state.MergeUsageStats(ctx, tx, builtins, actions)
		})
		if err != nil {
			return redisImportResult{}, fmt.Errorf("import Redis usage stats: %w", err)
		}
		if err := source.Delete(ctx, legacyBuiltinUsageStats, legacyActionUsageStats); err != nil {
			return redisImportResult{}, fmt.Errorf("delete imported Redis usage stats: %w", err)
		}
	}

	keys, err := scanLegacyRaffleKeys(ctx, source)
	if err != nil {
		return redisImportResult{}, err
	}

	result := redisImportResult{
		BuiltinStats: len(builtins),
		ActionStats:  len(actions),
	}
	for _, key := range keys {
		channel, err := legacyRaffleChannel(key)
		if err != nil {
			return redisImportResult{}, err
		}
		users, err := source.SetMembers(ctx, key)
		if err != nil {
			return redisImportResult{}, fmt.Errorf("read Redis raffle %q: %w", channel, err)
		}
		slices.Sort(users)

		if len(users) != 0 {
			err := dbx.Transact(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
				for _, user := range users {
					if err := state.RaffleAdd(ctx, tx, channel, user); err != nil {
						return fmt.Errorf("add raffle user: %w", err)
					}
				}
				return nil
			})
			if err != nil {
				return redisImportResult{}, fmt.Errorf("import Redis raffle %q: %w", channel, err)
			}
		}
		if err := source.Delete(ctx, key); err != nil {
			return redisImportResult{}, fmt.Errorf("delete imported Redis raffle %q: %w", channel, err)
		}
		result.Raffles++
		result.RaffleUsers += len(users)
	}

	return result, nil
}

func readLegacyUsageStats(ctx context.Context, source legacyRedisState, key string) (map[string]int64, error) {
	values, err := source.Hash(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("read Redis hash %q: %w", key, err)
	}

	stats := make(map[string]int64, len(values))
	for name, value := range values {
		count, err := strconv.ParseInt(value, 10, 64)
		if err != nil || count < 0 {
			return nil, fmt.Errorf("invalid Redis usage count %q=%q", name, value)
		}
		stats[name] = count
	}
	return stats, nil
}

func scanLegacyRaffleKeys(ctx context.Context, source legacyRedisState) ([]string, error) {
	seen := make(map[string]struct{})
	var cursor uint64
	for {
		keys, next, err := source.Scan(ctx, cursor, legacyRafflePattern, 100)
		if err != nil {
			return nil, fmt.Errorf("scan Redis raffles: %w", err)
		}
		for _, key := range keys {
			seen[key] = struct{}{}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return slices.Sorted(maps.Keys(seen)), nil
}

func legacyRaffleChannel(key string) (string, error) {
	const (
		prefix = "channel:"
		suffix = ":raffle:"
	)
	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
		return "", fmt.Errorf("invalid Redis raffle key %q", key)
	}
	channel := strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
	if channel == "" || strings.Contains(channel, ":") {
		return "", fmt.Errorf("invalid Redis raffle key %q", key)
	}
	return channel, nil
}
