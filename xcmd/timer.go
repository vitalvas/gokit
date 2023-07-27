package xcmd

import (
	"context"
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
