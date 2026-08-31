<template>
  <section class="market-section" aria-labelledby="market-title">
    <div class="market-shell">
      <header class="market-header">
        <div>
          <p class="kicker">{{ $t('market.liveMarket') }} / 01</p>
          <h2 id="market-title">AIX / WIN</h2>
          <p class="market-label">{{ $t('market.overview') }}</p>
        </div>
        <div v-if="latest" class="price-block">
          <strong>{{ formatPrice(latest.close) }}</strong>
          <span :class="change >= 0 ? 'up' : 'down'">{{ change >= 0 ? '+' : '' }}{{ change.toFixed(2) }}%</span>
        </div>
      </header>

      <div class="market-stats" :aria-label="$t('market.stats24h')">
        <div><span>{{ $t('market.high24h') }}</span><strong>{{ formatPrice(dayHigh) }}</strong></div>
        <div><span>{{ $t('market.low24h') }}</span><strong>{{ formatPrice(dayLow) }}</strong></div>
        <div><span>{{ $t('market.volume24h') }}</span><strong>{{ formatVolume(totalVolume) }}</strong></div>
        <div class="source"><span>{{ $t('market.sourceLabel') }}</span><strong><i></i> {{ $t('market.dataSource') }}</strong></div>
      </div>

      <div class="terminal">
        <div class="toolbar">
          <div class="intervals" :aria-label="$t('market.intervals')">
            <button v-for="item in intervals" :key="item" type="button" :class="{ active: interval === item }" :aria-pressed="interval === item" @click="changeInterval(item)">{{ item }}</button>
          </div>
          <div v-if="hovered" class="ohlc">
            <span>O {{ formatPrice(hovered.open) }}</span><span>H {{ formatPrice(hovered.high) }}</span><span>L {{ formatPrice(hovered.low) }}</span><span>C {{ formatPrice(hovered.close) }}</span>
          </div>
        </div>

        <div v-if="loading" class="chart-state">{{ $t('market.loading') }}</div>
        <div v-else-if="error" class="chart-state error">{{ error }}</div>
        <div v-else class="chart-wrap" @pointermove="handlePointer" @pointerleave="hoverIndex = null">
          <svg class="chart" viewBox="0 0 1000 430" role="img" aria-labelledby="chart-title chart-desc" preserveAspectRatio="none">
            <title id="chart-title">{{ $t('market.chartTitle', { interval }) }}</title>
            <desc id="chart-desc">{{ $t('market.chartDescription') }}</desc>
            <g class="grid">
              <line v-for="y in gridY" :key="`y-${y}`" x1="48" :y1="y" x2="932" :y2="y" />
              <line v-for="x in gridX" :key="`x-${x}`" :x1="x" y1="22" :x2="x" y2="382" />
            </g>
            <g class="volume">
              <rect v-for="(candle,index) in candles" :key="`v-${candle.time}`" :x="candleX(index)-candleWidth/2" :y="volumeY(candle.volume)" :width="candleWidth" :height="382-volumeY(candle.volume)" :class="candle.close >= candle.open ? 'up' : 'down'" />
            </g>
            <g class="candles">
              <g v-for="(candle,index) in candles" :key="candle.time" :class="candle.close >= candle.open ? 'up' : 'down'">
                <line :x1="candleX(index)" :y1="priceY(candle.high)" :x2="candleX(index)" :y2="priceY(candle.low)" />
                <rect :x="candleX(index)-candleWidth/2" :y="Math.min(priceY(candle.open),priceY(candle.close))" :width="candleWidth" :height="Math.max(2,Math.abs(priceY(candle.open)-priceY(candle.close)))" />
              </g>
            </g>
            <g class="axis-labels">
              <text v-for="(price,index) in priceTicks" :key="price" x="944" :y="gridY[index]+4">{{ formatPrice(price) }}</text>
              <text v-for="tick in timeTicks" :key="tick.index" :x="candleX(tick.index)" y="414" text-anchor="middle">{{ formatTime(tick.time) }}</text>
            </g>
            <g v-if="hoverIndex !== null && hovered" class="crosshair">
              <line :x1="candleX(hoverIndex)" y1="22" :x2="candleX(hoverIndex)" y2="382" />
              <line x1="48" :y1="priceY(hovered.close)" x2="932" :y2="priceY(hovered.close)" />
              <rect x="936" :y="priceY(hovered.close)-13" width="60" height="24" rx="3" />
              <text x="966" :y="priceY(hovered.close)+4" text-anchor="middle">{{ formatPrice(hovered.close) }}</text>
            </g>
          </svg>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getAixWinCandles, type Candle, type MarketInterval } from '@/services/marketData'

const { t: $t, locale } = useI18n()
const intervals: MarketInterval[] = ['15m', '1h', '4h', '1d']
const interval = ref<MarketInterval>('1h')
const candles = ref<Candle[]>([])
const loading = ref(true)
const error = ref('')
const hoverIndex = ref<number | null>(null)
let requestId = 0

// Array.prototype.at 直到较新的 WebView 才提供；索引写法兼容旧内核。
const latest = computed(() => candles.value[candles.value.length - 1])
const first = computed(() => candles.value[0])
const change = computed(() => latest.value && first.value ? ((latest.value.close - first.value.open) / first.value.open) * 100 : 0)
const dayHigh = computed(() => Math.max(...candles.value.map((item) => item.high), 0))
const dayLow = computed(() => candles.value.length ? Math.min(...candles.value.map((item) => item.low)) : 0)
const totalVolume = computed(() => candles.value.reduce((sum, item) => sum + item.volume, 0))
const priceMin = computed(() => dayLow.value - (dayHigh.value - dayLow.value) * 0.08)
const priceMax = computed(() => dayHigh.value + (dayHigh.value - dayLow.value) * 0.08)
const maxVolume = computed(() => Math.max(...candles.value.map((item) => item.volume), 1))
const candleWidth = computed(() => Math.max(4, Math.min(10, 680 / Math.max(candles.value.length, 1))))
const hovered = computed(() => hoverIndex.value === null ? null : candles.value[hoverIndex.value])
const gridY = [22, 94, 166, 238, 310]
const gridX = [48, 269, 490, 711, 932]
const priceTicks = computed(() => gridY.map((_, index) => priceMax.value - (priceMax.value - priceMin.value) * index / (gridY.length - 1)))
const timeTicks = computed(() => [0, 16, 32, 48, 63].filter((index) => candles.value[index]).map((index) => ({ index, time: candles.value[index].time })))

const priceY = (price: number) => 22 + ((priceMax.value - price) / Math.max(priceMax.value - priceMin.value, 0.001)) * 288
const volumeY = (volume: number) => 382 - (volume / maxVolume.value) * 54
const candleX = (index: number) => 52 + index * (876 / Math.max(candles.value.length - 1, 1))
const formatPrice = (value: number) => value.toFixed(4)
const formatVolume = (value: number) => value >= 1e6 ? `${(value / 1e6).toFixed(2)}M` : `${(value / 1e3).toFixed(1)}K`
const formatTime = (time: number) => new Intl.DateTimeFormat(locale.value, interval.value === '1d' ? { month: '2-digit', day: '2-digit' } : { hour: '2-digit', minute: '2-digit', hour12: false }).format(time)

const load = async () => {
  const currentRequest = ++requestId
  loading.value = true
  error.value = ''
  try {
    const result = await getAixWinCandles(interval.value)
    if (currentRequest === requestId) candles.value = result.candles
  } catch {
    if (currentRequest === requestId) error.value = $t('market.unavailable')
  } finally {
    if (currentRequest === requestId) loading.value = false
  }
}
const changeInterval = (value: MarketInterval) => {
  if (interval.value === value) return
  interval.value = value
  hoverIndex.value = null
  void load()
}
const handlePointer = (event: PointerEvent) => {
  const target = event.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()
  const ratio = Math.min(1, Math.max(0, (event.clientX - rect.left) / rect.width))
  hoverIndex.value = Math.round(ratio * (candles.value.length - 1))
}

onMounted(load)
</script>

<style scoped>
.market-section{position:relative;padding:94px max(34px,calc((100% - 1120px)/2));background:#f7f8fa;color:#0a0b0d;border-top:1px solid #dfe3ea}.market-section::before{content:"";position:absolute;top:-1px;left:max(34px,calc((100% - 1120px)/2));width:112px;height:8px;background:#0052ff}.market-shell{max-width:1120px;margin:0 auto}.market-header{display:flex;align-items:flex-end;justify-content:space-between;gap:24px}.kicker{margin:0 0 14px;color:#0052ff;font:600 9px/1 var(--aix-font-display);letter-spacing:.17em}.market-header h2{margin:0;font:700 42px/1 var(--aix-font-display);letter-spacing:-.04em}.market-label{margin:10px 0 0;color:#606775;font-size:14px}.price-block{display:flex;align-items:baseline;gap:14px}.price-block strong{font:700 40px/1 var(--aix-font-display);letter-spacing:-.04em}.price-block span{font:700 13px/1 var(--aix-font-display)}.up{color:#0052ff}.down{color:#3e4653}.market-stats{display:grid;grid-template-columns:repeat(4,1fr);margin-top:30px;border:1px solid #dfe3ea;background:#fff}.market-stats>div{display:flex;flex-direction:column;gap:9px;padding:17px 20px;border-right:1px solid #dfe3ea}.market-stats>div:last-child{border-right:0}.market-stats span{color:#707887;font:600 8px/1 var(--aix-font-display);letter-spacing:.14em}.market-stats strong{font:700 14px/1 var(--aix-font-display)}.source strong{display:flex;align-items:center;gap:7px}.source i{width:6px;height:6px;border-radius:50%;background:#0052ff;box-shadow:0 0 0 4px rgba(0,82,255,.1)}.terminal{margin-top:14px;border:1px solid #cfd5df;background:#fff;box-shadow:0 24px 70px rgba(12,23,48,.07)}.toolbar{display:flex;align-items:center;justify-content:space-between;min-height:52px;padding:0 18px;border-bottom:1px solid #dfe3ea}.intervals{display:flex;gap:4px}.intervals button{min-width:46px;padding:9px 10px;border:0;background:transparent;color:#697181;font:700 11px/1 var(--aix-font-display);cursor:pointer}.intervals button.active{background:#0052ff;color:#fff}.intervals button:focus-visible{outline:2px solid #0052ff;outline-offset:2px}.ohlc{display:flex;gap:14px;color:#697181;font:600 9px/1 var(--aix-font-display)}.chart-wrap{position:relative;width:100%;aspect-ratio:2.32/1;min-height:310px}.chart{display:block;width:100%;height:100%}.grid line{stroke:#e6e9ef;stroke-width:1}.volume rect{opacity:.1}.candles line{stroke:currentColor;stroke-width:1.25}.candles rect{fill:currentColor}.axis-labels{fill:#78808e;font:600 10px/1 var(--aix-font-display)}.crosshair line{stroke:#0052ff;stroke-width:1;stroke-dasharray:4 5;opacity:.6}.crosshair rect{fill:#0052ff}.crosshair text{fill:#fff;font:700 9px/1 var(--aix-font-display)}.chart-state{display:flex;align-items:center;justify-content:center;min-height:310px;color:#697181;font:650 14px/1 var(--aix-font-display)}.chart-state.error{color:#0a0b0d}
@media(max-width:759px){.market-section{padding:76px 22px}.market-section::before{left:22px;width:82px}.market-header{align-items:flex-start}.market-header h2{font-size:32px}.price-block{flex-direction:column;align-items:flex-end;gap:7px}.price-block strong{font-size:27px}.market-stats{grid-template-columns:repeat(2,1fr)}.market-stats>div:nth-child(2){border-right:0}.market-stats>div:nth-child(-n+2){border-bottom:1px solid #dfe3ea}.toolbar{align-items:flex-start;flex-direction:column;gap:10px;padding:12px}.ohlc{width:100%;justify-content:space-between;gap:4px}.chart-wrap{min-height:270px;aspect-ratio:1.35/1}.axis-labels{font-size:8px}}
.market-section{background:transparent;color:#f6f9ff;border-color:rgba(126,185,255,.14)}.market-section::before{height:3px;background:#1989ff;box-shadow:0 0 24px rgba(25,137,255,.5)}.kicker{color:#5eb4ff}.market-label,.market-stats span{color:#8fa1bb}.market-header h2,.price-block strong,.market-stats strong{color:#f6f9ff}.market-stats{border-color:rgba(126,185,255,.16);background:rgba(5,16,40,.54);backdrop-filter:blur(15px)}.market-stats>div{border-color:rgba(126,185,255,.12)}.terminal{border-color:rgba(126,185,255,.2);background:rgba(4,14,36,.82);box-shadow:0 32px 90px rgba(0,4,22,.5),inset 0 1px rgba(255,255,255,.06);backdrop-filter:blur(18px)}.toolbar{border-color:rgba(126,185,255,.13)}.price-block .up,.candles .up,.volume .up{color:#168bff;fill:#168bff;stroke:#168bff}.pair strong{color:#f6f9ff}.pair span,.intervals button{color:#8395af}.intervals button.active{background:linear-gradient(135deg,#168bff,#0052ff);color:#fff;box-shadow:0 8px 24px rgba(0,82,255,.28)}.source{color:#a8b8ce}.source i{background:#35a4ff;box-shadow:0 0 15px rgba(53,164,255,.65)}.chart{background:linear-gradient(180deg,rgba(8,24,57,.4),rgba(2,9,24,.7))}.grid line{stroke:rgba(126,185,255,.1)}.axis-labels{fill:#71839d}.crosshair line{stroke:#36a7ff}.crosshair rect{fill:#0d78ff}.crosshair text{fill:#fff}@media(max-width:759px){.market-stats>div:nth-child(-n+2){border-bottom-color:rgba(126,185,255,.12)}.terminal{margin-inline:-8px}.chart{min-height:330px}}@media(prefers-reduced-motion:reduce){.source i{box-shadow:none}}
</style>
