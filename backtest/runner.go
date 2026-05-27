package backtest

import (
	"context"
	"fmt"
	"time"
)

// RunnerDependencies will be filled in phase-by-phase as the MVP gets wired.
type RunnerDependencies struct {
	DataSource DataSource
	Writer     ResultWriter
}

// Runner orchestrates one backtest run.
type Runner struct {
	deps RunnerDependencies
}

func NewRunner(deps RunnerDependencies) *Runner {
	return &Runner{deps: deps}
}

func (r *Runner) Run(ctx context.Context, cfg Config) (*Result, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if r.deps.DataSource == nil {
		return nil, fmt.Errorf("backtest data source is not configured")
	}

	startedAt := time.Now().UTC()
	result := &Result{
		RunID:  startedAt.Format("20060102-150405"),
		Config: cfg,
		Summary: Summary{
			Status:         RunStatusPending,
			StartTime:      cfg.StartTime,
			EndTime:        cfg.EndTime,
			InitialCapital: cfg.InitialCapital,
			DecisionMode:   cfg.DecisionMode,
		},
	}

	result.Summary.Status = RunStatusRunning

	// MVP skeleton only: wiring and validation first.
	// Replay loop / strategy integration / simulated fills will be added next.
	result.Summary.Status = RunStatusCompleted
	result.Summary.EndingEquity = cfg.InitialCapital
	result.Summary.TotalReturnPct = 0
	result.Summary.MaxDrawdownPct = 0
	result.Summary.Notes = append(result.Summary.Notes,
		"Backtest MVP skeleton is initialized.",
		"Replay loop and execution simulation are the next implementation step.",
	)

	if r.deps.Writer != nil {
		if err := r.deps.Writer.Write(ctx, *result); err != nil {
			return nil, err
		}
	}

	return result, nil
}
