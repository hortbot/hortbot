package redis

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
)

const dedupe = "dedupe"

func (db *DB) DedupeMark(ctx context.Context, id string, expiry time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "DedupeMark")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(dedupe, id)
	return mark(client, key, expiry)
}

func (db *DB) DedupeCheck(ctx context.Context, id string, expiry time.Duration) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "DedupeCheck")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(dedupe, id)
	return checkAndRefresh(client, key, expiry)
}

func (db *DB) DedupeCheckAndMark(ctx context.Context, id string, expiry time.Duration) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "DedupeCheckAndMark")
	defer span.Finish()

	client := db.client.WithContext(ctx)
	key := buildKey(dedupe, id)
	return checkAndMark(client, key, expiry)
}
