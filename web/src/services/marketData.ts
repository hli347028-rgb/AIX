export type MarketInterval = '15m' | '1h' | '4h' | '1d'
export type MarketSource = 'demo' | 'api' | 'aix_win' | 'embed'

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

/** 默认走本后端按 AIX/WIN 现价生成的 K 线；不再依赖 AVE。 */
const KLINE_API = String(import.meta.env.VITE_KLINE_API || '/v1/market/kline').trim()
const KLINE_EMBED_URL = String(import.meta.env.VITE_KLINE_EMBED_URL || '').trim()
const KLINE_PAIR = String(import.meta.env.VITE_KLINE_PAIR || 'AIX-WIN').trim()
const API_BASE = String(import.meta.env.VITE_API || '').replace(/\/+$/, '')

const intervalMs: Record<MarketInterval, number> = {
  '15m': 15 * 60 * 1000,
  '1h': 60 * 60 * 1000,
  '4h': 4 * 60 * 60 * 1000,
  '1d': 24 * 60 * 60 * 1000,
}

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

/** @deprecated 兼容旧引用；现已按 AIX/WIN 价格生成 */
export const mapAveCandles = (rows: Array<Record<string, unknown>>): Candle[] => normalizeKlinePayload(rows)

function priceWiggle(unixSec: number, minutes: number): number {
  const x = unixSec / (minutes * 60) + minutes
  return Math.sin(x * 0.37) * 0.0048 + Math.cos(x * 0.19) * 0.0026
}

/** 用 AIX/WIN 现价本地绘制蜡烛图（接口不可用时的兜底）。 */
export function createPriceCandles(interval: MarketInterval, spot: number, count = 120): Candle[] {
  const step = intervalMs[interval]
  const minutes = step / 60_000
  const end = Math.floor(Date.now() / step) * step
  let previousClose = 0
  const safeSpot = Number.isFinite(spot) && spot > 0 ? spot : 1

  return Array.from({ length: count }, (_, index) => {
    const time = end - (count - 1 - index) * step
    const wiggle = priceWiggle(Math.floor(time / 1000), minutes) * safeSpot
    const open = previousClose > 0 ? previousClose : safeSpot
    const close = index === count - 1 ? safeSpot : Math.max(safeSpot * 0.2, safeSpot + wiggle)
    const high = Math.max(open, close) + Math.abs(wiggle) * 0.55
    const low = Math.max(1e-12, Math.min(open, close) - Math.abs(wiggle) * 0.4)
    previousClose = close
    return {
      time,
      open,
      high,
      low,
      close,
      volume: 80000 + Math.abs(Math.cos(index * 0.31)) * 420000 + index * 1800,
    }
  })
}

async function fetchSpotFromBalance(): Promise<number> {
  const token = typeof localStorage !== 'undefined' ? String(localStorage.getItem('token') || '').trim() : ''
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`
  const url = `${API_BASE}/v1/wallet/balance`
  const response = await fetch(url, { headers })
  if (!response.ok) throw new Error(`balance ${response.status}`)
  const body = await response.json()
  const aix = Number(body?.aix_price ?? body?.aixPrice)
  const win = Number(body?.win_price ?? body?.winPrice)
  const rate = Number(body?.aix_to_win_rate ?? body?.aixToWinRate)
  if (Number.isFinite(rate) && rate > 0) return rate
  if (Number.isFinite(aix) && aix > 0 && Number.isFinite(win) && win > 0) return aix / win
  throw new Error('spot unavailable')
}

function resolveApiUrl(interval: MarketInterval): string {
  const joiner = KLINE_API.indexOf('?') >= 0 ? '&' : '?'
  const query = `pair=${encodeURIComponent(KLINE_PAIR)}&interval=${encodeURIComponent(interval)}&limit=120`
  if (/^https?:\/\//i.test(KLINE_API)) return `${KLINE_API}${joiner}${query}`
  const path = KLINE_API.charAt(0) === '/' ? KLINE_API : `/${KLINE_API}`
  return `${API_BASE}${path}${joiner}${query}`
}

function sourceFromPayload(payload: unknown): MarketSource {
  if (payload && typeof payload === 'object') {
    const source = String((payload as Record<string, unknown>).source || '').trim().toLowerCase()
    if (source === 'aix_win' || source === 'demo' || source === 'api') return source as MarketSource
  }
  return 'aix_win'
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

const localPriceProvider: MarketDataProvider = {
  async getCandles(pair, interval) {
    const spot = await fetchSpotFromBalance()
    return { pair, interval, candles: createPriceCandles(interval, spot), source: 'aix_win' }
  },
}

export const getAixWinCandles = async (interval: MarketInterval): Promise<MarketSnapshot> => {
  try {
    return await remoteProvider.getCandles(KLINE_PAIR, interval)
  } catch {
    return localPriceProvider.getCandles(KLINE_PAIR, interval)
  }
}
