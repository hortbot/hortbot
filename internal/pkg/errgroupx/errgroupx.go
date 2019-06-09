// Package errgroupx implements a modified version of errgroup's Group, keeping
// the derived context internally and passing to functions called via Go.
//
// Storing a context in a struct is technically an antipattern, but using
// errgroup in practice can introduce unwanted context variables into the scope
// where the group is used.
package errgroupx

import (
	"context"
	"errors"

	"golang.org/x/sync/errgroup"
)

// ErrStop is a sentinel error used when stopping a group.
var ErrStop = errors.New("errgroupx: stop")

// Group wraps errgroup's Group, keeping the derived context to pass to
// functions called via Go.
type Group struct {
	ctx context.Context
	g   *errgroup.Group
}

// FromContext returns a new Group derived from ctx.
//
// The derived Context is canceled the first time a function passed to Go
// returns a non-nil error or the first time Wait returns, whichever occurs
// first.
func FromContext(ctx context.Context) *Group {
	g, gctx := errgroup.WithContext(ctx)
	return &Group{
		ctx: gctx,
		g:   g,
	}
}

// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error cancels the group; its error will be
// returned by Wait.
func (g *Group) Go(f func(context.Context) error) {
	g.g.Go(func() error {
		return f(g.ctx)
	})
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
func (g *Group) Wait() error {
	return g.g.Wait()
}

// Ctx returns the group's context.
func (g *Group) Ctx() context.Context {
	return g.ctx
}

// Stop stops the group by running a function on it that returns ErrStop.
func (g *Group) Stop() {
	g.g.Go(func() error {
		return ErrStop
	})
}

// WaitIgnoreStop works like Wait, but will return nil if the error is ErrStop.
func (g *Group) WaitIgnoreStop() error {
	err := g.g.Wait()
	if err == ErrStop {
		return nil
	}
	return err
}
