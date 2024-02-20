package pool

import "sync"

type Pool[T any] struct {
	p sync.Pool
}

func NewPool[T any](newFn func() T) *Pool[T] {
	return &Pool[T]{
		p: sync.Pool{
			New: func() any {
				return newFn()
			},
		},
	}
}

func (p *Pool[T]) Get() T {
	return p.p.Get().(T)
}

func (p *Pool[T]) Put(x T) {
	p.p.Put(x)
}
