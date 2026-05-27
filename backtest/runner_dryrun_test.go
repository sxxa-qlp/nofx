package backtest

import (
	"context"
	"testing"
	"time"
)

type stubDataSource struct {
	data map[string][]Candle
}

func (s stubDataSource) LoadCandles(ctx context.Context, symbol string, timeframe string, start time.Time, end time.Time) ([]Candle, error) {
	_ = ctx
	return append([]Candle(nil), s.data[timeframe]...), nil
}

func TestRunnerDryRun(t *testing.T) {
	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	mk := func(tf string, step time.Duration, n int) []Candle {
		out := make([]Candle, 0, n)
		for i := 0; i < n; i++ {
			o := base.Add(time.Duration(i) * step)
			c := o.Add(step)
			out = append(out, Candle{Symbol: "BTCUSDT", Timeframe: tf, OpenTime: o, CloseTime: c, Open: 100 + float64(i), High: 101 + float64(i), Low: 99 + float64(i), Close: 100.5 + float64(i), Volume: 1000})
		}
		return out
	}

	runner := NewRunner(RunnerDependencies{
		DataSource: stubDataSource{data: map[string][]Candle{
			"15m": mk("15m", 15*time.Minute, 12),
			"1h":  mk("1h", time.Hour, 12),
			"4h":  mk("4h", 4*time.Hour, 12),
		}},
	})

	res, err := runner.Run(context.Background(), Config{
		Name:             "dryrun",
		Symbols:          []string{"BTCUSDT"},
		InitialCapital:   1000,
		StartTime:        base,
		EndTime:          base.Add(24 * time.Hour),
		DecisionTF:       Timeframe15m,
		FillMode:         FillNextBarOpen,
		DecisionMode:     DecisionModeRules,
		ReplayTimeframes: []string{"15m", "1h", "4h"},
		MaxCycles:        3,
	})
	if err != nil {
		t.Fatalf("run dry run failed: %v", err)
	}
	if res == nil {
		t.Fatalf("expected result")
	}
	if len(res.Events) != 3 {
		t.Fatalf("expected 3 cycle events, got %d", len(res.Events))
	}
	if len(res.EquityCurve) != 3 {
		t.Fatalf("expected 3 equity points, got %d", len(res.EquityCurve))
	}
}
