package botstate

import (
	"context"
	"fmt"

	"github.com/hortbot/hortbot/internal/db/dbsql"
)

const raffleNamespace = "raffle"

// RaffleAdd adds a user to the channel's raffle. Duplicates are
// silently ignored.
func (*Store) RaffleAdd(ctx context.Context, queries *dbsql.Queries, channel, user string) error {
	err := queries.BotStateRaffleAdd(ctx, dbsql.BotStateRaffleAddParams{
		Channel: channel,
		UserID:  user,
	})
	if err != nil {
		return fmt.Errorf("raffle add: %w", err)
	}
	return nil
}

// RaffleReset clears the channel's raffle.
func (*Store) RaffleReset(ctx context.Context, queries *dbsql.Queries, channel string) error {
	err := queries.BotStateRaffleReset(ctx, channel)
	if err != nil {
		return fmt.Errorf("raffle reset: %w", err)
	}
	return nil
}

// RaffleCount returns the number of current raffle entries.
func (*Store) RaffleCount(ctx context.Context, queries *dbsql.Queries, channel string) (int64, error) {
	n, err := queries.BotStateRaffleCount(ctx, channel)
	if err != nil {
		return 0, fmt.Errorf("raffle count: %w", err)
	}
	return n, nil
}

// RaffleWinner removes and returns one randomly chosen entry from the
// channel's raffle, or ("", false, nil) if the raffle is empty.
func (s *Store) RaffleWinner(ctx context.Context, queries *dbsql.Queries, channel string) (string, bool, error) {
	winners, err := s.RaffleWinners(ctx, queries, channel, 1)
	if err != nil {
		return "", false, err
	}
	if len(winners) == 0 {
		return "", false, nil
	}
	return winners[0], true, nil
}

// RaffleWinners removes and returns up to n randomly chosen entries.
// exec must be a transaction to serialize concurrent selections.
func (s *Store) RaffleWinners(ctx context.Context, queries *dbsql.Queries, channel string, n int64) ([]string, error) {
	if n <= 0 {
		return []string{}, nil
	}

	var picked []string
	if err := withKeyLock(ctx, queries, raffleNamespace, channel); err != nil {
		return nil, err
	}

	entries, err := queries.BotStateListRaffleEntriesForUpdate(ctx, channel)
	if err != nil {
		return nil, fmt.Errorf("select entries: %w", err)
	}

	if len(entries) == 0 {
		return []string{}, nil
	}

	take := min(int(n), len(entries))
	// Partial Fisher-Yates shuffle.
	for i := range take {
		j := i + s.rand.IntN(len(entries)-i)
		entries[i], entries[j] = entries[j], entries[i]
	}
	picked = entries[:take:take]

	if err := queries.BotStateDeleteRaffleEntries(ctx, dbsql.BotStateDeleteRaffleEntriesParams{
		Channel: channel,
		UserIds: picked,
	}); err != nil {
		return nil, fmt.Errorf("delete winners: %w", err)
	}
	return picked, nil
}
