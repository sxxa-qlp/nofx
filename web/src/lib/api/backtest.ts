import type { BacktestRunRequest, BacktestResult } from '../../types/backtest'
import { API_BASE, httpClient } from './helpers'

export const backtestApi = {
  async runBacktest(request: BacktestRunRequest): Promise<BacktestResult> {
    const result = await httpClient.post<BacktestResult>(`${API_BASE}/backtests`, request)
    if (!result.success) throw new Error(result.message || 'Failed to run backtest')
    return result.data!
  },

  async getBacktestHistory(): Promise<BacktestResult[]> {
    const result = await httpClient.get<BacktestResult[]>(`${API_BASE}/backtests`)
    if (!result.success) throw new Error(result.message || 'Failed to fetch backtest history')
    return Array.isArray(result.data) ? result.data : []
  },

  async getBacktest(runId: string): Promise<BacktestResult> {
    const result = await httpClient.get<BacktestResult>(`${API_BASE}/backtests/${runId}`)
    if (!result.success) throw new Error(result.message || 'Failed to fetch backtest result')
    return result.data!
  },
}
