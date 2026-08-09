package botstate

import (
	"context"
	"fmt"

	"github.com/aarondl/sqlboiler/v4/boil"
)

const raffleNamespace = "raffle"

// RaffleAdd adds a user to the channel's raffle. Duplicates are
// silently ignored.
func (*Store) RaffleAdd(ctx context.Context, exec boil.ContextExecutor, channel, user string) error {
	_, err := exec.ExecContext(ctx, `
		INSERT INTO bot_raffle_entries (channel, user_id) VALUES ($1, $2)
		ON CONFLICT (channel, user_id) DO NOTHING
	`, channel, user)
	if err != nil {
		return fmt.Errorf("raffle add: %w", err)
	}
	return nil
}

// RaffleReset clears the channel's raffle.
func (*Store) RaffleReset(ctx context.Context, exec boil.ContextExecutor, channel string) error {
	_, err := exec.ExecContext(ctx,
		`DELETE FROM bot_raffle_entries WHERE channel = $1`, channel)
	if err != nil {
		return fmt.Errorf("raffle reset: %w", err)
	}
	return nil
}

// RaffleCount returns the number of current raffle entries.
func (*Store) RaffleCount(ctx context.Context, exec boil.ContextExecutor, channel string) (int64, error) {
	var n int64
	err := exec.QueryRowContext(ctx,
		`SELECT count(*) FROM bot_raffle_entries WHERE channel = $1`, channel,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("raffle count: %w", err)
	}
	return n, nil
}

// RaffleWinner removes and returns one randomly chosen entry from the
// channel's raffle, or ("", false, nil) if the raffle is empty.
func (s *Store) RaffleWinner(ctx context.Context, exec boil.ContextExecutor, channel string) (string, bool, error) {
	winners, err := s.RaffleWinners(ctx, exec, channel, 1)
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
func (s *Store) RaffleWinners(ctx context.Context, exec boil.ContextExecutor, channel string, n int64) ([]string, error) {
	if n <= 0 {
		return []string{}, nil
	}

	var picked []string
	if err := withKeyLock(ctx, exec, raffleNamespace, channel); err != nil {
		return nil, err
	}

	rows, err := exec.QueryContext(ctx, `
			SELECT user_id FROM bot_raffle_entries
			WHERE channel = $1
			ORDER BY user_id
			FOR UPDATE
		`, channel)
	if err != nil {
		return nil, fmt.Errorf("select entries: %w", err)
	}

	var entries []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		entries = append(entries, u)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("rows err: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("rows close: %w", err)
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

	if _, err := exec.ExecContext(ctx, `
			DELETE FROM bot_raffle_entries
			WHERE channel = $1 AND user_id = ANY($2)
		`, channel, picked); err != nil {
		return nil, fmt.Errorf("delete winners: %w", err)
	}
	return picked, nil
}
