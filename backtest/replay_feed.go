package backtest

import (
	"fmt"
	"sort"
	"time"
)

// ReplayFeed advances a decision clock over one primary timeframe series.
type ReplayFeed struct {
	Symbol    string
	Timeframe string
	Candles   []Candle
	idx       int
}

func NewReplayFeed(symbol string, timeframe string, candles []Candle) (*ReplayFeed, error) {
	if len(candles) == 0 {
		return nil, fmt.Errorf("replay feed requires candles")
	}
	copied := append([]Candle(nil), candles...)
	sort.Slice(copied, func(i, j int) bool {
		return copied[i].OpenTime.Before(copied[j].OpenTime)
	})
	return &ReplayFeed{Symbol: symbol, Timeframe: timeframe, Candles: copied}, nil
}

func (f *ReplayFeed) HasNext() bool {
	return f != nil && f.idx < len(f.Candles)
}

func (f *ReplayFeed) Current() (Candle, bool) {
	if !f.HasNext() {
		return Candle{}, false
	}
	return f.Candles[f.idx], true
}

func (f *ReplayFeed) Advance() (Candle, bool) {
	if !f.HasNext() {
		return Candle{}, false
	}
	c := f.Candles[f.idx]
	f.idx++
	return c, true
}

func (f *ReplayFeed) DecisionTime() (time.Time, bool) {
	c, ok := f.Current()
	if !ok {
		return time.Time{}, false
	}
	return c.CloseTime.UTC(), true
}

func (f *ReplayFeed) Window(n int) []Candle {
	if n <= 0 || f.idx <= 0 {
		return nil
	}
	end := f.idx
	start := end - n
	if start < 0 {
		start = 0
	}
	return append([]Candle(nil), f.Candles[start:end]...)
}
