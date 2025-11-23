package xcmd

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroup(t *testing.T) {
	t.Run("success with no errors", func(t *testing.T) {
		group, ctx := ErrGroup(context.Background())

		executed := make([]bool, 3)

		for i := 0; i < 3; i++ {
			idx := i
			group.Go(func(_ context.Context) error {
				executed[idx] = true
				return nil
			})
		}

		err := group.Wait()
		require.NoError(t, err)
		assert.Equal(t, []bool{true, true, true}, executed)
		assert.Error(t, ctx.Err())
	})

	t.Run("first error cancels context", func(t *testing.T) {
		group, ctx := ErrGroup(context.Background())

		expectedErr := errors.New("test error")
		started := make(chan struct{})
		blocked := make(chan struct{})

		group.Go(func(_ context.Context) error {
			close(started)
			<-blocked
			return nil
		})

		group.Go(func(_ context.Context) error {
			<-started
			return expectedErr
		})

		<-ctx.Done()
		close(blocked)

		err := group.Wait()
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, expectedErr, context.Cause(ctx))
	})

	t.Run("multiple errors returns first", func(t *testing.T) {
		group, ctx := ErrGroup(context.Background())

		err1 := errors.New("error 1")
		err2 := errors.New("error 2")

		started := make(chan struct{}, 2)

		group.Go(func(_ context.Context) error {
			started <- struct{}{}
			time.Sleep(10 * time.Millisecond)
			return err1
		})

		group.Go(func(_ context.Context) error {
			started <- struct{}{}
			time.Sleep(10 * time.Millisecond)
			return err2
		})

		<-started
		<-started

		err := group.Wait()
		require.Error(t, err)
		assert.True(t, err == err1 || err == err2)
		assert.Error(t, ctx.Err())
	})

	t.Run("context cancellation propagates", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		group, groupCtx := ErrGroup(ctx)

		blocked := make(chan struct{})
		ctxCancelled := make(chan struct{})

		group.Go(func(_ context.Context) error {
			<-blocked
			return nil
		})

		group.Go(func(_ context.Context) error {
			<-groupCtx.Done()
			close(ctxCancelled)
			return groupCtx.Err()
		})

		cancel()
		<-ctxCancelled
		close(blocked)

		err := group.Wait()
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("wait after error completes", func(t *testing.T) {
		group, ctx := ErrGroup(context.Background())

		expectedErr := errors.New("test error")

		group.Go(func(_ context.Context) error {
			return expectedErr
		})

		err := group.Wait()
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Error(t, ctx.Err())
	})

	t.Run("empty group returns no error", func(t *testing.T) {
		group, ctx := ErrGroup(context.Background())

		err := group.Wait()
		require.NoError(t, err)
		assert.Error(t, ctx.Err())
	})

	t.Run("concurrent goroutines execution", func(t *testing.T) {
		group, _ := ErrGroup(context.Background())

		const numGoroutines = 100
		counter := 0
		var mu sync.Mutex

		for i := 0; i < numGoroutines; i++ {
			group.Go(func(_ context.Context) error {
				mu.Lock()
				counter++
				mu.Unlock()
				return nil
			})
		}

		err := group.Wait()
		require.NoError(t, err)
		assert.Equal(t, numGoroutines, counter)
	})

	t.Run("error stops other goroutines via context", func(t *testing.T) {
		group, _ := ErrGroup(context.Background())

		expectedErr := errors.New("stop error")
		longRunning := make(chan bool, 1)

		group.Go(func(_ context.Context) error {
			return expectedErr
		})

		group.Go(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				longRunning <- true
				return ctx.Err()
			case <-time.After(5 * time.Second):
				longRunning <- false
				return nil
			}
		})

		err := group.Wait()
		require.Error(t, err)

		stopped := <-longRunning
		assert.True(t, stopped, "long running goroutine should have been cancelled")
	})
}

func TestGroupRace(t *testing.T) {
	t.Run("race condition on error assignment", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			group, _ := ErrGroup(context.Background())

			err1 := errors.New("error 1")
			err2 := errors.New("error 2")

			group.Go(func(_ context.Context) error {
				return err1
			})

			group.Go(func(_ context.Context) error {
				return err2
			})

			err := group.Wait()
			require.Error(t, err)
			assert.True(t, err == err1 || err == err2)
		}
	})
}
