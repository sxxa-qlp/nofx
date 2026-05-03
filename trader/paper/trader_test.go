package paper

import (
	"math"
	"testing"
	"time"
)

func fixedPrice(prices map[string]float64) PriceProvider {
	return func(symbol string) (float64, error) {
		return prices[symbol], nil
	}
}

func TestPaperTraderOpenLongAndUnrealizedPnL(t *testing.T) {
	prices := map[string]float64{"BTCUSDT": 100.0}
	pt := NewPaperTrader(10000, WithPriceProvider(fixedPrice(prices)), WithTakerFeeRate(0))

	if _, err := pt.OpenLong("BTCUSDT", 1, 10); err != nil {
		t.Fatalf("OpenLong failed: %v", err)
	}

	prices["BTCUSDT"] = 110
	positions, err := pt.GetPositions()
	if err != nil {
		t.Fatalf("GetPositions failed: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(positions))
	}
	if got := positions[0]["unRealizedProfit"].(float64); math.Abs(got-10) > 1e-9 {
		t.Fatalf("expected unrealized pnl 10, got %v", got)
	}

	balance, err := pt.GetBalance()
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}
	if got := balance["totalEquity"].(float64); math.Abs(got-10010) > 1e-9 {
		t.Fatalf("expected equity 10010, got %v", got)
	}
}

func TestPaperTraderCloseLongRealizesPnL(t *testing.T) {
	prices := map[string]float64{"ETHUSDT": 100.0}
	pt := NewPaperTrader(10000, WithPriceProvider(fixedPrice(prices)), WithTakerFeeRate(0))

	if _, err := pt.OpenLong("ETHUSDT", 2, 5); err != nil {
		t.Fatalf("OpenLong failed: %v", err)
	}
	prices["ETHUSDT"] = 125
	res, err := pt.CloseLong("ETHUSDT", 0)
	if err != nil {
		t.Fatalf("CloseLong failed: %v", err)
	}
	if got := res["realizedPnl"].(float64); math.Abs(got-50) > 1e-9 {
		t.Fatalf("expected realized pnl 50, got %v", got)
	}

	positions, err := pt.GetPositions()
	if err != nil {
		t.Fatalf("GetPositions failed: %v", err)
	}
	if len(positions) != 0 {
		t.Fatalf("expected no positions, got %d", len(positions))
	}

	balance, err := pt.GetBalance()
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}
	if got := balance["totalEquity"].(float64); math.Abs(got-10050) > 1e-9 {
		t.Fatalf("expected equity 10050, got %v", got)
	}

	closed, err := pt.GetClosedPnL(time.Time{}, 10)
	if err != nil {
		t.Fatalf("GetClosedPnL failed: %v", err)
	}
	if len(closed) != 1 || math.Abs(closed[0].RealizedPnL-50) > 1e-9 {
		t.Fatalf("unexpected closed pnl records: %+v", closed)
	}
}

func TestPaperTraderOpenShortAndClose(t *testing.T) {
	prices := map[string]float64{"BTCUSDT": 100.0}
	pt := NewPaperTrader(10000, WithPriceProvider(fixedPrice(prices)), WithTakerFeeRate(0))

	if _, err := pt.OpenShort("BTCUSDT", 3, 10); err != nil {
		t.Fatalf("OpenShort failed: %v", err)
	}
	prices["BTCUSDT"] = 90
	positions, err := pt.GetPositions()
	if err != nil {
		t.Fatalf("GetPositions failed: %v", err)
	}
	if got := positions[0]["unRealizedProfit"].(float64); math.Abs(got-30) > 1e-9 {
		t.Fatalf("expected short unrealized pnl 30, got %v", got)
	}
	if _, err := pt.CloseShort("BTCUSDT", 1); err != nil {
		t.Fatalf("partial CloseShort failed: %v", err)
	}
	positions, err = pt.GetPositions()
	if err != nil {
		t.Fatalf("GetPositions failed: %v", err)
	}
	if got := math.Abs(positions[0]["positionAmt"].(float64)); math.Abs(got-2) > 1e-9 {
		t.Fatalf("expected remaining qty 2, got %v", got)
	}
}
