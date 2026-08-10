// Package botstate stores transient bot state and usage counters in
// PostgreSQL.
//
// Production TTL decisions use PostgreSQL's clock so all application
// processes agree on expiry. Tests can inject a clock with WithNow.
package botstate

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
)

// Rand is the subset of math/rand/v2 used for raffle pops.
type Rand interface {
	IntN(n int) int
}

// Store provides typed access to bot state stored in PostgreSQL.
type Store struct {
	now  func() time.Time
	rand Rand
}

// Option configures a Store.
type Option func(*Store)

// WithNow overrides the PostgreSQL-backed time source.
func WithNow(now func() time.Time) Option {
	return func(s *Store) { s.now = now }
}

// WithRand overrides the random source used for raffle pops. The
// default uses a fresh math/rand/v2 generator.
func WithRand(r Rand) Option {
	return func(s *Store) { s.rand = r }
}

// New constructs a Store.
func New(opts ...Option) *Store {
	d := &Store{
		rand: defaultRand{},
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (s *Store) currentTime(ctx context.Context, queries *dbsql.Queries) (time.Time, error) {
	if s.now != nil {
		return s.now(), nil
	}

	now, err := queries.BotStateCurrentTime(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("get database time: %w", err)
	}
	return now.Time, nil
}

type defaultRand struct{}

func (defaultRand) IntN(n int) int { return rand.IntN(n) } //nolint:gosec // not security-sensitive: used for raffle pop ordering only.
