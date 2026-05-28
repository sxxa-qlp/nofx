package backtest

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// RunnerDependencies will be filled in phase-by-phase as the MVP gets wired.
type RunnerDependencies struct {
	DataSource        DataSource
	Writer            ResultWriter
	DecisionGenerator DecisionGenerator
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

	symbol := firstSymbol(cfg)
	if symbol == "" {
		return nil, fmt.Errorf("at least one symbol is required for dry run")
	}

	replayTFs := normalizeReplayTimeframes(cfg)
	windowSizes := normalizeWindowSizes(cfg, replayTFs)

	loaded := make(map[string][]Candle, len(replayTFs))
	for _, tf := range replayTFs {
		candles, err := r.deps.DataSource.LoadCandles(ctx, symbol, tf, cfg.StartTime, cfg.EndTime)
		if err != nil {
			return nil, fmt.Errorf("load candles %s %s: %w", symbol, tf, err)
		}
		loaded[tf] = candles
	}

	primaryTF := string(cfg.DecisionTF)
	primary, ok := loaded[primaryTF]
	if !ok || len(primary) == 0 {
		return nil, fmt.Errorf("primary timeframe %s has no candles", primaryTF)
	}

	feed, err := NewReplayFeed(symbol, primaryTF, primary)
	if err != nil {
		return nil, err
	}

	execEngine := NewSimExecutionEngine(cfg.InitialCapital, cfg.TakerFeeRate)
	cycles := 0
	for feed.HasNext() {
		current, _ := feed.Current()
		decisionTime := current.CloseTime.UTC()
		if decisionTime.Before(cfg.StartTime) || decisionTime.After(cfg.EndTime) {
			feed.Advance()
			continue
		}

		cyclePayload := map[string]any{
			"decision_time": decisionTime,
			"timeframes":    map[string]any{},
		}

		for _, tf := range replayTFs {
			window := closedWindowAt(loaded[tf], decisionTime, windowSizes[tf])
			if len(window) == 0 {
				continue
			}
			last := window[len(window)-1]
			cyclePayload["timeframes"].(map[string]any)[tf] = map[string]any{
				"bars":       len(window),
				"first_open": window[0].OpenTime,
				"last_close": last.CloseTime,
				"last_price": last.Close,
			}
		}

		result.Events = append(result.Events, BacktestEvent{
			Time:    decisionTime,
			Type:    EventCycleStart,
			Symbol:  symbol,
			Message: fmt.Sprintf("dry-run cycle %d", cycles+1),
			Payload: cyclePayload,
		})

		var decisionPayload map[string]any
		if r.deps.DecisionGenerator != nil {
			decisionPayload, derr := r.deps.DecisionGenerator.Generate(ctx, DecisionInput{
				Config:       cfg,
				Cycle:        cycles + 1,
				Symbol:       symbol,
				DecisionTime: decisionTime,
				Windows:      collectWindows(loaded, decisionTime, windowSizes),
			})
			if derr != nil {
				result.Events = append(result.Events, BacktestEvent{
					Time:    decisionTime,
					Type:    EventDecision,
					Symbol:  symbol,
					Message: fmt.Sprintf("decision generation failed: %v", derr),
					Payload: map[string]any{"error": derr.Error()},
				})
			} else if decisionPayload != nil {
				if qs, ok := decisionPayload["quant_signal"]; ok {
					result.Events = append(result.Events, BacktestEvent{
						Time:    decisionTime,
						Type:    EventQuantSignal,
						Symbol:  symbol,
						Message: fmt.Sprintf("quant signal cycle %d", cycles+1),
						Payload: map[string]any{"quant_signal": qs},
					})
				}
				result.Events = append(result.Events, BacktestEvent{
					Time:    decisionTime,
					Type:    EventDecision,
					Symbol:  symbol,
					Message: fmt.Sprintf("decision cycle %d", cycles+1),
					Payload: decisionPayload,
				})
			}
		}

		if decisionPayload != nil {
			if nextOpen, nerr := nextBarOpen(primary, decisionTime); nerr == nil {
				if trades, terr := execEngine.Apply(decisionPayload, decisionTime, nextOpen); terr == nil {
					for _, tr := range trades {
						result.Trades = append(result.Trades, tr)
						result.Events = append(result.Events, BacktestEvent{Time: decisionTime, Type: EventOrderFilled, Symbol: tr.Symbol, Message: fmt.Sprintf("simulated close %s", tr.Side), Payload: map[string]any{"trade": tr}})
					}
				}
			}
		}
		nextPrice := current.Close
		if eq, upnl, posCount := execEngine.MarkToMarket(nextPrice); true {
			result.EquityCurve = append(result.EquityCurve, EquityPoint{
				Time:          decisionTime,
				Equity:        eq,
				Available:     execEngine.Cash,
				UnrealizedPnL: upnl,
				RealizedPnL:   execEngine.Cash - execEngine.InitialCapital,
				DrawdownPct:   execEngine.drawdownPct(),
				PositionCount: posCount,
			})
		}

		cycles++
		feed.Advance()
		if cfg.MaxCycles > 0 && cycles >= cfg.MaxCycles {
			break
		}
	}

	result.Summary.Status = RunStatusCompleted
	result.Summary.EndingEquity = execEngine.LastEquity
	if cfg.InitialCapital > 0 {
		result.Summary.TotalReturnPct = (execEngine.LastEquity - cfg.InitialCapital) / cfg.InitialCapital * 100
	}
	result.Summary.MaxDrawdownPct = execEngine.drawdownPct()
	result.Summary.TotalTrades = len(result.Trades)
	wins := 0
	totalFees := 0.0
	for _, tr := range result.Trades {
		if tr.RealizedPnL > 0 {
			wins++
		}
		totalFees += tr.Fee
	}
	result.Summary.WinningTrades = wins
	if len(result.Trades) > 0 {
		result.Summary.WinRatePct = float64(wins) / float64(len(result.Trades)) * 100
	}
	result.Summary.TotalFees = totalFees
	result.Summary.Notes = append(result.Summary.Notes,
		fmt.Sprintf("Backtest run completed for %s.", symbol),
		fmt.Sprintf("Loaded timeframes: %v", replayTFs),
		fmt.Sprintf("Replay cycles processed: %d", cycles),
		fmt.Sprintf("Simulated execution enabled; generated %d trades.", len(result.Trades)),
	)

	if r.deps.Writer != nil {
		if err := r.deps.Writer.Write(ctx, *result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func firstSymbol(cfg Config) string {
	if len(cfg.Symbols) == 0 {
		return ""
	}
	return cfg.Symbols[0]
}

func normalizeReplayTimeframes(cfg Config) []string {
	if len(cfg.ReplayTimeframes) > 0 {
		seen := map[string]bool{}
		out := make([]string, 0, len(cfg.ReplayTimeframes))
		for _, tf := range cfg.ReplayTimeframes {
			if tf == "" || seen[tf] {
				continue
			}
			seen[tf] = true
			out = append(out, tf)
		}
		sort.Strings(out)
		return out
	}
	return []string{string(cfg.DecisionTF), "1h", "4h"}
}

func normalizeWindowSizes(cfg Config, timeframes []string) map[string]int {
	out := map[string]int{}
	for _, tf := range timeframes {
		if cfg.WindowSizes != nil {
			if n, ok := cfg.WindowSizes[tf]; ok && n > 0 {
				out[tf] = n
				continue
			}
		}
		switch tf {
		case "15m":
			out[tf] = 20
		case "1h":
			out[tf] = 20
		case "4h":
			out[tf] = 20
		default:
			out[tf] = 20
		}
	}
	return out
}

func collectWindows(all map[string][]Candle, decisionTime time.Time, windowSizes map[string]int) map[string][]Candle {
	out := make(map[string][]Candle, len(all))
	for tf, series := range all {
		out[tf] = closedWindowAt(series, decisionTime, windowSizes[tf])
	}
	return out
}

func closedWindowAt(series []Candle, decisionTime time.Time, maxBars int) []Candle {
	if len(series) == 0 || maxBars <= 0 {
		return nil
	}
	eligible := make([]Candle, 0, maxBars)
	for _, c := range series {
		if !c.CloseTime.After(decisionTime) {
			eligible = append(eligible, c)
		}
	}
	if len(eligible) == 0 {
		return nil
	}
	if len(eligible) > maxBars {
		eligible = eligible[len(eligible)-maxBars:]
	}
	return eligible
}
