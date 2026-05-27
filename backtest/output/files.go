package backtest

import "context"

// ResultWriter persists one backtest result.
type ResultWriter interface {
	Write(ctx context.Context, result Result) error
}
