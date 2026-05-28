package backtest

import (
	"context"
	"time"
)

// DecisionInput is the per-cycle input passed into a decision generator.
type DecisionInput struct {
	Config       Config
	Cycle        int
	Symbol       string
	DecisionTime time.Time
	Windows      map[string][]Candle
}

// DecisionGenerator produces cycle-level decision payloads for replay.
type DecisionGenerator interface {
	Generate(context.Context, DecisionInput) (map[string]any, error)
}
