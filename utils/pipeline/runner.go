package pipeline

import (
	"context"
	"time"

	logger "github.com/totvs/go-sdk/log"
)

// PeriodicRunner runs pipelines immediately and then on a fixed interval.
type PeriodicRunner struct {
	interval time.Duration
}

// NewPeriodicRunner creates a PeriodicRunner with the given interval.
func NewPeriodicRunner(interval time.Duration) *PeriodicRunner {
	return &PeriodicRunner{interval: interval}
}

// Run executes pipelines immediately and then periodically until ctx is cancelled.
func (r *PeriodicRunner) Run(ctx context.Context, pipelines ...*Pipeline) error {
	l := logger.FromContext(ctx)
	l.Info().Msgf("[Pipeline] Starting periodic runner with interval %v for %d pipeline(s)", r.interval, len(pipelines))

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.runAll(ctx, pipelines...)

	for {
		select {
		case <-ctx.Done():
			l.Info().Msg("[Pipeline] Periodic runner shutting down")
			return nil
		case <-ticker.C:
			r.runAll(ctx, pipelines...)
		}
	}
}

func (r *PeriodicRunner) runAll(ctx context.Context, pipelines ...*Pipeline) {
	for _, p := range pipelines {
		if p != nil {
			if err := p.Run(ctx); err != nil {
				logger.FromContext(ctx).Error(err).Msg("[Pipeline] Pipeline error")
			}
		}
	}
}
