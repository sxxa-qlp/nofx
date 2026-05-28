package quant

import "nofx/market"

type FactorBreakdown struct {
	TrendLongScore     float64 `json:"trend_long_score"`
	TrendShortScore    float64 `json:"trend_short_score"`
	MomentumLongScore  float64 `json:"momentum_long_score"`
	MomentumShortScore float64 `json:"momentum_short_score"`
	FlowLongScore      float64 `json:"flow_long_score"`
	FlowShortScore     float64 `json:"flow_short_score"`
	RiskPenalty        float64 `json:"risk_penalty"`
}

type RiskBudget struct {
	MaxPositionPct   float64 `json:"max_position_pct"`
	MaxLeverage      int     `json:"max_leverage"`
	AllowNewPosition bool    `json:"allow_new_position"`
}

type ExecutionHints struct {
	EntryStyle        string  `json:"entry_style"`
	StopLossHintPct   float64 `json:"stop_loss_hint_pct"`
	TakeProfitHintPct float64 `json:"take_profit_hint_pct"`
}

type Signal struct {
	Symbol          string             `json:"symbol"`
	Regime          market.RegimeLevel `json:"regime"`
	LongScore       float64            `json:"long_score"`
	ShortScore      float64            `json:"short_score"`
	Confidence      float64            `json:"confidence"`
	ActionBias      string             `json:"action_bias"`
	NoTrade         bool               `json:"no_trade"`
	FactorBreakdown FactorBreakdown    `json:"factor_breakdown"`
	RiskBudget      RiskBudget         `json:"risk_budget"`
	ExecutionHints  ExecutionHints     `json:"execution_hints"`
}
