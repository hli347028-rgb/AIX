export type MarketInterval = '15m' | '1h' | '4h' | '1d'

export interface Candle {
  time: number
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export interface MarketSnapshot {
  pair: 'AIX/WIN'
  interval: MarketInterval
  candles: Candle[]
  source: 'demo' | 'ave'
}

export interface MarketDataProvider {
  getCandles(pair: string, interval: MarketInterval): Promise<MarketSnapshot>
}

const intervalMs: Record<MarketInterval, number> = {
  '15m': 15 * 60 * 1000,
  '1h': 60 * 60 * 1000,
  '4h': 4 * 60 * 60 * 1000,
  '1d': 24 * 60 * 60 * 1000,
}

const seedByInterval: Record<MarketInterval, number> = { '15m': 11, '1h': 23, '4h': 37, '1d': 53 }

function createDemoCandles(interval: MarketInterval, count = 64): Candle[] {
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

const demoProvider: MarketDataProvider = {
  async getCandles(_pair, interval) {
    return { pair: 'AIX/WIN', interval, candles: createDemoCandles(interval), source: 'demo' }
  },
}

// AVE 接入边界：后续在此实现请求与字段映射，并将 activeProvider 指向 aveProvider。
// UI 只依赖 MarketSnapshot，AVE 原始响应格式不会渗透到图表组件。
export const mapAveCandles = (rows: Array<Record<string, unknown>>): Candle[] => rows.map((row) => ({
  time: Number(row.time ?? row.timestamp),
  open: Number(row.open),
  high: Number(row.high),
  low: Number(row.low),
  close: Number(row.close),
  volume: Number(row.volume ?? 0),
}))

const activeProvider: MarketDataProvider = demoProvider

export const getAixWinCandles = (interval: MarketInterval) => activeProvider.getCandles('AIX-WIN', interval)
