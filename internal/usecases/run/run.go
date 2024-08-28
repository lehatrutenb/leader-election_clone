package run

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/metrics"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states"
)

var _ Runner = &LoopRunner{}

type Runner interface {
	Run(ctx context.Context, state states.AutomataState) error
}

func NewLoopRunner(logger *slog.Logger, metrics *metrics.Metrics) *LoopRunner {
	logger = logger.With("subsystem", "StateRunner")
	return &LoopRunner{
		logger:  logger,
		metrics: metrics,
	}
}

type LoopRunner struct {
	logger  *slog.Logger
	metrics *metrics.Metrics
}

func (r *LoopRunner) Run(ctx context.Context, state states.AutomataState) error {
	for state != nil {
		r.logger.LogAttrs(ctx, slog.LevelInfo, "start running state", slog.String("state", state.String()))
		r.metrics.CurState.Set(float64(state.Int()))
		r.metrics.AmtStateChanges.Inc()
		r.metrics.CurStateStartTime.SetToCurrentTime()
		r.metrics.CurStateStartTime.Desc()

		var err error
		state, err = state.Run(ctx)
		if err != nil {
			return fmt.Errorf("state %s run: %w", state.String(), err)
		}
	}
	r.logger.LogAttrs(ctx, slog.LevelInfo, "no new state, finish")
	return nil
}
