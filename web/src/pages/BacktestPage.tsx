import { useMemo, useState } from 'react'
import useSWR from 'swr'
import { useSearchParams } from 'react-router-dom'
import { ResponsiveContainer, LineChart, Line, XAxis, YAxis, Tooltip, CartesianGrid } from 'recharts'
import { api } from '../lib/api'
import { Input } from '../components/ui/input'
import { NofxSelect } from '../components/ui/select'
import { useLanguage } from '../contexts/LanguageContext'
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
  const { language } = useLanguage()
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
    run_real_ai_all_cycles: false,
    use_extended_quant_data: false,
  })

  const [isRunning, setIsRunning] = useState(false)
  const [result, setResult] = useState<BacktestResult | null>(null)
  const [error, setError] = useState<string>('')
  const [selectedEventIndex, setSelectedEventIndex] = useState<number>(0)

  const selectedTrader = traders?.find(t => t.trader_id === form.trader_id)
  const zh = language === 'zh'
  const cycleEvents = (result?.events || []).filter((e) => e.type === 'cycle_start')
  const selectedCycleEvent = cycleEvents[selectedEventIndex] || cycleEvents[0]

  const onRun = async () => {
    setError('')
    setIsRunning(true)
    try {
      const res = await api.runBacktest(form)
      setResult(res)
      setSelectedEventIndex(0)
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
              <h1 className="text-2xl font-bold text-nofx-text-main">{zh ? '回测实验室 / Backtest Lab' : 'Backtest Lab / 回测实验室'}</h1>
              <p className="text-sm text-nofx-text-muted mt-1">{zh ? 'MVP：先验证历史数据加载与回放 dry run，再接策略执行。' : 'MVP: validate historical loading and replay dry run first, then wire strategy execution.'}</p>
            </div>
            <div className="text-xs font-mono text-nofx-text-muted">{selectedTrader ? `${zh ? '交易员' : 'Trader'}: ${selectedTrader.trader_name}` : (zh ? '未选择交易员 / No trader selected' : 'No trader selected / 未选择交易员')}</div>
          </div>
        </div>

        <div className="grid grid-cols-1 xl:grid-cols-[380px_minmax(0,1fr)] gap-6">
          <div className="nofx-glass p-6 rounded-lg border border-white/5 space-y-4 h-fit">
            <h2 className="text-lg font-semibold">{zh ? '运行表单 / Run Form' : 'Run Form / 运行表单'}</h2>

            <div>
              <label className="block text-xs text-nofx-text-muted mb-2">{zh ? '交易员 / Trader' : 'Trader / 交易员'}</label>
              <div className="bg-black/30 border border-white/10 rounded px-3 py-2 text-sm">
                <NofxSelect
                  value={form.trader_id || ''}
                  onChange={(value) => setForm(prev => ({ ...prev, trader_id: value }))}
                  options={(traders || []).map(t => ({ value: t.trader_id, label: t.trader_name }))}
                />
              </div>
            </div>

            <div>
              <label className="block text-xs text-nofx-text-muted mb-2">{zh ? '交易对 / Symbol' : 'Symbol / 交易对'}</label>
              <Input value={form.symbol || ''} onChange={(e) => setForm(prev => ({ ...prev, symbol: e.target.value }))} />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-nofx-text-muted mb-2">{zh ? '决策周期 / Decision TF' : 'Decision TF / 决策周期'}</label>
                <div className="bg-black/30 border border-white/10 rounded px-3 py-2 text-sm">
                  <NofxSelect
                    value={form.decision_timeframe}
                    onChange={(value) => setForm(prev => ({ ...prev, decision_timeframe: value }))}
                    options={tfOptions}
                  />
                </div>
              </div>
              <div>
                <label className="block text-xs text-nofx-text-muted mb-2">{zh ? '最大周期数 / Max Cycles' : 'Max Cycles / 最大周期数'}</label>
                <Input type="number" value={form.max_cycles || 0} onChange={(e) => setForm(prev => ({ ...prev, max_cycles: Number(e.target.value) }))} />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-nofx-text-muted mb-2">{zh ? '初始资金 / Initial Capital' : 'Initial Capital / 初始资金'}</label>
                <Input type="number" value={form.initial_capital} onChange={(e) => setForm(prev => ({ ...prev, initial_capital: Number(e.target.value) }))} />
              </div>
              <div>
                <label className="block text-xs text-nofx-text-muted mb-2">{zh ? '滑点 / Slippage (bps)' : 'Slippage (bps) / 滑点'}</label>
                <Input type="number" value={form.slippage_bps} onChange={(e) => setForm(prev => ({ ...prev, slippage_bps: Number(e.target.value) }))} />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-nofx-text-muted mb-2">{zh ? '开始时间 / Start' : 'Start / 开始时间'}</label>
                <Input
                  type="datetime-local"
                  value={isoDateTimeLocal(new Date(form.start_time))}
                  onChange={(e) => setForm(prev => ({ ...prev, start_time: new Date(e.target.value).toISOString() }))}
                />
              </div>
              <div>
                <label className="block text-xs text-nofx-text-muted mb-2">{zh ? '结束时间 / End' : 'End / 结束时间'}</label>
                <Input
                  type="datetime-local"
                  value={isoDateTimeLocal(new Date(form.end_time))}
                  onChange={(e) => setForm(prev => ({ ...prev, end_time: new Date(e.target.value).toISOString() }))}
                />
              </div>
            </div>

            <div className="space-y-2 rounded border border-white/10 bg-black/20 p-3 text-xs">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={!!form.run_real_ai_all_cycles}
                  onChange={(e) => setForm(prev => ({ ...prev, run_real_ai_all_cycles: e.target.checked }))}
                />
                <span>{zh ? '全周期真实 AI / Run real AI on all cycles' : 'Run real AI on all cycles / 全周期真实 AI'}</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={!!form.use_extended_quant_data}
                  onChange={(e) => setForm(prev => ({ ...prev, use_extended_quant_data: e.target.checked }))}
                />
                <span>{zh ? '使用扩展量化数据（nofxos）/ Use extended quant data' : 'Use extended quant data (nofxos) / 使用扩展量化数据'}</span>
              </label>
            </div>

            <button
              onClick={onRun}
              disabled={isRunning}
              className="w-full px-4 py-3 rounded-lg font-semibold transition-all bg-[#F0B90B] text-black hover:brightness-110 disabled:opacity-50"
            >
              {isRunning ? (zh ? '运行中… / Running…' : 'Running… / 运行中…') : (zh ? '运行回测 / Run Backtest' : 'Run Backtest / 运行回测')}
            </button>

            {error ? <div className="text-sm text-red-400">{error}</div> : null}
          </div>

          <div className="space-y-6 min-w-0">
            <div className="nofx-glass p-6 rounded-lg border border-white/5">
              <h2 className="text-lg font-semibold mb-4">{zh ? '结果摘要 / Summary' : 'Summary / 结果摘要'}</h2>
              {result ? (
                <div className="grid grid-cols-2 md:grid-cols-3 gap-4 text-sm">
                  <Metric label={zh ? '状态 / Status' : 'Status / 状态'} value={result.summary.status} />
                  <Metric label={zh ? '收益 / Return' : 'Return / 收益'} value={`${result.summary.total_return_pct.toFixed(2)}%`} />
                  <Metric label={zh ? '最大回撤 / Max DD' : 'Max DD / 最大回撤'} value={`${result.summary.max_drawdown_pct.toFixed(2)}%`} />
                  <Metric label={zh ? '交易数 / Trades' : 'Trades / 交易数'} value={String(result.summary.total_trades)} />
                  <Metric label={zh ? '结束权益 / Ending Equity' : 'Ending Equity / 结束权益'} value={result.summary.ending_equity.toFixed(2)} />
                  <Metric label="Run ID" value={result.run_id} mono />
                </div>
              ) : (
                <div className="text-sm text-nofx-text-muted">{zh ? '暂无结果 / No result yet.' : 'No result yet. / 暂无结果'}</div>
              )}
            </div>

            <div className="nofx-glass p-6 rounded-lg border border-white/5">
              <h2 className="text-lg font-semibold mb-4">{zh ? '回放 / 决策（dry run）' : 'Replay / Decisions (dry run) / 回放决策'}</h2>
              <div className="mb-4 rounded border border-white/10 bg-black/20 p-3">
                <div className="flex items-center justify-between gap-3 flex-wrap mb-3">
                  <div className="text-sm font-semibold text-nofx-text-main">{zh ? '输入 Bar 可视化 / Input Bar Visualization' : 'Input Bar Visualization / 输入 Bar 可视化'}</div>
                  {cycleEvents.length > 0 && (
                    <div className="bg-black/30 border border-white/10 rounded px-3 py-2 text-sm min-w-[180px]">
                      <NofxSelect
                        value={selectedEventIndex}
                        onChange={(value) => setSelectedEventIndex(Number(value))}
                        options={cycleEvents.map((e, idx) => ({ value: idx, label: zh ? `周期 ${idx + 1}` : `Cycle ${idx + 1}` }))}
                      />
                    </div>
                  )}
                </div>
                {selectedCycleEvent?.payload?.timeframes ? (
                  <div className="space-y-4">
                    {Object.entries(selectedCycleEvent.payload.timeframes).map(([tf, value]) => (
                      <TimeframeBarsCard key={tf} timeframe={tf} value={value as any} zh={zh} />
                    ))}
                  </div>
                ) : (
                  <div className="text-sm text-nofx-text-muted">{zh ? '暂无 bar 数据 / No bar data yet.' : 'No bar data yet. / 暂无 bar 数据'}</div>
                )}
              </div>
              <div className="space-y-3 max-h-[420px] overflow-y-auto pr-2">
                {(result?.events || []).map((event, idx) => (
                  <div key={idx} className="rounded border border-white/5 bg-black/20 p-3 text-xs font-mono">
                    <div className="flex justify-between gap-3 flex-wrap">
                      <span className="text-[#F0B90B]">{event.type}</span>
                      <span className="text-nofx-text-muted">{event.time}</span>
                    </div>
                    <div className="mt-2 text-nofx-text-main">{localizeEventMessage(event.message, zh)}</div>
                    {event.payload?.timeframes ? (
                      <div className="mt-3 grid grid-cols-1 md:grid-cols-3 gap-2">
                        {Object.entries(event.payload.timeframes).map(([tf, value]) => {
                          const v = value as any
                          return (
                            <div key={tf} className="rounded bg-white/5 p-2">
                              <div className="text-[#F0B90B] mb-1">{tf}</div>
                              <div>{zh ? `K线数: ${v.bars}` : `Bars: ${v.bars}`}</div>
                              <div>{zh ? `最新价: ${v.last_price}` : `Last: ${v.last_price}`}</div>
                              <div className="text-nofx-text-muted">{String(v.last_close)}</div>
                            </div>
                          )
                        })}
                      </div>
                    ) : null}
                    {event.payload?.quant_signal ? (
                      <div className="mt-3 rounded bg-white/5 p-3 text-xs">
                        <div className="font-semibold mb-2 text-[#F0B90B]">{zh ? '量化信号 / Quant Signal' : 'Quant Signal / 量化信号'}</div>
                        <div>{zh ? `市场状态: ${event.payload.quant_signal.regime}` : `Regime: ${event.payload.quant_signal.regime}`}</div>
                        <div>{zh ? `多头分数: ${Number(event.payload.quant_signal.long_score).toFixed(4)}` : `Long Score: ${Number(event.payload.quant_signal.long_score).toFixed(4)}`}</div>
                        <div>{zh ? `空头分数: ${Number(event.payload.quant_signal.short_score).toFixed(4)}` : `Short Score: ${Number(event.payload.quant_signal.short_score).toFixed(4)}`}</div>
                        <div>{zh ? `置信度: ${Number(event.payload.quant_signal.confidence).toFixed(4)}` : `Confidence: ${Number(event.payload.quant_signal.confidence).toFixed(4)}`}</div>
                        <div>{zh ? `偏向动作: ${event.payload.quant_signal.action_bias}` : `Action Bias: ${event.payload.quant_signal.action_bias}`}</div>
                      </div>
                    ) : null}
                  </div>
                ))}
                {!result?.events?.length ? <div className="text-sm text-nofx-text-muted">{zh ? '暂无回放事件 / No replay events yet.' : 'No replay events yet. / 暂无回放事件'}</div> : null}
              </div>
            </div>

            <div className="nofx-glass p-6 rounded-lg border border-white/5">
              <h2 className="text-lg font-semibold mb-4">{zh ? '运行历史 / Run History' : 'Run History / 运行历史'}</h2>
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
                {!history?.length ? <div className="text-sm text-nofx-text-muted">{zh ? '暂无历史 / No history yet.' : 'No history yet. / 暂无历史'}</div> : null}
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

function TimeframeBarsCard({ timeframe, value, zh }: { timeframe: string; value: any; zh: boolean }) {
  const candles = (value?.candles || []).map((c: any) => ({
    time: String(c.close_time).slice(11, 16),
    open: Number(c.open),
    high: Number(c.high),
    low: Number(c.low),
    close: Number(c.close),
    volume: Number(c.volume),
  }))

  return (
    <div className="rounded border border-white/10 bg-white/5 p-3">
      <div className="flex items-center justify-between gap-3 flex-wrap mb-3">
        <div className="text-[#F0B90B] font-semibold">{timeframe}</div>
        <div className="text-xs text-nofx-text-muted">{zh ? `K线数: ${value?.bars ?? 0} / 最新价: ${value?.last_price ?? '-'}` : `Bars: ${value?.bars ?? 0} / Last: ${value?.last_price ?? '-'}`}</div>
      </div>
      <div className="h-48 w-full mb-3">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={candles}>
            <CartesianGrid strokeDasharray="3 3" stroke="#2B3139" />
            <XAxis dataKey="time" stroke="#848E9C" />
            <YAxis stroke="#848E9C" domain={['auto', 'auto']} />
            <Tooltip />
            <Line type="monotone" dataKey="close" stroke="#F0B90B" dot={false} strokeWidth={2} />
          </LineChart>
        </ResponsiveContainer>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-xs font-mono">
          <thead>
            <tr className="text-nofx-text-muted border-b border-white/10">
              <th className="text-left py-1">Time</th>
              <th className="text-right py-1">Open</th>
              <th className="text-right py-1">High</th>
              <th className="text-right py-1">Low</th>
              <th className="text-right py-1">Close</th>
              <th className="text-right py-1">Vol</th>
            </tr>
          </thead>
          <tbody>
            {candles.map((c: any, idx: number) => (
              <tr key={idx} className="border-b border-white/5 last:border-0">
                <td className="py-1">{c.time}</td>
                <td className="py-1 text-right">{c.open.toFixed(2)}</td>
                <td className="py-1 text-right">{c.high.toFixed(2)}</td>
                <td className="py-1 text-right">{c.low.toFixed(2)}</td>
                <td className="py-1 text-right">{c.close.toFixed(2)}</td>
                <td className="py-1 text-right">{c.volume.toFixed(2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function localizeEventMessage(message: string | undefined, zh: boolean) {
  if (!message) return ''
  if (message.startsWith('dry-run cycle')) {
    const n = message.replace('dry-run cycle ', '')
    return zh ? `回放周期 ${n}` : `Dry-run cycle ${n}`
  }
  if (message.startsWith('quant signal cycle')) {
    const n = message.replace('quant signal cycle ', '')
    return zh ? `量化信号周期 ${n}` : `Quant signal cycle ${n}`
  }
  if (message.startsWith('decision cycle')) {
    const n = message.replace('decision cycle ', '')
    return zh ? `决策周期 ${n}` : `Decision cycle ${n}`
  }
  return message
}
