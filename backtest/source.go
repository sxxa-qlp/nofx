package backtest

import (
	"context"
	"time"
)

// Candle is the normalized historical market bar for replay.
type Candle struct {
	Symbol    string    `json:"symbol"`
	Timeframe string    `json:"timeframe"`
	OpenTime  time.Time `json:"open_time"`
	CloseTime time.Time `json:"close_time"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

// DataSource defines the minimum historical data contract for MVP.
type DataSource interface {
	LoadCandles(ctx context.Context, symbol string, timeframe string, start time.Time, end time.Time) ([]Candle, error)
}
