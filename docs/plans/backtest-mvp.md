# NOFX Backtest MVP Plan

## Goal
Build a minimal backtesting mode that reuses the existing NOFX strategy engine and paper trader logic, but replaces:
- realtime market/exchange inputs
- real order execution
with:
- historical data replay
- simulated order execution

The MVP is for strategy validation and tuning, not perfect exchange-matching.

---

## MVP Scope

### Included
1. Historical replay using real Binance Futures market data
2. Bar-by-bar simulation (decision on bar close, execution on next bar open)
3. Reuse existing strategy engine / AI decision flow as much as possible
4. Use existing `trader/paper` exchange as the execution backend where possible
5. Support core metrics:
   - total return
   - max drawdown
   - win rate
   - pnl by trade
   - fee-adjusted pnl
   - equity curve
6. Produce a machine-readable result file + human summary

### Not in MVP
1. Partial take profit / partial stop loss
2. Intra-bar tick simulation
3. Order book simulation
4. Funding fee precision beyond a simple periodic model
5. Multi-run parameter sweep UI
6. AI response caching optimization (can be phase 2)

---

## Design Principles

1. **Reuse > rewrite**
   - Reuse current strategy engine and decision schema
   - Reuse paper trader for fills/accounting where possible

2. **No future leak**
   - Only closed bars available at decision time
   - Order fills occur on next bar open in MVP

3. **Deterministic first**
   - Same input/config should produce same result

4. **Minimal intrusion**
   - Add new backtest modules instead of rewriting live trading paths

---

## Proposed Architecture

### New packages

#### `backtest/`
Core runner and orchestration.

Suggested files:
- `backtest/config.go` — backtest config schema
- `backtest/runner.go` — main replay loop
- `backtest/report.go` — result aggregation and summary
- `backtest/types.go` — request/result/event structs

#### `backtest/data/`
Historical data loading and replay.

Suggested files:
- `backtest/data/source.go` — source interface
- `backtest/data/binance_loader.go` — load historical klines/funding/OI
- `backtest/data/replay_feed.go` — time advancement / replay cursor

#### `backtest/output/`
Persistence for runs.

Suggested files:
- `backtest/output/json.go`
- `backtest/output/files.go`

---

## Reuse Points in Existing Code

### 1. Strategy engine
Keep using current strategy decision flow wherever possible.
Relevant areas:
- `kernel/`
- `trader/auto_trader_loop.go`
- `trader/auto_trader_decision.go`
- `mcp/` AI client flow

### 2. Paper trader
Existing local simulated exchange already exists:
- `trader/paper/trader.go`

This should be the MVP execution backend.
Needed MVP adjustment may be:
- inject replay prices rather than realtime mark prices
- ensure fills happen from replayed OHLC logic

### 3. Trader config / strategy config
Reuse current trader/strategy config structures so backtest config can reference:
- strategy id
- ai model id
- exchange type assumptions
- leverage rules

---

## MVP Execution Logic

For each decision timestamp `T`:

1. Load historical bars up to and including the latest fully closed bar at `T`
2. Build the same multi-timeframe context the live strategy uses
3. Build account snapshot from paper trader state
4. Run existing strategy/AI decision logic
5. Translate decision to simulated orders
6. Execute fills at **next bar open**
7. Update virtual account and positions
8. Record:
   - decision
   - fill
   - equity
   - drawdown
   - fees

---

## Data Requirements for MVP

### Required
- Klines:
  - 15m
  - 1h
  - 4h
- Symbol metadata / trading rules
- Fee assumptions

### Optional in MVP if easily available
- funding rate
- open interest

If OI/funding are difficult to wire immediately, allow them to be disabled in MVP.

---

## Assumptions for MVP

1. Entry/exit fill at next bar open
2. Flat fee model (configurable)
3. Fixed slippage per trade (configurable)
4. Only full open / full close in MVP
5. One deterministic replay timeline

---

## Deliverables

### Deliverable 1: Backtest config
A config payload/file that defines:
- trader / strategy reference
- date range
- symbols / candidate pool
- initial capital
- fee/slippage assumptions
- whether to use real AI calls

### Deliverable 2: Runnable backtest entry
Either:
- CLI subcommand, or
- internal runner callable from API / admin route

Preferred MVP path: **internal runner first**, CLI/API wrapper second.

### Deliverable 3: Result artifacts
Per run output:
- `summary.json`
- `trades.json`
- `equity.csv`
- readable summary text

### Deliverable 4: Validation run
Run one known strategy over a short historical period and verify:
- completes end-to-end
- produces stable output
- no future leak obvious in timestamps

---

## Phased MVP Tasks

### Phase A — Skeleton
1. Add `backtest/` package
2. Define config/result structs
3. Add runner skeleton
4. Add result writer

### Phase B — Replay data
1. Implement historical kline loader
2. Implement replay clock/feed
3. Build multi-timeframe slices for strategy input

### Phase C — Strategy integration
1. Reuse current strategy decision entrypoint
2. Create backtest context adapter
3. Run strategy in replay loop

### Phase D — Execution simulation
1. Integrate paper trader or lightweight execution wrapper
2. Apply next-bar-open fills
3. Add fees/slippage
4. Track positions / equity

### Phase E — Reporting
1. Aggregate trades
2. Compute metrics
3. Export summary + artifacts

---

## Success Criteria

MVP is considered done when:
1. We can run a backtest over a chosen date range
2. It uses real historical Binance data
3. It reuses current NOFX strategy logic
4. It outputs deterministic results and core metrics
5. It is good enough to compare strategy versions and tune major parameters

---

## Phase 2 After MVP
1. Partial TP / partial SL
2. AI response cache
3. Parameter sweeps / grid search
4. Walk-forward testing
5. Funding/OI precision improvements
6. Better fill model
