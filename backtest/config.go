package backtest

import (
	"fmt"
	"time"
)

// Config defines one backtest run.
type Config struct {
	Name            string        `json:"name"`
	TraderID        string        `json:"trader_id"`
	StrategyID      string        `json:"strategy_id"`
	UserID          string        `json:"user_id"`
	Exchange        string        `json:"exchange"`
	Symbols         []string      `json:"symbols,omitempty"`
	CandidateLimit  int           `json:"candidate_limit,omitempty"`
	InitialCapital  float64       `json:"initial_capital"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	DecisionTF      BarTimeframe  `json:"decision_timeframe"`
	FillMode        OrderFillMode `json:"fill_mode"`
	DecisionMode    DecisionMode  `json:"decision_mode"`
	UseFunding      bool          `json:"use_funding"`
	UseOpenInterest bool          `json:"use_open_interest"`
	TakerFeeRate    float64       `json:"taker_fee_rate"`
	SlippageBps     float64       `json:"slippage_bps"`
	MaxCycles       int           `json:"max_cycles,omitempty"`
	OutputDir       string        `json:"output_dir,omitempty"`
}

func (c Config) Validate() error {
	if c.InitialCapital <= 0 {
		return fmt.Errorf("initial_capital must be > 0")
	}
	if c.StartTime.IsZero() || c.EndTime.IsZero() {
		return fmt.Errorf("start_time and end_time are required")
	}
	if !c.EndTime.After(c.StartTime) {
		return fmt.Errorf("end_time must be after start_time")
	}
	if c.DecisionTF == "" {
		return fmt.Errorf("decision_timeframe is required")
	}
	if c.FillMode == "" {
		return fmt.Errorf("fill_mode is required")
	}
	if c.DecisionMode == "" {
		return fmt.Errorf("decision_mode is required")
	}
	return nil
}
