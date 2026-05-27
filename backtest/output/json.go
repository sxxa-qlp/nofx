package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// JSONFileWriter writes a single summary JSON file for MVP.
type JSONFileWriter struct {
	Dir string
}

func (w JSONFileWriter) Write(ctx context.Context, result Result) error {
	_ = ctx
	if w.Dir == "" {
		return nil
	}
	if err := os.MkdirAll(w.Dir, 0o755); err != nil {
		return fmt.Errorf("create backtest output dir: %w", err)
	}
	path := filepath.Join(w.Dir, result.RunID+".json")
	buf, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal backtest result: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("write backtest result: %w", err)
	}
	return nil
}
