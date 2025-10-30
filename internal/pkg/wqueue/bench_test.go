package wqueue_test

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/wqueue"
)

func BenchmarkQueueSameName(b *testing.B) {
	for workers := range 6 {
		workers := int(math.Pow(2, float64(workers)))

		b.Run(fmt.Sprintf("%d workers", workers), func(b *testing.B) {
			q := wqueue.NewQueue[string](16)

			ctx, cancel := context.WithCancel(b.Context())
			defer cancel()

			var workerWG sync.WaitGroup
			workerWG.Add(workers)

			for range workers {
				go func() {
					defer workerWG.Done()
					q.Worker(ctx) //nolint:errcheck
				}()
			}

			var wg sync.WaitGroup
			fn := func(attach wqueue.Attacher) {
				wg.Done()
			}

			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					wg.Add(1)
					q.Put(ctx, "queue", fn) //nolint:errcheck
				}
			})

			wg.Wait()
			cancel()
			workerWG.Wait()
		})
	}
}

func BenchmarkQueueManyNames(b *testing.B) {
	const names = 200

	for workers := range 6 {
		workers := int(math.Pow(2, float64(workers)))

		b.Run(fmt.Sprintf("%d workers", workers), func(b *testing.B) {
			q := wqueue.NewQueue[string](16)

			ctx, cancel := context.WithCancel(b.Context())
			defer cancel()

			var workerWG sync.WaitGroup
			workerWG.Add(workers)

			for range workers {
				go func() {
					defer workerWG.Done()
					q.Worker(ctx) //nolint:errcheck
				}()
			}

			var wg sync.WaitGroup
			fn := func(attach wqueue.Attacher) {
				wg.Done()
			}

			var name atomic.Uint64

			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					wg.Add(1)
					q.Put(ctx, strconv.FormatUint(name.Add(1)%names, 10), fn) //nolint:errcheck
				}
			})

			wg.Wait()
			cancel()
			workerWG.Wait()
		})
	}
}
