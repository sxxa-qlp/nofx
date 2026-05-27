package backtest

import (
	"context"
	"testing"
	"time"
)

func TestCoinankLoaderSmoke(t *testing.T) {
	loader := NewCoinankLoader()
	end := time.Now().UTC()
	start := end.Add(-6 * time.Hour)
	candles, err := loader.LoadCandles(context.Background(), "BTCUSDT", "15m", start, end)
	if err != nil {
		t.Fatalf("load candles failed: %v", err)
	}
	if len(candles) == 0 {
		t.Fatalf("expected candles, got 0")
	}
}
