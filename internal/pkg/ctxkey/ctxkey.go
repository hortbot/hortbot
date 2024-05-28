package ctxkey

import "context"

type key struct {
	name string
}

func (k key) String() string {
	return k.name
}

type ContextKey[T any] struct {
	key          *key
	defaultValue T
}

func NewContextKey[T any](name string, defaultValue T) ContextKey[T] {
	return ContextKey[T]{
		key:          &key{name: name},
		defaultValue: defaultValue,
	}
}

func (k ContextKey[T]) Value(ctx context.Context) T {
	if v := ctx.Value(k.key); v != nil {
		return v.(T)
	}
	return k.defaultValue
}

func (k ContextKey[T]) WithValue(ctx context.Context, value T) context.Context {
	return context.WithValue(ctx, k.key, value)
}
