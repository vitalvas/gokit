package xcmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func PeriodicRun(ctx context.Context, execute func(ctx context.Context) error, period time.Duration) error {
	timer := time.NewTicker(period)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-timer.C:
			if err := execute(ctx); err != nil {
				return err
			}
		}
	}
}

func PeriodicRunWithSignal(
	ctx context.Context,
	execute func(ctx context.Context) error,
	period time.Duration,
	signals ...os.Signal,
) error {
	timer := time.NewTicker(period)
	defer timer.Stop()

	if signals == nil {
		signals = append(signals, syscall.SIGUSR1)
	}

	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, signals...)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-timer.C:
			if err := execute(ctx); err != nil {
				return err
			}

		case <-sigChan:
			if err := execute(ctx); err != nil {
				return err
			}
		}
	}
}
