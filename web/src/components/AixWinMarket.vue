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
        <div class="source"><span>{{ $t('market.sourceLabel') }}</span><strong><i></i> {{ sourceLabel }}</strong></div>
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

        <div v-if="embedSrc" class="chart-wrap">
          <iframe
            class="kline-embed"
            data-kline-chart
            :src="embedSrc"
            :title="$t('market.chartTitle', { interval })"
            loading="lazy"
            referrerpolicy="no-referrer"
          />
        </div>
        <div v-else-if="loading" class="chart-state">{{ $t('market.loading') }}</div>
        <div v-else-if="error" class="chart-state error">{{ error }}</div>
        <div v-else class="chart-wrap">
          <KlineChart
            class="chart"
            :candles="candles"
            :interval="interval"
            :locale="locale"
            :title="$t('market.chartTitle', { interval })"
            @hover="hovered = $event"
          />
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import KlineChart from '@/components/KlineChart.vue'
import {
  getAixWinCandles,
  hasKlineEmbed,
  resolveKlineEmbedUrl,
  type Candle,
  type MarketInterval,
  type MarketSource,
} from '@/services/marketData'

const { t: $t, locale } = useI18n()
const intervals: MarketInterval[] = ['15m', '1h', '4h', '1d']
const interval = ref<MarketInterval>('1h')
const candles = ref<Candle[]>([])
const source = ref<MarketSource>(hasKlineEmbed ? 'embed' : 'demo')
const loading = ref(!hasKlineEmbed)
const error = ref('')
const hovered = ref<Candle | null>(null)
let requestId = 0

const latest = computed(() => candles.value[candles.value.length - 1])
const first = computed(() => candles.value[0])
const change = computed(() => latest.value && first.value ? ((latest.value.close - first.value.open) / first.value.open) * 100 : 0)
const dayHigh = computed(() => candles.value.length ? Math.max.apply(null, candles.value.map((item) => item.high)) : 0)
const dayLow = computed(() => candles.value.length ? Math.min.apply(null, candles.value.map((item) => item.low)) : 0)
const totalVolume = computed(() => candles.value.reduce((sum, item) => sum + item.volume, 0))
const embedSrc = computed(() => hasKlineEmbed ? resolveKlineEmbedUrl(interval.value) : '')
const sourceLabel = computed(() => {
  if (source.value === 'demo') return $t('market.dataSourceDemo')
  if (source.value === 'embed') return $t('market.dataSourceEmbed')
  return $t('market.dataSourceLive')
})

const formatPrice = (value: number) => {
  if (!Number.isFinite(value) || !value) return '--'
  return value >= 1 ? value.toFixed(4) : value.toFixed(6)
}
const formatVolume = (value: number) => {
  if (!value) return '--'
  return value >= 1e6 ? `${(value / 1e6).toFixed(2)}M` : `${(value / 1e3).toFixed(1)}K`
}

const load = async () => {
  if (hasKlineEmbed) {
    source.value = 'embed'
    loading.value = false
    error.value = ''
    return
  }
  const currentRequest = ++requestId
  loading.value = true
  error.value = ''
  hovered.value = null
  try {
    const result = await getAixWinCandles(interval.value)
    if (currentRequest !== requestId) return
    candles.value = result.candles
    source.value = result.source
  } catch {
    if (currentRequest === requestId) error.value = $t('market.unavailable')
  } finally {
    if (currentRequest === requestId) loading.value = false
  }
}

const changeInterval = (value: MarketInterval) => {
  if (interval.value === value) return
  interval.value = value
  hovered.value = null
  void load()
}

onMounted(load)
</script>

<style scoped>
.market-section{position:relative;padding:94px max(34px,calc((100% - 1120px)/2));background:#f7f8fa;color:#0a0b0d;border-top:1px solid #dfe3ea}.market-section::before{content:"";position:absolute;top:-1px;left:max(34px,calc((100% - 1120px)/2));width:112px;height:8px;background:#0052ff}.market-shell{max-width:1120px;margin:0 auto}.market-header{display:flex;align-items:flex-end;justify-content:space-between;gap:24px}.kicker{margin:0 0 14px;color:#0052ff;font:600 9px/1 var(--aix-font-display);letter-spacing:.17em}.market-header h2{margin:0;font:700 42px/1 var(--aix-font-display);letter-spacing:-.04em}.market-label{margin:10px 0 0;color:#606775;font-size:14px}.price-block{display:flex;align-items:baseline;gap:14px}.price-block strong{font:700 40px/1 var(--aix-font-display);letter-spacing:-.04em}.price-block span{font:700 13px/1 var(--aix-font-display)}.up{color:#0052ff}.down{color:#3e4653}.market-stats{display:grid;grid-template-columns:repeat(4,1fr);margin-top:30px;border:1px solid #dfe3ea;background:#fff}.market-stats>div{display:flex;flex-direction:column;gap:9px;padding:17px 20px;border-right:1px solid #dfe3ea}.market-stats>div:last-child{border-right:0}.market-stats span{color:#707887;font:600 8px/1 var(--aix-font-display);letter-spacing:.14em}.market-stats strong{font:700 14px/1 var(--aix-font-display)}.source strong{display:flex;align-items:center;gap:7px}.source i{width:6px;height:6px;border-radius:50%;background:#0052ff;box-shadow:0 0 0 4px rgba(0,82,255,.1)}.terminal{margin-top:14px;border:1px solid #cfd5df;background:#fff;box-shadow:0 24px 70px rgba(12,23,48,.07)}.toolbar{display:flex;align-items:center;justify-content:space-between;min-height:52px;padding:0 18px;border-bottom:1px solid #dfe3ea}.intervals{display:flex;gap:4px}.intervals button{min-width:46px;padding:9px 10px;border:0;background:transparent;color:#697181;font:700 11px/1 var(--aix-font-display);cursor:pointer}.intervals button.active{background:#0052ff;color:#fff}.intervals button:focus-visible{outline:2px solid #0052ff;outline-offset:2px}.ohlc{display:flex;gap:14px;color:#697181;font:600 9px/1 var(--aix-font-display)}.chart-wrap{position:relative;width:100%;aspect-ratio:2.32/1;min-height:310px}.chart,.kline-embed{display:block;width:100%;height:100%;border:0;background:transparent}.chart-state{display:flex;align-items:center;justify-content:center;min-height:310px;color:#697181;font:650 14px/1 var(--aix-font-display)}.chart-state.error{color:#0a0b0d}
@media(max-width:759px){.market-section{padding:76px 22px}.market-section::before{left:22px;width:82px}.market-header{align-items:flex-start}.market-header h2{font-size:32px}.price-block{flex-direction:column;align-items:flex-end;gap:7px}.price-block strong{font-size:27px}.market-stats{grid-template-columns:repeat(2,1fr)}.market-stats>div:nth-child(2){border-right:0}.market-stats>div:nth-child(-n+2){border-bottom:1px solid #dfe3ea}.toolbar{align-items:flex-start;flex-direction:column;gap:10px;padding:12px}.ohlc{width:100%;justify-content:space-between;gap:4px}.chart-wrap{min-height:270px;aspect-ratio:1.35/1}}
.market-section{background:transparent;color:#f6f9ff;border-color:rgba(126,185,255,.14)}.market-section::before{height:3px;background:#1989ff;box-shadow:0 0 24px rgba(25,137,255,.5)}.kicker{color:#5eb4ff}.market-label,.market-stats span{color:#8fa1bb}.market-header h2,.price-block strong,.market-stats strong{color:#f6f9ff}.market-stats{border-color:rgba(126,185,255,.16);background:rgba(5,16,40,.54);backdrop-filter:blur(15px)}.market-stats>div{border-color:rgba(126,185,255,.12)}.terminal{border-color:rgba(126,185,255,.2);background:rgba(4,14,36,.82);box-shadow:0 32px 90px rgba(0,4,22,.5),inset 0 1px rgba(255,255,255,.06);backdrop-filter:blur(18px)}.toolbar{border-color:rgba(126,185,255,.13)}.price-block .up{color:#168bff}.pair strong{color:#f6f9ff}.pair span,.intervals button{color:#8395af}.intervals button.active{background:linear-gradient(135deg,#168bff,#0052ff);color:#fff;box-shadow:0 8px 24px rgba(0,82,255,.28)}.source{color:#a8b8ce}.source i{background:#35a4ff;box-shadow:0 0 15px rgba(53,164,255,.65)}.chart-wrap{background:linear-gradient(180deg,rgba(8,24,57,.4),rgba(2,9,24,.7))}.chart-state{color:#8395af}.chart-state.error{color:#f6f9ff}@media(max-width:759px){.market-stats>div:nth-child(-n+2){border-bottom-color:rgba(126,185,255,.12)}.terminal{margin-inline:-8px}.chart-wrap{min-height:330px}}@media(prefers-reduced-motion:reduce){.source i{box-shadow:none}}
</style>
