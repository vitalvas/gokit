package xcmd

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
)

func WaitInterrupted(ctx context.Context, signals ...os.Signal) error {
	sigChan := make(chan os.Signal, 1)

	if signals == nil {
		signals = append(signals, syscall.SIGINT, syscall.SIGTERM)
	}

	signal.Notify(sigChan, signals...)

	select {
	case v := <-sigChan:
		return errors.New(v.String())

	case <-ctx.Done():
		return ctx.Err()
	}
}
