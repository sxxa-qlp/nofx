import { useMemo, useState } from 'react'
import useSWR from 'swr'
import { useSearchParams } from 'react-router-dom'
import { api } from '../lib/api'
import { Input } from '../components/ui/input'
import { NofxSelect } from '../components/ui/select'
import type { TraderInfo } from '../types'
import type { BacktestResult, BacktestRunRequest } from '../types/backtest'

const tfOptions = [
  { value: '15m', label: '15m' },
  { value: '1h', label: '1h' },
  { value: '4h', label: '4h' },
]

function isoDateTimeLocal(d: Date) {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export default function BacktestPage() {
  const [searchParams] = useSearchParams()
  const traderIdFromQuery = searchParams.get('trader_id') || ''
  const strategyIdFromQuery = searchParams.get('strategy_id') || ''

  const { data: traders } = useSWR<TraderInfo[]>('backtest-traders', api.getTraders)
  const { data: history, mutate: mutateHistory } = useSWR<BacktestResult[]>('backtest-history', api.getBacktestHistory, {
    shouldRetryOnError: false,
  })

  const now = useMemo(() => new Date(), [])
  const defaultStart = useMemo(() => new Date(now.getTime() - 7 * 24 * 3600 * 1000), [now])

  const [form, setForm] = useState<BacktestRunRequest>({
    trader_id: traderIdFromQuery,
    strategy_id: strategyIdFromQuery,
    symbol: 'BTCUSDT',
    start_time: new Date(defaultStart).toISOString(),
    end_time: new Date(now).toISOString(),
    decision_timeframe: '15m',
    replay_timeframes: ['15m', '1h', '4h'],
    initial_capital: 1000,
    taker_fee_rate: 0.0004,
    slippage_bps: 2,
    max_cycles: 50,
  })

  const [isRunning, setIsRunning] = useState(false)
  const [result, setResult] = useState<BacktestResult | null>(null)
  const [error, setError] = useState<string>('')

  const selectedTrader = traders?.find(t => t.trader_id === form.trader_id)

  const onRun = async () => {
    setError('')
    setIsRunning(true)
    try {
      const res = await api.runBacktest(form)
      setResult(res)
      await mutateHistory()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to run backtest')
    } finally {
      setIsRunning(false)
    }
  }

  return (
    <div className="min-h-screen px-4 md:px-8 py-6" style={{ background: '#0B0E11', color: '#EAECEF' }}>
      <div className="max-w-7xl mx-auto space-y-6">
        <div className="nofx-glass p-6 rounded-lg border border-white/5">
          <div className="flex items-start justify-between gap-4 flex-wrap">
            <div>
              <h1 className="text-2xl font-bold text-nofx-text-main">Backtest / 回测实验室</h1>
              <p className="text-sm text-nofx-text-muted mt-1">MVP: historical loading + replay dry run first, then strategy and execution.</p>
            </div>
            <div className="text-xs font-mono text-nofx-text-muted">{selectedTrader ? `Trader: ${selectedTrader.trader_name}` : 'No trader selected'}</div>
          </div>
        </div>

        <div className="grid grid-cols-1 xl:grid-cols-[380px_minmax(0,1fr)] gap-6">
          <div className="nofx-glass p-6 rounded-lg border border-white/5 space-y-4 h-fit">
            <h2 className="text-lg font-semibold">Run Form</h2>

            <div>
              <label className="block text-xs text-nofx-text-muted mb-2">Trader</label>
              <div className="bg-black/30 border border-white/10 rounded px-3 py-2 text-sm">
                <NofxSelect
                  value={form.trader_id || ''}
                  onChange={(value) => setForm(prev => ({ ...prev, trader_id: value }))}
                  options={(traders || []).map(t => ({ value: t.trader_id, label: t.trader_name }))}
                />
              </div>
            </div>

            <div>
              <label className="block text-xs text-nofx-text-muted mb-2">Symbol</label>
              <Input value={form.symbol || ''} onChange={(e) => setForm(prev => ({ ...prev, symbol: e.target.value }))} />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-nofx-text-muted mb-2">Decision TF</label>
                <div className="bg-black/30 border border-white/10 rounded px-3 py-2 text-sm">
                  <NofxSelect
                    value={form.decision_timeframe}
                    onChange={(value) => setForm(prev => ({ ...prev, decision_timeframe: value }))}
                    options={tfOptions}
                  />
                </div>
              </div>
              <div>
                <label className="block text-xs text-nofx-text-muted mb-2">Max Cycles</label>
                <Input type="number" value={form.max_cycles || 0} onChange={(e) => setForm(prev => ({ ...prev, max_cycles: Number(e.target.value) }))} />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-nofx-text-muted mb-2">Initial Capital</label>
                <Input type="number" value={form.initial_capital} onChange={(e) => setForm(prev => ({ ...prev, initial_capital: Number(e.target.value) }))} />
              </div>
              <div>
                <label className="block text-xs text-nofx-text-muted mb-2">Slippage (bps)</label>
                <Input type="number" value={form.slippage_bps} onChange={(e) => setForm(prev => ({ ...prev, slippage_bps: Number(e.target.value) }))} />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-nofx-text-muted mb-2">Start</label>
                <Input
                  type="datetime-local"
                  value={isoDateTimeLocal(new Date(form.start_time))}
                  onChange={(e) => setForm(prev => ({ ...prev, start_time: new Date(e.target.value).toISOString() }))}
                />
              </div>
              <div>
                <label className="block text-xs text-nofx-text-muted mb-2">End</label>
                <Input
                  type="datetime-local"
                  value={isoDateTimeLocal(new Date(form.end_time))}
                  onChange={(e) => setForm(prev => ({ ...prev, end_time: new Date(e.target.value).toISOString() }))}
                />
              </div>
            </div>

            <button
              onClick={onRun}
              disabled={isRunning}
              className="w-full px-4 py-3 rounded-lg font-semibold transition-all bg-[#F0B90B] text-black hover:brightness-110 disabled:opacity-50"
            >
              {isRunning ? 'Running…' : 'Run Backtest'}
            </button>

            {error ? <div className="text-sm text-red-400">{error}</div> : null}
          </div>

          <div className="space-y-6 min-w-0">
            <div className="nofx-glass p-6 rounded-lg border border-white/5">
              <h2 className="text-lg font-semibold mb-4">Summary</h2>
              {result ? (
                <div className="grid grid-cols-2 md:grid-cols-3 gap-4 text-sm">
                  <Metric label="Status" value={result.summary.status} />
                  <Metric label="Return" value={`${result.summary.total_return_pct.toFixed(2)}%`} />
                  <Metric label="Max DD" value={`${result.summary.max_drawdown_pct.toFixed(2)}%`} />
                  <Metric label="Trades" value={String(result.summary.total_trades)} />
                  <Metric label="Ending Equity" value={result.summary.ending_equity.toFixed(2)} />
                  <Metric label="Run ID" value={result.run_id} mono />
                </div>
              ) : (
                <div className="text-sm text-nofx-text-muted">No result yet.</div>
              )}
            </div>

            <div className="nofx-glass p-6 rounded-lg border border-white/5">
              <h2 className="text-lg font-semibold mb-4">Replay / Decisions (dry run)</h2>
              <div className="space-y-3 max-h-[420px] overflow-y-auto pr-2">
                {(result?.events || []).map((event, idx) => (
                  <div key={idx} className="rounded border border-white/5 bg-black/20 p-3 text-xs font-mono">
                    <div className="flex justify-between gap-3 flex-wrap">
                      <span className="text-[#F0B90B]">{event.type}</span>
                      <span className="text-nofx-text-muted">{event.time}</span>
                    </div>
                    <div className="mt-2 text-nofx-text-main">{event.message}</div>
                    {event.payload?.timeframes ? (
                      <div className="mt-3 grid grid-cols-1 md:grid-cols-3 gap-2">
                        {Object.entries(event.payload.timeframes).map(([tf, value]) => {
                          const v = value as any
                          return (
                            <div key={tf} className="rounded bg-white/5 p-2">
                              <div className="text-[#F0B90B] mb-1">{tf}</div>
                              <div>bars: {v.bars}</div>
                              <div>last: {v.last_price}</div>
                              <div className="text-nofx-text-muted">{String(v.last_close)}</div>
                            </div>
                          )
                        })}
                      </div>
                    ) : null}
                  </div>
                ))}
                {!result?.events?.length ? <div className="text-sm text-nofx-text-muted">No replay events yet.</div> : null}
              </div>
            </div>

            <div className="nofx-glass p-6 rounded-lg border border-white/5">
              <h2 className="text-lg font-semibold mb-4">Run History</h2>
              <div className="space-y-3">
                {(history || []).map((item) => (
                  <div key={item.run_id} className="rounded border border-white/5 bg-black/20 p-3 text-sm flex items-center justify-between gap-4">
                    <div>
                      <div className="font-mono text-[#F0B90B]">{item.run_id}</div>
                      <div className="text-nofx-text-muted text-xs">{item.summary.notes?.[0] || item.summary.status}</div>
                    </div>
                    <div className="text-right">
                      <div>{item.summary.total_return_pct.toFixed(2)}%</div>
                      <div className="text-xs text-nofx-text-muted">DD {item.summary.max_drawdown_pct.toFixed(2)}%</div>
                    </div>
                  </div>
                ))}
                {!history?.length ? <div className="text-sm text-nofx-text-muted">No history yet.</div> : null}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function Metric({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="rounded border border-white/5 bg-black/20 p-3">
      <div className="text-xs text-nofx-text-muted mb-1">{label}</div>
      <div className={`${mono ? 'font-mono' : ''} text-nofx-text-main font-semibold break-all`}>{value}</div>
    </div>
  )
}
