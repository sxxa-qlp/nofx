package backtest

import "time"

// RunStatus represents backtest run state.
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
)

// DecisionMode controls how strategy decisions are produced during backtest.
type DecisionMode string

const (
	DecisionModeLiveAI DecisionMode = "live_ai"
	DecisionModeRules  DecisionMode = "rules_only"
	DecisionModeReplay DecisionMode = "replay_cached"
)

// BarTimeframe is the replay decision clock.
type BarTimeframe string

const (
	Timeframe15m BarTimeframe = "15m"
	Timeframe1h  BarTimeframe = "1h"
	Timeframe4h  BarTimeframe = "4h"
)

// OrderFillMode defines the simplified fill assumption for MVP.
type OrderFillMode string

const (
	FillNextBarOpen OrderFillMode = "next_bar_open"
)

// BacktestEventType captures major lifecycle events.
type BacktestEventType string

const (
	EventCycleStart    BacktestEventType = "cycle_start"
	EventDecision      BacktestEventType = "decision"
	EventOrderFilled   BacktestEventType = "order_filled"
	EventPositionClose BacktestEventType = "position_close"
	EventSnapshot      BacktestEventType = "equity_snapshot"
)

// BacktestEvent is a timeline record for later debugging/reporting.
type BacktestEvent struct {
	Time    time.Time         `json:"time"`
	Type    BacktestEventType `json:"type"`
	Symbol  string            `json:"symbol,omitempty"`
	Message string            `json:"message,omitempty"`
	Payload map[string]any    `json:"payload,omitempty"`
}

// EquityPoint is a point on the equity curve.
type EquityPoint struct {
	Time          time.Time `json:"time"`
	Equity        float64   `json:"equity"`
	Available     float64   `json:"available"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	RealizedPnL   float64   `json:"realized_pnl"`
	DrawdownPct   float64   `json:"drawdown_pct"`
	PositionCount int       `json:"position_count"`
}

// TradeRecord is the normalized trade output for reports.
type TradeRecord struct {
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"`
	EntryTime   time.Time `json:"entry_time"`
	ExitTime    time.Time `json:"exit_time"`
	EntryPrice  float64   `json:"entry_price"`
	ExitPrice   float64   `json:"exit_price"`
	Quantity    float64   `json:"quantity"`
	Leverage    int       `json:"leverage"`
	RealizedPnL float64   `json:"realized_pnl"`
	Fee         float64   `json:"fee"`
	HoldMinutes int       `json:"hold_minutes"`
}

// Summary is the top-level KPI output for a backtest run.
type Summary struct {
	Status         RunStatus    `json:"status"`
	StartTime      time.Time    `json:"start_time"`
	EndTime        time.Time    `json:"end_time"`
	InitialCapital float64      `json:"initial_capital"`
	EndingEquity   float64      `json:"ending_equity"`
	TotalReturnPct float64      `json:"total_return_pct"`
	MaxDrawdownPct float64      `json:"max_drawdown_pct"`
	TotalTrades    int          `json:"total_trades"`
	WinningTrades  int          `json:"winning_trades"`
	WinRatePct     float64      `json:"win_rate_pct"`
	TotalFees      float64      `json:"total_fees"`
	DecisionMode   DecisionMode `json:"decision_mode"`
	Notes          []string     `json:"notes,omitempty"`
}

// Result is the full output payload for a single run.
type Result struct {
	RunID       string          `json:"run_id"`
	Config      Config          `json:"config"`
	Summary     Summary         `json:"summary"`
	EquityCurve []EquityPoint   `json:"equity_curve"`
	Trades      []TradeRecord   `json:"trades"`
	Events      []BacktestEvent `json:"events,omitempty"`
}
