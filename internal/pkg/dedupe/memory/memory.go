package memory

import (
	"context"
	"sync"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/dedupe"
)

type Dedupe struct {
	m sync.Map

	expiry        time.Duration
	pruneInterval time.Duration
	stopChan      chan struct{}
}

func New(expiry time.Duration, pruneInterval time.Duration) *Dedupe {
	d := &Dedupe{
		expiry:        expiry,
		pruneInterval: pruneInterval,
		stopChan:      make(chan struct{}),
	}

	go d.run()

	return d
}

var _ dedupe.Deduplicator = (*Dedupe)(nil)

func (d *Dedupe) Mark(_ context.Context, id string) error {
	var expire interface{} = time.Now().Add(d.expiry)
	d.m.Store(id, expire)
	return nil
}

func (d *Dedupe) Check(_ context.Context, id string) (seen bool, err error) {
	_, seen = d.m.Load(id)
	if seen {
		var expire interface{} = time.Now().Add(d.expiry)
		d.m.Store(id, expire)
	}

	return seen, nil
}

func (d *Dedupe) CheckAndMark(_ context.Context, id string) (seen bool, err error) {
	var expire interface{} = time.Now().Add(d.expiry)

	_, seen = d.m.LoadOrStore(id, expire)
	if seen {
		d.m.Store(id, expire)
	}

	return seen, nil
}

func (d *Dedupe) Stop() {
	close(d.stopChan)
}

func (d *Dedupe) run() {
	ticker := time.NewTicker(d.pruneInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopChan:
			return
		case <-ticker.C:
		}

		now := time.Now()

		d.m.Range(func(key, value interface{}) bool {
			if now.After(value.(time.Time)) {
				d.m.Delete(key)
			}
			return true
		})
	}
}
