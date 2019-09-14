package bnsq

import (
	"encoding/json"
	"time"
)

type message struct {
	Timestamp time.Time       `json:"timestamp"`
	TraceSpan []byte          `json:"trace_span"`
	Payload   json.RawMessage `json:"payload"`
}

func (m *message) payload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}
