package wqueue_test

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/wqueue"
	"go.uber.org/atomic"
)

func BenchmarkQueueSameName(b *testing.B) {
	for workers := 0; workers < 6; workers++ {
		workers := int(math.Pow(2, float64(workers)))

		b.Run(fmt.Sprintf("%d workers", workers), func(b *testing.B) {
			q := wqueue.NewQueue(16)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var workerWG sync.WaitGroup
			workerWG.Add(workers)

			for i := 0; i < workers; i++ {
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

	for workers := 0; workers < 6; workers++ {
		workers := int(math.Pow(2, float64(workers)))

		b.Run(fmt.Sprintf("%d workers", workers), func(b *testing.B) {
			q := wqueue.NewQueue(16)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var workerWG sync.WaitGroup
			workerWG.Add(workers)

			for i := 0; i < workers; i++ {
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
					q.Put(ctx, strconv.FormatUint(name.Inc()%names, 10), fn) //nolint:errcheck
				}
			})

			wg.Wait()
			cancel()
			workerWG.Wait()
		})
	}
}
