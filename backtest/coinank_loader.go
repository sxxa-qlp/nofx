package backtest

import (
	"context"
	"fmt"
	"nofx/provider/coinank"
	"nofx/provider/coinank/coinank_api"
	"nofx/provider/coinank/coinank_enum"
	"strings"
	"time"
)

// CoinankLoader loads historical candles from CoinAnk open endpoints.
type CoinankLoader struct{}

func NewCoinankLoader() *CoinankLoader { return &CoinankLoader{} }

func (l *CoinankLoader) LoadCandles(ctx context.Context, symbol string, timeframe string, start time.Time, end time.Time) ([]Candle, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if !end.After(start) {
		return nil, fmt.Errorf("end must be after start")
	}
	interval, err := mapInterval(timeframe)
	if err != nil {
		return nil, err
	}
	exchange := mapExchange(symbol)
	size := estimateBars(timeframe, start, end)
	if size < 10 {
		size = 10
	}
	if size > 2000 {
		size = 2000
	}
	rows, err := coinank_api.Kline(ctx, strings.ToUpper(symbol), exchange, end.UnixMilli(), coinank_enum.To, size, interval)
	if err != nil {
		return nil, fmt.Errorf("coinank load candles: %w", err)
	}
	out := make([]Candle, 0, len(rows))
	for _, row := range rows {
		openTime := time.UnixMilli(row.StartTime).UTC()
		closeTime := time.UnixMilli(row.EndTime).UTC()
		if closeTime.Before(start) || openTime.After(end) {
			continue
		}
		out = append(out, Candle{Symbol: strings.ToUpper(symbol), Timeframe: timeframe, OpenTime: openTime, CloseTime: closeTime, Open: row.Open, High: row.High, Low: row.Low, Close: row.Close, Volume: row.Volume})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no candles returned for %s %s", symbol, timeframe)
	}
	return out, nil
}

func mapInterval(tf string) (coinank_enum.Interval, error) {
	switch tf {
	case "1m":
		return coinank_enum.Minute1, nil
	case "3m":
		return coinank_enum.Minute3, nil
	case "5m":
		return coinank_enum.Minute5, nil
	case "15m":
		return coinank_enum.Minute15, nil
	case "30m":
		return coinank_enum.Minute30, nil
	case "1h":
		return coinank_enum.Hour1, nil
	case "2h":
		return coinank_enum.Hour2, nil
	case "4h":
		return coinank_enum.Hour4, nil
	case "6h":
		return coinank_enum.Hour6, nil
	case "8h":
		return coinank_enum.Hour8, nil
	case "12h":
		return coinank_enum.Hour12, nil
	case "1d":
		return coinank_enum.Day1, nil
	default:
		return "", fmt.Errorf("unsupported timeframe: %s", tf)
	}
}

func mapExchange(symbol string) coinank_enum.Exchange {
	_ = symbol
	return coinank_enum.Binance
}

func estimateBars(tf string, start, end time.Time) int {
	delta := end.Sub(start)
	minutes := delta.Minutes()
	barMinutes := 15.0
	switch tf {
	case "1m":
		barMinutes = 1
	case "3m":
		barMinutes = 3
	case "5m":
		barMinutes = 5
	case "15m":
		barMinutes = 15
	case "30m":
		barMinutes = 30
	case "1h":
		barMinutes = 60
	case "2h":
		barMinutes = 120
	case "4h":
		barMinutes = 240
	case "6h":
		barMinutes = 360
	case "8h":
		barMinutes = 480
	case "12h":
		barMinutes = 720
	case "1d":
		barMinutes = 1440
	}
	return int(minutes/barMinutes) + 20
}

var _ DataSource = (*CoinankLoader)(nil)
var _ = coinank.KlineResult{}
