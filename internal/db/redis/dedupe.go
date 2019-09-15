package redis

import (
	"context"
	"time"

	"go.opencensus.io/trace"
)

const keyDedupe = keyStr("dedupe")

func (db *DB) DedupeMark(ctx context.Context, id string, expiry time.Duration) error {
	ctx, span := trace.StartSpan(ctx, "DedupeMark")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(keyDedupe.is(id))
	return mark(client, key, expiry)
}

func (db *DB) DedupeCheck(ctx context.Context, id string, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "DedupeCheck")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(keyDedupe.is(id))
	return checkAndRefresh(client, key, expiry)
}

func (db *DB) DedupeCheckAndMark(ctx context.Context, id string, expiry time.Duration) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "DedupeCheckAndMark")
	defer span.End()

	client := db.client.WithContext(ctx)
	key := buildKey(keyDedupe.is(id))
	return checkAndMark(client, key, expiry)
}
