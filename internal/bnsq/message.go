package bnsq

import (
	"encoding/json"
	"time"

	"github.com/opentracing/opentracing-go"
)

type message struct {
	Timestamp    time.Time                  `json:"timestamp"`
	TraceCarrier opentracing.TextMapCarrier `json:"trace_carrier"`
	Payload      json.RawMessage            `json:"payload"`
}

func (m *message) payload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}
