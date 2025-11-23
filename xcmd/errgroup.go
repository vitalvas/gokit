package xcmd

import (
	"context"
	"sync"
)

// Group is similar to errgroup.Group but cancels all goroutines on first error.
// Unlike errgroup which waits for all goroutines to complete, this Group
// cancels the context immediately when the first error occurs.
type Group struct {
	ctx     context.Context
	cancel  context.CancelCauseFunc
	wg      sync.WaitGroup
	errOnce sync.Once
	err     error
}

// ErrGroup returns a new Group and an associated Context derived from ctx.
// The derived Context is canceled when the first goroutine returns an error,
// or when all goroutines complete successfully, whichever happens first.
func ErrGroup(ctx context.Context) (*Group, context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)
	return &Group{ctx: ctx, cancel: cancel}, ctx
}

// Go calls the given function in a new goroutine.
// The first call to return a non-nil error cancels the group's context.
// All subsequent errors are ignored.
func (g *Group) Go(f func(ctx context.Context) error) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		if err := f(g.ctx); err != nil {
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel(err)
				}
			})
		}
	}()
}

// Wait blocks until all function calls from the Go method have returned,
// then returns the first non-nil error (if any) from them.
func (g *Group) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel(nil)
	}
	return g.err
}
