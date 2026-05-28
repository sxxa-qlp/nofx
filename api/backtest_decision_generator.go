package api

import (
	"context"
	"fmt"
	"nofx/backtest"
	"nofx/kernel"
	"nofx/market"
	"nofx/mcp"
	"nofx/quant"
	"nofx/store"
	"strings"
	"time"
)

type apiBacktestDecisionGenerator struct {
	engine      *kernel.StrategyEngine
	aiClient    mcp.AIClient
	runRealAI   bool
	maxAICycles int
	variant     string
}

func newAPIDecisionGenerator(strategyCfg *store.StrategyConfig, aiClient mcp.AIClient, runRealAI bool, runAllCycles bool) *apiBacktestDecisionGenerator {
	engine := kernel.NewStrategyEngine(strategyCfg)
	maxCycles := 1
	if runAllCycles {
		maxCycles = 1000000
	}
	return &apiBacktestDecisionGenerator{
		engine:      engine,
		aiClient:    aiClient,
		runRealAI:   runRealAI,
		maxAICycles: maxCycles,
		variant:     "balanced",
	}
}

func (g *apiBacktestDecisionGenerator) Generate(ctx context.Context, in backtest.DecisionInput) (map[string]any, error) {
	if g == nil || g.engine == nil {
		return nil, nil
	}
	cfg := g.engine.GetConfig()
	primaryTF := cfg.Indicators.Klines.PrimaryTimeframe
	if primaryTF == "" {
		primaryTF = string(in.Config.DecisionTF)
	}
	primaryWindow := in.Windows[primaryTF]
	if len(primaryWindow) == 0 {
		for _, series := range in.Windows {
			if len(series) > 0 {
				primaryWindow = series
				break
			}
		}
	}
	if len(primaryWindow) == 0 {
		return nil, fmt.Errorf("no primary window available")
	}
	longerTF := cfg.Indicators.Klines.LongerTimeframe
	if longerTF == "" {
		if _, ok := in.Windows["4h"]; ok {
			longerTF = "4h"
		} else if _, ok := in.Windows["1h"]; ok {
			longerTF = "1h"
		}
	}
	longerWindow := in.Windows[longerTF]

	mkt, err := market.BuildDataFromKlines(in.Symbol, toMarketKlines(primaryWindow), toMarketKlines(longerWindow))
	if err != nil {
		return nil, err
	}
	quantDataMap := g.engine.FetchQuantDataBatch([]string{in.Symbol})
	oiRankingData := g.engine.FetchOIRankingData()
	netFlowRankingData := g.engine.FetchNetFlowRankingData()
	priceRankingData := g.engine.FetchPriceRankingData()
	ctxObj := &kernel.Context{
		CurrentTime:    in.DecisionTime.UTC().Format("2006-01-02 15:04:05 UTC"),
		RuntimeMinutes: 0,
		CallCount:      in.Cycle,
		Account: kernel.AccountInfo{
			TotalEquity:      in.Config.InitialCapital,
			AvailableBalance: in.Config.InitialCapital,
			UnrealizedPnL:    0,
			TotalPnL:         0,
			TotalPnLPct:      0,
			MarginUsed:       0,
			MarginUsedPct:    0,
			PositionCount:    0,
		},
		Positions:      []kernel.PositionInfo{},
		CandidateCoins: []kernel.CandidateCoin{{Symbol: in.Symbol, Sources: []string{"backtest"}}},
		PromptVariant:  g.variant,
		MarketDataMap: map[string]*market.Data{
			in.Symbol: mkt,
		},
		OITopDataMap:       make(map[string]*kernel.OITopData),
		QuantDataMap:       quantDataMap,
		OIRankingData:      oiRankingData,
		NetFlowRankingData: netFlowRankingData,
		PriceRankingData:   priceRankingData,
	}

	signal := quant.BuildSignal(in.Symbol, mkt, quantDataMap[in.Symbol], oiRankingData, netFlowRankingData, priceRankingData)
	payload := map[string]any{
		"mode":            "prompt_preview",
		"symbol":          in.Symbol,
		"cycle":           in.Cycle,
		"candidate_count": 1,
		"quant_signal":    signal,
		"strategy_quant_flags": map[string]any{
			"enable_quant_data":      cfg.Indicators.EnableQuantData,
			"enable_quant_oi":        cfg.Indicators.EnableQuantOI,
			"enable_quant_netflow":   cfg.Indicators.EnableQuantNetflow,
			"enable_oi_ranking":      cfg.Indicators.EnableOIRanking,
			"enable_netflow_ranking": cfg.Indicators.EnableNetFlowRanking,
			"enable_price_ranking":   cfg.Indicators.EnablePriceRanking,
			"nofxos_api_key_present": cfg.Indicators.NofxOSAPIKey != "",
		},
		"quant_input": map[string]any{
			"has_quant_data":      quantDataMap[in.Symbol] != nil,
			"has_oi_ranking":      oiRankingData != nil,
			"has_netflow_ranking": netFlowRankingData != nil,
			"has_price_ranking":   priceRankingData != nil,
		},
	}

	systemPrompt := g.engine.BuildSystemPrompt(in.Config.InitialCapital, g.variant)
	userPrompt := g.engine.BuildUserPrompt(ctxObj)
	userPrompt += "\n\n## Quant Signal Layer\n"
	userPrompt += fmt.Sprintf("Regime: %s\n", signal.Regime)
	userPrompt += fmt.Sprintf("Long Score: %.4f | Short Score: %.4f | Confidence: %.4f | Bias: %s | NoTrade: %v\n", signal.LongScore, signal.ShortScore, signal.Confidence, signal.ActionBias, signal.NoTrade)
	userPrompt += fmt.Sprintf("Factor Breakdown: trend_long=%.4f trend_short=%.4f momentum_long=%.4f momentum_short=%.4f flow_long=%.4f flow_short=%.4f risk_penalty=%.4f\n", signal.FactorBreakdown.TrendLongScore, signal.FactorBreakdown.TrendShortScore, signal.FactorBreakdown.MomentumLongScore, signal.FactorBreakdown.MomentumShortScore, signal.FactorBreakdown.FlowLongScore, signal.FactorBreakdown.FlowShortScore, signal.FactorBreakdown.RiskPenalty)
	userPrompt += fmt.Sprintf("Risk Budget: max_position_pct=%.4f max_leverage=%d allow_new_position=%v\n", signal.RiskBudget.MaxPositionPct, signal.RiskBudget.MaxLeverage, signal.RiskBudget.AllowNewPosition)
	userPrompt += "Treat the quant signal as the primary bias layer. Only override it when you have a strong reason, and explain the override explicitly.\n"
	payload["system_prompt_preview"] = truncate(systemPrompt, 400)
	payload["user_prompt_preview"] = truncate(userPrompt, 400)
	payload["system_prompt_length"] = len(systemPrompt)
	payload["user_prompt_length"] = len(userPrompt)

	if g.runRealAI && g.aiClient != nil && in.Cycle <= g.maxAICycles {
		fd, err := kernel.GetFullDecisionWithStrategy(ctxObj, g.aiClient, g.engine, g.variant)
		if err != nil {
			payload["mode"] = "ai_error"
			payload["ai_error"] = err.Error()
			return payload, nil
		}
		payload["mode"] = "ai_decision"
		payload["ai_request_duration_ms"] = fd.AIRequestDurationMs
		payload["cot_preview"] = truncate(fd.CoTTrace, 400)
		payload["raw_response_preview"] = truncate(fd.RawResponse, 400)
		payload["decisions"] = fd.Decisions
	} else if g.runRealAI {
		payload["mode"] = "prompt_only_after_ai_sample_limit"
		payload["note"] = fmt.Sprintf("Real AI execution is limited to the first %d cycle(s) in MVP to control cost.", g.maxAICycles)
	}

	return payload, nil
}

func toMarketKlines(window []backtest.Candle) []market.Kline {
	out := make([]market.Kline, 0, len(window))
	for _, c := range window {
		out = append(out, market.Kline{
			OpenTime:  c.OpenTime.UnixMilli(),
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
			Volume:    c.Volume,
			CloseTime: c.CloseTime.UnixMilli(),
		})
	}
	return out
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func buildAIClientFromModel(model *store.AIModel) mcp.AIClient {
	if model == nil || !model.Enabled || model.APIKey == "" {
		return nil
	}
	provider := model.Provider
	apiKey := string(model.APIKey)
	aiClient := mcp.NewAIClientByProvider(provider)
	if aiClient == nil {
		aiClient = mcp.NewClient()
	}
	switch provider {
	case "claw402":
		aiClient.SetAPIKey(apiKey, "", model.CustomModelName)
	default:
		aiClient.SetAPIKey(apiKey, model.CustomAPIURL, model.CustomModelName)
	}
	aiClient.SetTimeout(90 * time.Second)
	return aiClient
}
