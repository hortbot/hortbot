package redis

import (
	"context"
	"time"

	"go.opencensus.io/trace"
)

const keyDedupe = keyStr("dedupe")

// DedupeMark marks a given ID as seen.
func (db *DB) DedupeMark(ctx context.Context, id string, expiry time.Duration) error {
	ctx, span := trace.StartSpan(ctx, "DedupeMark")
	defer span.End()

	key := buildKey(keyDedupe.is(id))
	return mark(ctx, db.client, key, expiry)
}

// DedupeCheck checks if an ID has been seen, and if seen refreshes its expiry.
func (db *DB) DedupeCheck(ctx context.Context, id string, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "DedupeCheck")
	defer span.End()

	key := buildKey(keyDedupe.is(id))
	return checkAndRefresh(ctx, db.client, key, expiry)
}

// DedupeCheckAndMark checks if an ID has been seen, and if it not, marks it as seen.
func (db *DB) DedupeCheckAndMark(ctx context.Context, id string, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "DedupeCheckAndMark")
	defer span.End()

	key := buildKey(keyDedupe.is(id))
	return checkAndMark(ctx, db.client, key, expiry)
}
