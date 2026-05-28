package api

import (
	"context"
	"fmt"
	"nofx/backtest"
	"nofx/kernel"
	"nofx/market"
	"nofx/mcp"
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

func newAPIDecisionGenerator(strategyCfg *store.StrategyConfig, aiClient mcp.AIClient, runRealAI bool) *apiBacktestDecisionGenerator {
	engine := kernel.NewStrategyEngine(strategyCfg)
	return &apiBacktestDecisionGenerator{
		engine:      engine,
		aiClient:    aiClient,
		runRealAI:   runRealAI,
		maxAICycles: 1,
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
		OITopDataMap: make(map[string]*kernel.OITopData),
		QuantDataMap: make(map[string]*kernel.QuantData),
	}

	payload := map[string]any{
		"mode":            "prompt_preview",
		"symbol":          in.Symbol,
		"cycle":           in.Cycle,
		"candidate_count": 1,
	}

	systemPrompt := g.engine.BuildSystemPrompt(in.Config.InitialCapital, g.variant)
	userPrompt := g.engine.BuildUserPrompt(ctxObj)
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
