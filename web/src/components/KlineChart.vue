<template>
  <div
    ref="wrapRef"
    class="kline-chart"
    data-kline-chart
    role="img"
    :aria-label="title"
    @wheel.stop.prevent="onWheel"
    @pointerdown="onPointerDown"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @pointercancel="onPointerUp"
    @pointerleave="clearHover"
  >
    <canvas ref="canvasRef" class="kline-canvas" />
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Candle, MarketInterval } from '@/services/marketData'

const props = defineProps<{
  candles: Candle[]
  interval: MarketInterval
  locale: string
  title: string
}>()

const emit = defineEmits<{
  (event: 'hover', candle: Candle | null): void
}>()

const wrapRef = ref<HTMLElement | null>(null)
const canvasRef = ref<HTMLCanvasElement | null>(null)
const viewStart = ref(0)
const viewCount = ref(64)
const hoverIndex = ref(-1)
let dragging = false
let dragX = 0
let dragStart = 0
let raf = 0

const pad = { top: 16, right: 62, bottom: 28, left: 10 }
const volumeRatio = 0.22

const clampView = () => {
  const total = props.candles.length
  if (!total) {
    viewStart.value = 0
    viewCount.value = 64
    return
  }
  viewCount.value = Math.max(12, Math.min(total, viewCount.value))
  viewStart.value = Math.max(0, Math.min(total - viewCount.value, viewStart.value))
}

const visibleCandles = () => {
  clampView()
  return props.candles.slice(viewStart.value, viewStart.value + viewCount.value)
}

const formatPrice = (value: number) => {
  if (!Number.isFinite(value)) return '--'
  if (value >= 100) return value.toFixed(2)
  if (value >= 1) return value.toFixed(4)
  return value.toFixed(6)
}

const formatTime = (time: number) => new Intl.DateTimeFormat(props.locale, props.interval === '1d'
  ? { month: '2-digit', day: '2-digit' }
  : { hour: '2-digit', minute: '2-digit', hour12: false }).format(time)

const indexFromX = (x: number, width: number, count: number) => {
  const chartWidth = Math.max(1, width - pad.left - pad.right)
  const ratio = Math.min(1, Math.max(0, (x - pad.left) / chartWidth))
  return Math.round(ratio * Math.max(count - 1, 0))
}

const draw = () => {
  const canvas = canvasRef.value
  const wrap = wrapRef.value
  if (!canvas || !wrap) return
  const width = Math.max(1, wrap.clientWidth)
  const height = Math.max(1, wrap.clientHeight)
  const dpr = Math.max(1, window.devicePixelRatio || 1)
  if (canvas.width !== Math.round(width * dpr) || canvas.height !== Math.round(height * dpr)) {
    canvas.width = Math.round(width * dpr)
    canvas.height = Math.round(height * dpr)
    canvas.style.width = `${width}px`
    canvas.style.height = `${height}px`
  }
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, width, height)

  const candles = visibleCandles()
  if (!candles.length) return

  const chartWidth = width - pad.left - pad.right
  const chartHeight = height - pad.top - pad.bottom
  const volumeHeight = chartHeight * volumeRatio
  const priceHeight = chartHeight - volumeHeight - 10
  const highs = candles.map((item) => item.high)
  const lows = candles.map((item) => item.low)
  const volumes = candles.map((item) => item.volume)
  const priceMax = Math.max.apply(null, highs)
  const priceMin = Math.min.apply(null, lows)
  const pricePad = Math.max((priceMax - priceMin) * 0.08, priceMax * 0.002, 0.0001)
  const top = priceMax + pricePad
  const bottom = Math.max(0, priceMin - pricePad)
  const maxVolume = Math.max.apply(null, volumes.concat([1]))
  const step = chartWidth / Math.max(candles.length, 1)
  const bodyWidth = Math.max(3, Math.min(12, step * 0.62))
  const priceY = (price: number) => pad.top + ((top - price) / (top - bottom)) * priceHeight
  const volumeY = (volume: number) => pad.top + priceHeight + 10 + (1 - volume / maxVolume) * volumeHeight
  const candleX = (index: number) => pad.left + step * index + step / 2

  ctx.strokeStyle = 'rgba(126,185,255,.12)'
  ctx.lineWidth = 1
  for (let i = 0; i < 5; i += 1) {
    const y = pad.top + (priceHeight * i) / 4
    ctx.beginPath()
    ctx.moveTo(pad.left, y)
    ctx.lineTo(width - pad.right, y)
    ctx.stroke()
  }

  ctx.font = '600 10px AixDisplay, sans-serif'
  ctx.fillStyle = '#71839d'
  ctx.textAlign = 'left'
  ctx.textBaseline = 'middle'
  for (let i = 0; i < 5; i += 1) {
    const price = top - ((top - bottom) * i) / 4
    ctx.fillText(formatPrice(price), width - pad.right + 8, pad.top + (priceHeight * i) / 4)
  }

  for (let i = 0; i < candles.length; i += 1) {
    const candle = candles[i]
    const x = candleX(i)
    const up = candle.close >= candle.open
    const color = up ? '#168bff' : '#8b95a5'
    ctx.strokeStyle = color
    ctx.fillStyle = color
    ctx.globalAlpha = 0.22
    ctx.fillRect(x - bodyWidth / 2, volumeY(candle.volume), bodyWidth, pad.top + priceHeight + 10 + volumeHeight - volumeY(candle.volume))
    ctx.globalAlpha = 1
    ctx.beginPath()
    ctx.moveTo(x, priceY(candle.high))
    ctx.lineTo(x, priceY(candle.low))
    ctx.stroke()
    const bodyTop = priceY(Math.max(candle.open, candle.close))
    const bodyBottom = priceY(Math.min(candle.open, candle.close))
    ctx.fillRect(x - bodyWidth / 2, bodyTop, bodyWidth, Math.max(2, bodyBottom - bodyTop))
  }

  ctx.fillStyle = '#71839d'
  ctx.textAlign = 'center'
  ctx.textBaseline = 'top'
  const ticks = [0, Math.floor((candles.length - 1) / 2), candles.length - 1]
  for (let i = 0; i < ticks.length; i += 1) {
    const index = ticks[i]
    if (candles[index]) ctx.fillText(formatTime(candles[index].time), candleX(index), height - 18)
  }

  if (hoverIndex.value >= 0 && hoverIndex.value < candles.length) {
    const candle = candles[hoverIndex.value]
    const x = candleX(hoverIndex.value)
    const y = priceY(candle.close)
    ctx.strokeStyle = 'rgba(54,167,255,.55)'
    ctx.setLineDash([4, 5])
    ctx.beginPath()
    ctx.moveTo(x, pad.top)
    ctx.lineTo(x, pad.top + priceHeight + 10 + volumeHeight)
    ctx.moveTo(pad.left, y)
    ctx.lineTo(width - pad.right, y)
    ctx.stroke()
    ctx.setLineDash([])
    ctx.fillStyle = '#0d78ff'
    ctx.fillRect(width - pad.right + 4, y - 10, 54, 20)
    ctx.fillStyle = '#fff'
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText(formatPrice(candle.close), width - pad.right + 31, y)
  }
}

const scheduleDraw = () => {
  cancelAnimationFrame(raf)
  raf = requestAnimationFrame(draw)
}

const emitHover = (index: number, count: number) => {
  hoverIndex.value = index
  emit('hover', index >= 0 && index < count ? visibleCandles()[index] : null)
  scheduleDraw()
}

const clearHover = () => {
  hoverIndex.value = -1
  emit('hover', null)
  scheduleDraw()
}

const onWheel = (event: WheelEvent) => {
  const next = event.deltaY > 0 ? Math.round(viewCount.value * 1.12) : Math.round(viewCount.value / 1.12)
  const center = viewStart.value + viewCount.value / 2
  viewCount.value = next
  clampView()
  viewStart.value = Math.round(center - viewCount.value / 2)
  clampView()
  scheduleDraw()
}

const onPointerDown = (event: PointerEvent) => {
  dragging = true
  dragX = event.clientX
  dragStart = viewStart.value
  try {
    (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
  } catch {}
}

const onPointerMove = (event: PointerEvent) => {
  const wrap = wrapRef.value
  if (!wrap) return
  const rect = wrap.getBoundingClientRect()
  const candles = visibleCandles()
  if (dragging) {
    const step = (rect.width - pad.left - pad.right) / Math.max(viewCount.value, 1)
    viewStart.value = dragStart + Math.round((dragX - event.clientX) / Math.max(step, 4))
    clampView()
    scheduleDraw()
    return
  }
  emitHover(indexFromX(event.clientX - rect.left, rect.width, candles.length), candles.length)
}

const onPointerUp = () => {
  dragging = false
}

const resetView = () => {
  const total = props.candles.length
  viewCount.value = Math.min(80, Math.max(24, total))
  viewStart.value = Math.max(0, total - viewCount.value)
  hoverIndex.value = -1
  emit('hover', null)
  scheduleDraw()
}

const onResize = () => scheduleDraw()

watch(() => props.candles, resetView)
watch(() => props.interval, resetView)

onMounted(() => {
  resetView()
  addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  cancelAnimationFrame(raf)
  removeEventListener('resize', onResize)
})
</script>

<style scoped>
.kline-chart {
  position: relative;
  width: 100%;
  height: 100%;
  min-height: 220px;
  touch-action: none;
  cursor: crosshair;
  user-select: none;
}

.kline-canvas {
  display: block;
  width: 100%;
  height: 100%;
}
</style>
