package bnsq

import (
	"encoding/json"
	"time"

	"github.com/leononame/clock"
)

type message struct {
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

func newMessage(payload interface{}, clk clock.Clock) (*message, error) {
	p, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &message{
		Timestamp: clk.Now(),
		Payload:   p,
	}, nil
}

func (m *message) payload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}
