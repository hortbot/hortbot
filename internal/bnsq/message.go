package bnsq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hortbot/hortbot/internal/bnsq/bnsqmeta"
	"github.com/hortbot/hortbot/internal/pkg/correlation"
	"github.com/rs/xid"
)

type message struct {
	Metadata Metadata        `json:"metadata"`
	Payload  json.RawMessage `json:"payload"`
}

func (m *message) payload(v any) error {
	return json.Unmarshal(m.Payload, v) //nolint:wrapcheck
}

// Metadata contains metadata that will be sent with every NSQ message.
type Metadata struct {
	Timestamp   time.Time `json:"timestamp"`
	Correlation xid.ID    `json:"xid"`
}

// With adds metadata to the context.
func (m *Metadata) With(ctx context.Context) context.Context {
	ctx = correlation.WithID(ctx, m.Correlation)
	ctx = bnsqmeta.WithTimestamp(ctx, m.Timestamp)
	return ctx
}
