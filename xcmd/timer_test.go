package xcmd

import (
	"context"
	"errors"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeriodicRun(t *testing.T) {
	t.Run("executes function periodically", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		counter := 0
		var mu sync.Mutex

		err := PeriodicRun(ctx, func(_ context.Context) error {
			mu.Lock()
			counter++
			mu.Unlock()
			return nil
		}, 50*time.Millisecond)

		require.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)

		mu.Lock()
		count := counter
		mu.Unlock()

		assert.GreaterOrEqual(t, count, 3)
		assert.LessOrEqual(t, count, 6)
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		counter := 0
		var mu sync.Mutex

		done := make(chan error, 1)
		go func() {
			done <- PeriodicRun(ctx, func(_ context.Context) error {
				mu.Lock()
				counter++
				mu.Unlock()
				return nil
			}, 50*time.Millisecond)
		}()

		time.Sleep(125 * time.Millisecond)
		cancel()

		err := <-done
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)

		mu.Lock()
		count := counter
		mu.Unlock()

		assert.GreaterOrEqual(t, count, 1)
		assert.LessOrEqual(t, count, 4)
	})

	t.Run("stops on execute function error", func(t *testing.T) {
		ctx := context.Background()

		counter := 0
		expectedErr := errors.New("execution error")

		err := PeriodicRun(ctx, func(_ context.Context) error {
			counter++
			if counter >= 3 {
				return expectedErr
			}
			return nil
		}, 10*time.Millisecond)

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 3, counter)
	})

	t.Run("waits for first tick before execution", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		executed := false
		var mu sync.Mutex

		done := make(chan error, 1)
		go func() {
			done <- PeriodicRun(ctx, func(_ context.Context) error {
				mu.Lock()
				executed = true
				mu.Unlock()
				return nil
			}, 50*time.Millisecond)
		}()

		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		firstCheck := executed
		mu.Unlock()
		assert.False(t, firstCheck, "should not execute before first tick")

		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		secondCheck := executed
		mu.Unlock()
		assert.True(t, secondCheck, "should execute after first tick")

		<-done
	})
}

func TestPeriodicRunWithSignal(t *testing.T) {
	t.Run("executes function periodically", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancel()

		counter := 0
		var mu sync.Mutex

		err := PeriodicRunWithSignal(ctx, func(_ context.Context) error {
			mu.Lock()
			counter++
			mu.Unlock()
			return nil
		}, 50*time.Millisecond)

		require.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)

		mu.Lock()
		count := counter
		mu.Unlock()

		assert.GreaterOrEqual(t, count, 3)
		assert.LessOrEqual(t, count, 6)
	})

	t.Run("executes on signal", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		counter := 0
		var mu sync.Mutex
		signalReceived := make(chan struct{}, 1)
		started := make(chan struct{})

		done := make(chan error, 1)
		go func() {
			close(started)
			done <- PeriodicRunWithSignal(ctx, func(_ context.Context) error {
				mu.Lock()
				counter++
				if counter >= 1 {
					select {
					case signalReceived <- struct{}{}:
					default:
					}
				}
				mu.Unlock()
				return nil
			}, 2*time.Second, syscall.SIGUSR1)
		}()

		<-started
		time.Sleep(50 * time.Millisecond)

		proc, err := os.FindProcess(os.Getpid())
		require.NoError(t, err)

		err = proc.Signal(syscall.SIGUSR1)
		require.NoError(t, err)

		select {
		case <-signalReceived:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("signal execution timeout")
		}

		mu.Lock()
		finalCount := counter
		mu.Unlock()

		assert.GreaterOrEqual(t, finalCount, 1)

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("test cleanup timeout")
		}
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		counter := 0
		var mu sync.Mutex

		done := make(chan error, 1)
		go func() {
			done <- PeriodicRunWithSignal(ctx, func(_ context.Context) error {
				mu.Lock()
				counter++
				mu.Unlock()
				return nil
			}, 50*time.Millisecond, syscall.SIGUSR1)
		}()

		time.Sleep(125 * time.Millisecond)
		cancel()

		err := <-done
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("stops on execute function error", func(t *testing.T) {
		ctx := context.Background()

		counter := 0
		expectedErr := errors.New("execution error")

		err := PeriodicRunWithSignal(ctx, func(_ context.Context) error {
			counter++
			if counter >= 3 {
				return expectedErr
			}
			return nil
		}, 10*time.Millisecond, syscall.SIGUSR1)

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 3, counter)
	})

	t.Run("uses default signal when nil", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		executed := false

		err := PeriodicRunWithSignal(ctx, func(_ context.Context) error {
			executed = true
			return nil
		}, 50*time.Millisecond)

		require.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		assert.True(t, executed)
	})

	t.Run("handles multiple signals", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		counter := 0
		var mu sync.Mutex
		executed := make(chan struct{}, 3)
		started := make(chan struct{})

		done := make(chan error, 1)
		go func() {
			close(started)
			done <- PeriodicRunWithSignal(ctx, func(_ context.Context) error {
				mu.Lock()
				counter++
				mu.Unlock()
				select {
				case executed <- struct{}{}:
				default:
				}
				return nil
			}, 10*time.Second, syscall.SIGUSR1, syscall.SIGUSR2)
		}()

		<-started
		time.Sleep(50 * time.Millisecond)

		proc, err := os.FindProcess(os.Getpid())
		require.NoError(t, err)

		err = proc.Signal(syscall.SIGUSR1)
		require.NoError(t, err)

		select {
		case <-executed:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("first signal execution timeout")
		}

		time.Sleep(50 * time.Millisecond)

		err = proc.Signal(syscall.SIGUSR2)
		require.NoError(t, err)

		select {
		case <-executed:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("second signal execution timeout")
		}

		mu.Lock()
		count := counter
		mu.Unlock()

		assert.GreaterOrEqual(t, count, 2)

		cancel()

		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("test cleanup timeout")
		}
	})
}
