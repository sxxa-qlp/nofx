package backtest

import (
	"fmt"
	"strings"
	"time"
)

type SimPosition struct {
	Symbol     string
	Side       string
	EntryPrice float64
	Quantity   float64
	Leverage   int
	EntryTime  time.Time
}

type SimExecutionEngine struct {
	InitialCapital float64
	Cash           float64
	FeeRate        float64
	Positions      map[string]*SimPosition
	Trades         []TradeRecord
	PeakEquity     float64
	LastEquity     float64
}

func NewSimExecutionEngine(initialCapital, feeRate float64) *SimExecutionEngine {
	if initialCapital <= 0 {
		initialCapital = 1000
	}
	if feeRate < 0 {
		feeRate = 0
	}
	return &SimExecutionEngine{
		InitialCapital: initialCapital,
		Cash:           initialCapital,
		FeeRate:        feeRate,
		Positions:      make(map[string]*SimPosition),
		PeakEquity:     initialCapital,
		LastEquity:     initialCapital,
	}
}

func simPosKey(symbol, side string) string {
	return strings.ToUpper(symbol) + "|" + strings.ToLower(side)
}

func (e *SimExecutionEngine) MarkToMarket(price float64) (equity float64, unrealized float64, positionCount int) {
	equity = e.Cash
	for _, pos := range e.Positions {
		if pos.Side == "short" {
			unrealized += (pos.EntryPrice - price) * pos.Quantity
		} else {
			unrealized += (price - pos.EntryPrice) * pos.Quantity
		}
	}
	equity += unrealized
	positionCount = len(e.Positions)
	if equity > e.PeakEquity {
		e.PeakEquity = equity
	}
	e.LastEquity = equity
	return
}

func (e *SimExecutionEngine) drawdownPct() float64 {
	if e.PeakEquity <= 0 {
		return 0
	}
	dd := (e.PeakEquity - e.LastEquity) / e.PeakEquity * 100
	if dd < 0 {
		return 0
	}
	return dd
}

func (e *SimExecutionEngine) Apply(decision map[string]any, decisionTime time.Time, nextOpen float64) ([]TradeRecord, error) {
	if decision == nil {
		return nil, nil
	}
	raw, ok := decision["decisions"]
	if !ok {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, nil
	}
	var emitted []TradeRecord
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		action, _ := m["action"].(string)
		symbol, _ := m["symbol"].(string)
		if symbol == "" {
			continue
		}
		switch action {
		case "open_long", "open_short":
			if nextOpen <= 0 {
				continue
			}
			positionUSD := asFloat(m["position_size_usd"])
			if positionUSD <= 0 {
				continue
			}
			lev := int(asFloat(m["leverage"]))
			if lev <= 0 {
				lev = 1
			}
			qty := positionUSD / nextOpen
			fee := positionUSD * e.FeeRate
			if e.Cash < fee {
				continue
			}
			e.Cash -= fee
			side := "long"
			if action == "open_short" {
				side = "short"
			}
			e.Positions[simPosKey(symbol, side)] = &SimPosition{Symbol: symbol, Side: side, EntryPrice: nextOpen, Quantity: qty, Leverage: lev, EntryTime: decisionTime}
		case "close_long", "close_short":
			side := "long"
			if action == "close_short" {
				side = "short"
			}
			key := simPosKey(symbol, side)
			pos := e.Positions[key]
			if pos == nil || nextOpen <= 0 {
				continue
			}
			notional := pos.Quantity * nextOpen
			fee := notional * e.FeeRate
			realized := 0.0
			if pos.Side == "short" {
				realized = (pos.EntryPrice - nextOpen) * pos.Quantity
			} else {
				realized = (nextOpen - pos.EntryPrice) * pos.Quantity
			}
			e.Cash += realized - fee
			trade := TradeRecord{Symbol: pos.Symbol, Side: pos.Side, EntryTime: pos.EntryTime, ExitTime: decisionTime, EntryPrice: pos.EntryPrice, ExitPrice: nextOpen, Quantity: pos.Quantity, Leverage: pos.Leverage, RealizedPnL: realized, Fee: fee, HoldMinutes: int(decisionTime.Sub(pos.EntryTime).Minutes())}
			e.Trades = append(e.Trades, trade)
			emitted = append(emitted, trade)
			delete(e.Positions, key)
		}
	}
	return emitted, nil
}

func asFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return 0
	}
}

func nextBarOpen(series []Candle, decisionTime time.Time) (float64, error) {
	for _, c := range series {
		if c.OpenTime.After(decisionTime) {
			return c.Open, nil
		}
	}
	return 0, fmt.Errorf("next bar open not found")
}
