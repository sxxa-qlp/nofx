package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"nofx/backtest"
	"nofx/store"
	"time"

	"github.com/gin-gonic/gin"
)

const backtestResultDir = "data/backtests"

type backtestRequest struct {
	TraderID             string   `json:"trader_id"`
	StrategyID           string   `json:"strategy_id"`
	Symbol               string   `json:"symbol"`
	StartTime            string   `json:"start_time"`
	EndTime              string   `json:"end_time"`
	DecisionTF           string   `json:"decision_timeframe"`
	ReplayTimeframes     []string `json:"replay_timeframes"`
	InitialCapital       float64  `json:"initial_capital"`
	TakerFeeRate         float64  `json:"taker_fee_rate"`
	SlippageBps          float64  `json:"slippage_bps"`
	MaxCycles            int      `json:"max_cycles"`
	RunRealAIAllCycles   bool     `json:"run_real_ai_all_cycles"`
	UseExtendedQuantData bool     `json:"use_extended_quant_data"`
}

func (s *Server) handleRunBacktest(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req backtestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		SafeBadRequest(c, "Invalid backtest request")
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		SafeBadRequest(c, "Invalid start_time")
		return
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		SafeBadRequest(c, "Invalid end_time")
		return
	}

	decisionTF := backtest.BarTimeframe(req.DecisionTF)
	if decisionTF == "" {
		decisionTF = backtest.Timeframe15m
	}

	symbol := defaultIfEmpty(req.Symbol, "BTCUSDT")
	cfg := backtest.Config{
		Name:           "mvp-dry-run",
		TraderID:       req.TraderID,
		StrategyID:     req.StrategyID,
		UserID:         userID,
		Exchange:       "binance",
		Symbols:        []string{symbol},
		InitialCapital: defaultFloat(req.InitialCapital, 1000),
		StartTime:      startTime.UTC(),
		EndTime:        endTime.UTC(),
		DecisionTF:     decisionTF,
		FillMode:       backtest.FillNextBarOpen,
		DecisionMode:   backtest.DecisionModeRules,
		TakerFeeRate:   req.TakerFeeRate,
		SlippageBps:    req.SlippageBps,
		MaxCycles:      req.MaxCycles,
		OutputDir:      backtestResultDir,
	}

	var decisionGenerator backtest.DecisionGenerator
	if req.TraderID != "" {
		fullCfg, ferr := s.store.Trader().GetFullConfig(userID, req.TraderID)
		if ferr == nil && fullCfg != nil && fullCfg.Strategy != nil {
			var strategyCfg store.StrategyConfig
			if err := json.Unmarshal([]byte(fullCfg.Strategy.Config), &strategyCfg); err == nil {
				if strategyCfg.Indicators.Klines.PrimaryTimeframe != "" {
					cfg.DecisionTF = backtest.BarTimeframe(strategyCfg.Indicators.Klines.PrimaryTimeframe)
				}
				cfg.ReplayTimeframes = pickReplayTimeframes(&strategyCfg, req.ReplayTimeframes)
				if req.UseExtendedQuantData {
					ensureQuantFeatures(&strategyCfg)
				}
				decisionGenerator = newAPIDecisionGenerator(&strategyCfg, buildAIClientFromModel(fullCfg.AIModel), true, req.RunRealAIAllCycles, req.UseExtendedQuantData)
			}
		}
	}
	if len(cfg.ReplayTimeframes) == 0 {
		cfg.ReplayTimeframes = defaultTimeframes(req.ReplayTimeframes)
	}

	history := backtest.NewHistoryStore(backtestResultDir)
	runner := backtest.NewRunner(backtest.RunnerDependencies{
		DataSource:        backtest.NewCoinankLoader(),
		Writer:            history,
		DecisionGenerator: decisionGenerator,
	})

	result, err := runner.Run(context.Background(), cfg)
	if err != nil {
		SafeInternalError(c, "Failed to run backtest", err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) handleListBacktests(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	items, err := backtest.NewHistoryStore(backtestResultDir).List(context.Background())
	if err != nil {
		SafeInternalError(c, "Failed to list backtests", err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleGetBacktest(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	runID := c.Param("id")
	if runID == "" {
		SafeBadRequest(c, "Missing backtest id")
		return
	}
	item, err := backtest.NewHistoryStore(backtestResultDir).Get(context.Background(), runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Backtest %s not found", runID)})
		return
	}
	c.JSON(http.StatusOK, item)
}

func defaultIfEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func defaultFloat(v, fallback float64) float64 {
	if v <= 0 {
		return fallback
	}
	return v
}

func defaultTimeframes(v []string) []string {
	if len(v) == 0 {
		return []string{"15m", "1h", "4h"}
	}
	return v
}

func ensureQuantFeatures(cfg *store.StrategyConfig) {
	if cfg == nil {
		return
	}
	if cfg.Indicators.NofxOSAPIKey == "" {
		cfg.Indicators.NofxOSAPIKey = store.GetDefaultStrategyConfig("zh").Indicators.NofxOSAPIKey
	}
	cfg.Indicators.EnableQuantData = true
	cfg.Indicators.EnableQuantOI = true
	cfg.Indicators.EnableQuantNetflow = true
	cfg.Indicators.EnableOIRanking = true
	cfg.Indicators.EnableNetFlowRanking = true
	cfg.Indicators.EnablePriceRanking = true
	if cfg.Indicators.OIRankingDuration == "" {
		cfg.Indicators.OIRankingDuration = "1h"
	}
	if cfg.Indicators.NetFlowRankingDuration == "" {
		cfg.Indicators.NetFlowRankingDuration = "1h"
	}
	if cfg.Indicators.PriceRankingDuration == "" {
		cfg.Indicators.PriceRankingDuration = "1h,4h,24h"
	}
	if cfg.Indicators.OIRankingLimit <= 0 {
		cfg.Indicators.OIRankingLimit = 10
	}
	if cfg.Indicators.NetFlowRankingLimit <= 0 {
		cfg.Indicators.NetFlowRankingLimit = 10
	}
	if cfg.Indicators.PriceRankingLimit <= 0 {
		cfg.Indicators.PriceRankingLimit = 10
	}
}

func pickReplayTimeframes(cfg *store.StrategyConfig, fallback []string) []string {
	if cfg != nil && len(cfg.Indicators.Klines.SelectedTimeframes) > 0 {
		return cfg.Indicators.Klines.SelectedTimeframes
	}
	if len(fallback) > 0 {
		return fallback
	}
	return []string{"15m", "1h", "4h"}
}
