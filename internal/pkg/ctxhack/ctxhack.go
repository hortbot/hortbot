package ctxhack

import "context"

type combination struct {
	context.Context
	values []context.Context
}

// Combine combines contexts together. The returned context will get Deadline,
// Done, and Err from the first context (cancellation), and values from the
// rest, with a search in order.
func Combine(cancellation context.Context, values ...context.Context) context.Context {
	return &combination{
		Context: cancellation,
		values:  values,
	}
}

func (c *combination) Value(key interface{}) interface{} {
	for _, ctx := range c.values {
		if v := ctx.Value(key); v != nil {
			return v
		}
	}
	return nil
}
