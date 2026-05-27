package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HistoryStore persists and reads backtest results from disk for the MVP.
type HistoryStore struct {
	Dir string
}

func NewHistoryStore(dir string) *HistoryStore {
	return &HistoryStore{Dir: dir}
}

func (s *HistoryStore) Save(ctx context.Context, result Result) error {
	_ = ctx
	if s.Dir == "" {
		return fmt.Errorf("history store dir is empty")
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	buf, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Dir, result.RunID+".json"), buf, 0o644)
}

// Write implements ResultWriter so the history store can be used directly by Runner.
func (s *HistoryStore) Write(ctx context.Context, result Result) error {
	return s.Save(ctx, result)
}

func (s *HistoryStore) Get(ctx context.Context, runID string) (*Result, error) {
	_ = ctx
	buf, err := os.ReadFile(filepath.Join(s.Dir, runID+".json"))
	if err != nil {
		return nil, err
	}
	var out Result
	if err := json.Unmarshal(buf, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *HistoryStore) List(ctx context.Context) ([]Result, error) {
	_ = ctx
	if s.Dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Result{}, nil
		}
		return nil, err
	}
	results := make([]Result, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		buf, err := os.ReadFile(filepath.Join(s.Dir, entry.Name()))
		if err != nil {
			continue
		}
		var item Result
		if err := json.Unmarshal(buf, &item); err != nil {
			continue
		}
		results = append(results, item)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].RunID > results[j].RunID
	})
	return results, nil
}
