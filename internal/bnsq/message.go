package bnsq

import (
	"encoding/json"
	"time"

	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

type message struct {
	Metadata Metadata        `json:"metadata"`
	Payload  json.RawMessage `json:"payload"`
}

func (m *message) payload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}

type Metadata struct {
	Timestamp time.Time `json:"timestamp"`
	TraceSpan []byte    `json:"trace_span"`
}

func (m *Metadata) ParentSpan() trace.SpanContext {
	parent, _ := propagation.FromBinary(m.TraceSpan)
	return parent
}
