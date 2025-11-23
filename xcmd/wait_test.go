package xcmd

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitInterrupted(t *testing.T) {
	t.Run("waits for SIGINT", func(t *testing.T) {
		ctx := context.Background()

		done := make(chan error, 1)
		go func() {
			done <- WaitInterrupted(ctx, syscall.SIGINT)
		}()

		time.Sleep(50 * time.Millisecond)

		proc, err := os.FindProcess(os.Getpid())
		require.NoError(t, err)

		err = proc.Signal(syscall.SIGINT)
		require.NoError(t, err)

		select {
		case err := <-done:
			require.Error(t, err)
			assert.Contains(t, err.Error(), "interrupt")
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for signal")
		}
	})

	t.Run("waits for SIGTERM", func(t *testing.T) {
		ctx := context.Background()

		done := make(chan error, 1)
		go func() {
			done <- WaitInterrupted(ctx, syscall.SIGTERM)
		}()

		time.Sleep(50 * time.Millisecond)

		proc, err := os.FindProcess(os.Getpid())
		require.NoError(t, err)

		err = proc.Signal(syscall.SIGTERM)
		require.NoError(t, err)

		select {
		case err := <-done:
			require.Error(t, err)
			assert.Contains(t, err.Error(), "terminated")
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for signal")
		}
	})

	t.Run("uses default signals when nil", func(t *testing.T) {
		ctx := context.Background()

		done := make(chan error, 1)
		go func() {
			done <- WaitInterrupted(ctx)
		}()

		time.Sleep(50 * time.Millisecond)

		proc, err := os.FindProcess(os.Getpid())
		require.NoError(t, err)

		err = proc.Signal(syscall.SIGINT)
		require.NoError(t, err)

		select {
		case err := <-done:
			require.Error(t, err)
			assert.Contains(t, err.Error(), "interrupt")
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for signal")
		}
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)
		go func() {
			done <- WaitInterrupted(ctx, syscall.SIGINT, syscall.SIGTERM)
		}()

		time.Sleep(50 * time.Millisecond)
		cancel()

		select {
		case err := <-done:
			require.Error(t, err)
			assert.Equal(t, context.Canceled, err)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for context cancellation")
		}
	})

	t.Run("stops on context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := WaitInterrupted(ctx, syscall.SIGINT, syscall.SIGTERM)

		require.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("handles custom signal", func(t *testing.T) {
		ctx := context.Background()

		done := make(chan error, 1)
		go func() {
			done <- WaitInterrupted(ctx, syscall.SIGUSR1)
		}()

		time.Sleep(50 * time.Millisecond)

		proc, err := os.FindProcess(os.Getpid())
		require.NoError(t, err)

		err = proc.Signal(syscall.SIGUSR1)
		require.NoError(t, err)

		select {
		case err := <-done:
			require.Error(t, err)
			assert.Contains(t, err.Error(), "user defined signal 1")
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for signal")
		}
	})

	t.Run("handles multiple signals", func(t *testing.T) {
		ctx := context.Background()

		done := make(chan error, 1)
		go func() {
			done <- WaitInterrupted(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
		}()

		time.Sleep(50 * time.Millisecond)

		proc, err := os.FindProcess(os.Getpid())
		require.NoError(t, err)

		err = proc.Signal(syscall.SIGUSR1)
		require.NoError(t, err)

		select {
		case err := <-done:
			require.Error(t, err)
			assert.Contains(t, err.Error(), "user defined signal 1")
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for signal")
		}
	})
}
