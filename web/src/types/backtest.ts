export interface BacktestRunRequest {
  trader_id?: string
  strategy_id?: string
  symbol?: string
  start_time: string
  end_time: string
  decision_timeframe: string
  replay_timeframes: string[]
  initial_capital: number
  taker_fee_rate: number
  slippage_bps: number
  max_cycles?: number
  run_real_ai_all_cycles?: boolean
  use_extended_quant_data?: boolean
}

export interface BacktestSummary {
  status: string
  start_time: string
  end_time: string
  initial_capital: number
  ending_equity: number
  total_return_pct: number
  max_drawdown_pct: number
  total_trades: number
  winning_trades: number
  win_rate_pct: number
  total_fees: number
  notes?: string[]
}

export interface BacktestEvent {
  time: string
  type: string
  symbol?: string
  message?: string
  payload?: Record<string, any>
}

export interface BacktestResult {
  run_id: string
  summary: BacktestSummary
  events: BacktestEvent[]
  equity_curve: Array<{
    time: string
    equity: number
    available: number
    unrealized_pnl: number
    realized_pnl: number
    drawdown_pct: number
    position_count: number
  }>
  trades: Array<Record<string, any>>
}
