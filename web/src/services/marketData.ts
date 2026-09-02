export type MarketInterval = '15m' | '1h' | '4h' | '1d'
export type MarketSource = 'demo' | 'api' | 'ave' | 'embed'

export interface Candle {
  time: number
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export interface MarketSnapshot {
  pair: string
  interval: MarketInterval
  candles: Candle[]
  source: MarketSource
}

export interface MarketDataProvider {
  getCandles(pair: string, interval: MarketInterval): Promise<MarketSnapshot>
}

const KLINE_API = String(import.meta.env.VITE_KLINE_API || '').trim()
const KLINE_EMBED_URL = String(import.meta.env.VITE_KLINE_EMBED_URL || '').trim()
const KLINE_PAIR = String(import.meta.env.VITE_KLINE_PAIR || 'AIX-WIN').trim()
const API_BASE = String(import.meta.env.VITE_API || '').replace(/\/+$/, '')

const intervalMs: Record<MarketInterval, number> = {
  '15m': 15 * 60 * 1000,
  '1h': 60 * 60 * 1000,
  '4h': 4 * 60 * 60 * 1000,
  '1d': 24 * 60 * 60 * 1000,
}

const seedByInterval: Record<MarketInterval, number> = { '15m': 11, '1h': 23, '4h': 37, '1d': 53 }

export const klineEmbedUrl = KLINE_EMBED_URL
export const klinePair = KLINE_PAIR
export const hasKlineApi = Boolean(KLINE_API)
export const hasKlineEmbed = Boolean(KLINE_EMBED_URL)

export function resolveKlineEmbedUrl(interval: MarketInterval): string {
  if (!KLINE_EMBED_URL) return ''
  const joiner = KLINE_EMBED_URL.indexOf('?') >= 0 ? '&' : '?'
  return `${KLINE_EMBED_URL}${joiner}pair=${encodeURIComponent(KLINE_PAIR)}&interval=${encodeURIComponent(interval)}`
}

function toMs(value: number): number {
  if (!Number.isFinite(value) || value <= 0) return 0
  return value < 1e12 ? value * 1000 : value
}

function asCandle(time: number, open: number, high: number, low: number, close: number, volume: number): Candle | null {
  const stamp = toMs(time)
  if (!stamp || ![open, high, low, close].every(Number.isFinite)) return null
  return {
    time: stamp,
    open,
    high: Math.max(high, open, close),
    low: Math.min(low, open, close),
    close,
    volume: Number.isFinite(volume) ? volume : 0,
  }
}

function fromObject(row: Record<string, unknown>): Candle | null {
  return asCandle(
    Number(row.time ?? row.timestamp ?? row.t ?? row.openTime ?? row.open_time),
    Number(row.open ?? row.o),
    Number(row.high ?? row.h),
    Number(row.low ?? row.l),
    Number(row.close ?? row.c),
    Number(row.volume ?? row.vol ?? row.v ?? 0),
  )
}

function fromArray(row: unknown[]): Candle | null {
  if (row.length < 5) return null
  return asCandle(Number(row[0]), Number(row[1]), Number(row[2]), Number(row[3]), Number(row[4]), Number(row[5] ?? 0))
}

function collectRows(payload: unknown): unknown[] {
  if (Array.isArray(payload)) return payload
  if (!payload || typeof payload !== 'object') return []
  const row = payload as Record<string, unknown>
  const nested = row.data && typeof row.data === 'object' && !Array.isArray(row.data)
    ? row.data as Record<string, unknown>
    : null
  const candidates = [
    row.candles, row.klines, row.list, row.rows, row.items, row.points,
    nested && (nested.candles || nested.klines || nested.list || nested.rows || nested.points),
  ]
  for (let i = 0; i < candidates.length; i += 1) {
    if (Array.isArray(candidates[i])) return candidates[i] as unknown[]
  }
  return []
}

export function normalizeKlinePayload(payload: unknown): Candle[] {
  const candles: Candle[] = []
  const rows = collectRows(payload)
  for (let i = 0; i < rows.length; i += 1) {
    const row = rows[i]
    const candle = Array.isArray(row)
      ? fromArray(row)
      : row && typeof row === 'object'
        ? fromObject(row as Record<string, unknown>)
        : null
    if (candle) candles.push(candle)
  }
  candles.sort((a, b) => a.time - b.time)
  return candles
}

export const mapAveCandles = (rows: Array<Record<string, unknown>>): Candle[] => normalizeKlinePayload(rows)

function createDemoCandles(interval: MarketInterval, count = 120): Candle[] {
  const step = intervalMs[interval]
  const end = Math.floor(Date.now() / step) * step
  let previousClose = 1.728 + seedByInterval[interval] * 0.0007

  return Array.from({ length: count }, (_, index) => {
    const phase = index + seedByInterval[interval]
    const drift = Math.sin(phase * 0.39) * 0.021 + Math.cos(phase * 0.17) * 0.012 + 0.003
    const open = previousClose
    const close = Math.max(0.6, open + drift)
    const wick = 0.012 + Math.abs(Math.sin(phase * 0.71)) * 0.025
    const candle = {
      time: end - (count - 1 - index) * step,
      open,
      high: Math.max(open, close) + wick,
      low: Math.min(open, close) - wick * 0.82,
      close,
      volume: 520000 + Math.abs(Math.cos(phase * 0.31)) * 1080000 + index * 3900,
    }
    previousClose = close
    return candle
  })
}

function resolveApiUrl(interval: MarketInterval): string {
  const joiner = KLINE_API.indexOf('?') >= 0 ? '&' : '?'
  const query = `pair=${encodeURIComponent(KLINE_PAIR)}&interval=${encodeURIComponent(interval)}&limit=300`
  if (/^https?:\/\//i.test(KLINE_API)) return `${KLINE_API}${joiner}${query}`
  const path = KLINE_API.charAt(0) === '/' ? KLINE_API : `/${KLINE_API}`
  return `${API_BASE}${path}${joiner}${query}`
}

function sourceFromApi(): MarketSource {
  return /ave/i.test(KLINE_API) ? 'ave' : 'api'
}

const demoProvider: MarketDataProvider = {
  async getCandles(pair, interval) {
    return { pair, interval, candles: createDemoCandles(interval), source: 'demo' }
  },
}

function sourceFromPayload(payload: unknown): MarketSource {
  if (payload && typeof payload === 'object') {
    const source = String((payload as Record<string, unknown>).source || '').trim().toLowerCase()
    if (source === 'ave') return 'ave'
  }
  return sourceFromApi()
}

const remoteProvider: MarketDataProvider = {
  async getCandles(pair, interval) {
    const response = await fetch(resolveApiUrl(interval), { headers: { Accept: 'application/json' } })
    if (!response.ok) throw new Error(`kline ${response.status}`)
    const payload = await response.json()
    const candles = normalizeKlinePayload(payload)
    if (!candles.length) throw new Error('empty kline')
    return { pair, interval, candles, source: sourceFromPayload(payload) }
  },
}

const activeProvider: MarketDataProvider = KLINE_API ? remoteProvider : demoProvider

export const getAixWinCandles = (interval: MarketInterval) => activeProvider.getCandles(KLINE_PAIR, interval)
