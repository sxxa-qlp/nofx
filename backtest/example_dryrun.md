# Backtest first dry run example

Target shape for first dry run:
- symbol: BTCUSDT
- decision timeframe: 15m
- replay timeframes: 15m / 1h / 4h
- purpose: validate historical loading + replay timing only

Expected output:
- generated result JSON in configured output dir
- cycle events with per-timeframe window summaries
- flat equity curve (no execution yet)
