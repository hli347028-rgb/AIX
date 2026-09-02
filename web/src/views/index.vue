<template>
  <div
    ref="stageRef"
    class="timeline"
    :class="{ 'intro-complete': activeNode > 0 }"
    :style="timelineStyle"
    @wheel.prevent="onWheel"
    @pointerdown="onPointerDown"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @pointercancel="onPointerCancel"
  >
    <Teleport to="body"><Header /></Teleport>
    <main class="space-stage" :aria-label="$t('home.experienceAria')">
      <div class="space-camera" aria-hidden="true">
        <div
          v-for="(scene, index) in scenes"
          :key="scene.id"
          class="scene-image"
          :class="`scene-image--${index}`"
          :style="sceneStyle(index, scene.image)"
        />
        <div class="speed-lines" />
        <div class="vignette" />
      </div>

      <div class="opening" aria-live="polite">
        <div class="opening-space" aria-hidden="true">
          <div class="opening-tunnel" />
          <div class="opening-energy" />
          <div class="opening-vignette" />
        </div>
        <div class="opening-copy">
          <p>{{ $t('home.openingKicker') }}</p>
          <h1><span>{{ $t('home.openingPrefix') }}</span>{{ $t('home.openingTitle') }}</h1>
          <small>{{ $t('home.openingHint') }}</small>
        </div>
      </div>

      <section class="chapter chapter--future" :class="chapterClass(1)" :style="chapterStyle(1)">
        <article class="future-content">
          <header class="section-label"><span>01</span><small>FUTUREFI / STRATEGIC NARRATIVE</small></header>
          <div class="future-layout">
            <div class="future-lead">
              <p class="future-kicker">AIX × FUTUREFI</p>
              <h2>{{ $t('home.futureTitle') }}<br /><em>{{ $t('home.futureEmphasis') }}</em></h2>
              <p class="lead">{{ $t('home.futureLead') }}</p>
            </div>
            <div class="future-story">
              <p>{{ $t('home.futureStory1') }}</p>
              <div class="definition"><small>CORE PROJECT</small><strong>AIX — AI Prediction Exchange</strong><span>AI × Prediction Market × DeFi</span></div>
              <p>{{ $t('home.futureStory2') }}</p>
            </div>
          </div>
          <div class="protocol-axis" :aria-label="$t('home.protocolAria')">
            <div><small>01</small><b>AI Prediction Exchange</b><span>{{ $t('home.predictionTrading') }}</span></div><i>×</i>
            <div><small>02</small><b>FutureFi DAO</b><span>{{ $t('home.futureProtocol') }}</span></div><i>×</i>
            <div><small>03</small><b>WIN Chain</b><span>{{ $t('home.chainInfrastructure') }}</span></div>
          </div>
          <footer class="future-footer">
            <p class="thesis">{{ $t('home.flow') }}</p>
            <button class="primary-action" type="button" @click="router.push('/futurefi')"><span>{{ $t('home.viewNarrative') }}</span><b>→</b></button>
          </footer>
        </article>
      </section>

      <section class="chapter chapter--market" :class="chapterClass(2)" :style="chapterStyle(2)">
        <div class="market-intro content-surface">
          <header class="section-label"><span>02</span><small>{{ $t('market.liveMarket') }}</small></header>
          <h2>AIX / WIN</h2>
          <p>{{ $t('home.marketDescription') }}</p>
        </div>
        <div class="embedded-panel"><AixWinMarket embedded /></div>
      </section>

      <section class="chapter chapter--ecosystem" :class="chapterClass(3)" :style="chapterStyle(3)">
        <article class="ecosystem-copy content-surface">
          <header class="section-label"><span>03</span><small>ECOSYSTEM NETWORK</small></header>
          <h2>{{ $t('home.ecosystemTitle') }}<br /><em>{{ $t('home.ecosystemEmphasis') }}</em></h2>
          <p>{{ $t('home.ecosystemDescription') }}</p>
          <dl><div><dt>{{ $t('home.networkStatus') }}</dt><dd>ONLINE</dd></div><div><dt>{{ $t('home.ecosystemPartners') }}</dt><dd>06</dd></div><div><dt>{{ $t('home.settlementNetwork') }}</dt><dd>MULTI-CHAIN</dd></div></dl>
        </article>
        <div class="partners-panel"><PartnersWall /></div>
      </section>

      <section class="chapter chapter--hero" :class="chapterClass(4)" :style="chapterStyle(4)">
        <div class="hero-copy">
          <div class="eyebrow"><i /><span>AI PREDICTION EXCHANGE</span></div>
          <h2>{{ $t('home.heroTitle') }}<br /><em>{{ $t('home.heroEmphasis') }}</em></h2>
          <p>{{ $t('home.heroDescription') }}</p>
          <button class="primary-action hero-primary" type="button" @click="router.push('/node')"><span>{{ $t('home.startNow') }}</span><b>→</b></button>
          <div class="status"><span><i />{{ $t('home.oracle') }} <strong>ONLINE</strong></span><span>{{ $t('home.tradingPair') }} <strong>AIX / WIN</strong></span></div>
          <nav class="hero-actions" :aria-label="$t('home.quickLinksAria')">
            <button type="button" @click="router.push('/recharge')"><small>01</small><span>{{ $t('home.assetRecharge') }}</span><b>↗</b></button>
            <button class="featured" type="button" @click="openPrediction"><small>02</small><span>{{ $t('home.predictFuture') }}</span><b>→</b></button>
            <button type="button" @click="openWallet"><small>03</small><span>{{ $t('home.getWinWallet') }}</span><b>↗</b></button>
          </nav>
        </div>
      </section>

      <div class="timeline-ui" :aria-label="$t('home.currentChapter')"><span>{{ String(activeNode).padStart(2, '0') }}</span><div><i /></div><b>04</b></div>
      <p class="control-hint">{{ $t('home.controlHint') }}</p>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Header from '@/components/Header.vue'
import AixWinMarket from '@/components/AixWinMarket.vue'
import PartnersWall from '@/components/PartnersWall.vue'

const router = useRouter()
const { t: $t } = useI18n()
const stageRef = ref<HTMLElement | null>(null)
const scenes = [
  { id: 'opening', image: '/assets/timeline-00-time-tunnel-v2.png' },
  { id: 'future', image: '/assets/timeline-04-consensus.png' },
  { id: 'market', image: '/assets/timeline-03-market.png' },
  { id: 'ecosystem', image: '/assets/timeline-05-ecosystem.png' },
  { id: 'hero', image: '/assets/timeline-06-destination.png' },
]
const target = ref(0)
const current = ref(0)
const activeNode = ref(0)
const dragging = ref(false)
let startX = 0
let startY = 0
let startValue = 0
let raf = 0
let snapTimer = 0
type TouchAxis = 'pending' | 'horizontal' | 'vertical'
let touchAxis: TouchAxis = 'pending'
let touchStartX = 0
let touchStartY = 0
let touchLastX = 0
let touchLastY = 0
let touchLastTime = 0
let touchStartTime = 0
let touchStartNode = 0
let touchVelocity = 0
let touchChapter: HTMLElement | null = null
let touchTracking = false

const timelineStyle = computed(() => {
  const t = current.value
  const fraction = t - Math.floor(t)
  const wave = Math.sin(t * Math.PI)
  return {
    '--t': String(t),
    '--node': String(activeNode.value),
    '--fraction': String(fraction),
    '--cam-z': `${(fraction - 0.5) * 230}px`,
    '--cam-rotate': `${wave * 10}deg`,
    '--cam-scale': String(1.06 + wave * 0.15),
    '--speed-opacity': String(Math.max(0, wave * 0.46)),
  }
})
const sceneStyle = (index: number, image: string) => {
  const t = current.value
  const distance = Math.abs(t - index)
  const opacity = Math.max(0, Math.min(1, 1 - distance))
  const scale = 1.06 + distance * (index === 4 ? 0.08 : 0.1)
  const rotate = index === 1
    ? (t - 1) * -6
    : index === 2
      ? (t - 2) * 8
      : index === 3
        ? (t - 3) * -7
        : 0
  return {
    backgroundImage: `linear-gradient(90deg,rgba(1,7,20,.94),rgba(1,7,20,.26) 58%,rgba(1,7,20,.5)),url(${image})`,
    opacity: String(opacity),
    transform: rotate ? `scale(${scale}) rotate(${rotate}deg)` : `scale(${scale})`,
  }
}
const chapterClass = (index: number) => ({ active: Math.abs(current.value - index) < 0.98, interactive: Math.abs(current.value - index) < 0.08 })
const chapterStyle = (index: number) => {
  const distance = current.value - index
  const absoluteDistance = Math.abs(distance)
  const easedDistance = Math.sign(distance) * Math.min(1, absoluteDistance)
  return {
    '--chapter-opacity': Math.max(0, 1 - absoluteDistance * 1.05),
    '--chapter-scale': 1 + easedDistance * 0.055 - absoluteDistance * 0.018,
    '--chapter-x': `${easedDistance * -7}vw`,
    '--chapter-y': `${easedDistance * -3}vh`,
  }
}
const clamp = (value: number) => Math.max(0, Math.min(4, value))
const enterFuture = () => { target.value = 1 }
const animate = () => {
  const difference = target.value - current.value
  const easing = isPhone() ? Math.min(0.16, 0.105 + Math.abs(difference) * 0.035) : 0.09
  current.value += difference * easing
  if (Math.abs(difference) < 0.0006) current.value = target.value
  activeNode.value = Math.round(current.value)
  raf = requestAnimationFrame(animate)
}
const snap = () => {
  clearTimeout(snapTimer)
  snapTimer = window.setTimeout(() => { target.value = Math.round(target.value) }, 180)
}
const onWheel = (event: WheelEvent) => {
  const delta = Math.abs(event.deltaY) >= Math.abs(event.deltaX) ? event.deltaY : event.deltaX
  target.value = clamp(target.value + delta * 0.0012)
  snap()
}
const isPhone = () => window.innerWidth <= 540
const isInteractiveTarget = (targetElement: EventTarget | null) =>
  targetElement instanceof HTMLElement && Boolean(targetElement.closest('button,a,input,select,textarea,iframe,[role="button"],[data-kline-chart]'))

const onPointerDown = (event: PointerEvent) => {
  if (isPhone() || isInteractiveTarget(event.target)) return
  dragging.value = true
  startX = event.clientX
  startY = event.clientY
  startValue = target.value
  stageRef.value?.setPointerCapture(event.pointerId)
}
const onPointerMove = (event: PointerEvent) => {
  if (!dragging.value || isPhone()) return
  const deltaX = startX - event.clientX
  const deltaY = startY - event.clientY
  const delta = Math.abs(deltaX) >= Math.abs(deltaY) ? deltaX : deltaY
  target.value = clamp(startValue + delta / Math.max(240, Math.min(innerWidth, innerHeight) * 0.62))
}
const onPointerUp = () => {
  if (!dragging.value || isPhone()) return
  dragging.value = false
  snap()
}
const onPointerCancel = () => {
  if (isPhone()) return
  dragging.value = false
  snap()
}

const onTouchStart = (event: TouchEvent) => {
  if (!isPhone() || event.touches.length !== 1 || isInteractiveTarget(event.target)) return
  const touch = event.touches[0]
  touchTracking = true
  touchAxis = 'pending'
  touchStartX = touch.clientX
  touchStartY = touch.clientY
  touchLastX = touch.clientX
  touchLastY = touch.clientY
  touchStartTime = performance.now()
  touchLastTime = touchStartTime
  touchStartNode = Math.round(target.value)
  touchVelocity = 0
  touchChapter = (event.target as HTMLElement).closest<HTMLElement>('.chapter')
}
const onTouchMove = (event: TouchEvent) => {
  if (!touchTracking || event.touches.length !== 1) return
  const touch = event.touches[0]
  const deltaX = touch.clientX - touchStartX
  const deltaY = touch.clientY - touchStartY
  if (touchAxis === 'pending' && Math.hypot(deltaX, deltaY) >= 7) {
    touchAxis = Math.abs(deltaX) > Math.abs(deltaY) * 1.05 ? 'horizontal' : 'vertical'
  }
  if (touchAxis === 'pending') return
  const primaryDelta = touchAxis === 'horizontal' ? deltaX : deltaY
  const previousPrimary = touchAxis === 'horizontal' ? touchLastX : touchLastY
  const currentPrimary = touchAxis === 'horizontal' ? touch.clientX : touch.clientY
  if (touchAxis === 'vertical' && touchChapter) {
    const movingUp = deltaY < 0
    const canScrollDown = touchChapter.scrollTop + touchChapter.clientHeight < touchChapter.scrollHeight - 2
    const canScrollUp = touchChapter.scrollTop > 2
    if ((movingUp && canScrollDown) || (!movingUp && canScrollUp)) {
      event.preventDefault()
      touchChapter.scrollTop -= touch.clientY - touchLastY
      touchLastX = touch.clientX
      touchLastY = touch.clientY
      touchStartX = touch.clientX
      touchStartY = touch.clientY
      touchLastTime = performance.now()
      return
    }
  }
  event.preventDefault()
  const now = performance.now()
  const elapsed = Math.max(8, now - touchLastTime)
  const instantVelocity = (currentPrimary - previousPrimary) / elapsed
  touchVelocity = touchVelocity * 0.65 + instantVelocity * 0.35
  touchLastX = touch.clientX
  touchLastY = touch.clientY
  touchLastTime = now
  const travelBase = touchAxis === 'horizontal' ? Math.max(260, innerWidth * 0.82) : Math.max(300, innerHeight * 0.58)
  const travel = -primaryDelta / travelBase
  const rawProgress = touchStartNode + travel
  const resistedProgress = rawProgress < 0 ? rawProgress * 0.18 : rawProgress > 4 ? 4 + (rawProgress - 4) * 0.18 : rawProgress
  current.value = resistedProgress
  target.value = resistedProgress
  activeNode.value = Math.round(clamp(resistedProgress))
}
const resetChapterScroll = (node: number) => {
  const chapter = stageRef.value?.querySelectorAll<HTMLElement>('.chapter')[node - 1]
  if (chapter) chapter.scrollTop = 0
}
const finishTouch = (cancelled = false) => {
  if (!touchTracking && touchAxis === 'pending') return
  const distance = touchAxis === 'horizontal' ? touchLastX - touchStartX : touchLastY - touchStartY
  const duration = Math.max(1, performance.now() - touchStartTime)
  const averageVelocity = distance / duration
  const velocity = Math.abs(touchVelocity) > Math.abs(averageVelocity) ? touchVelocity : averageVelocity
  let destination = touchStartNode
  if (!cancelled && touchAxis !== 'pending') {
    const threshold = touchAxis === 'horizontal' ? Math.min(82, innerWidth * 0.2) : Math.min(92, innerHeight * 0.12)
    const shouldAdvance = Math.abs(distance) >= threshold || Math.abs(velocity) >= 0.38
    if (shouldAdvance) destination += distance < 0 ? 1 : -1
  }
  destination = clamp(destination)
  target.value = destination
  activeNode.value = destination
  if (destination !== touchStartNode && destination > 0) requestAnimationFrame(() => resetChapterScroll(destination))
  touchTracking = false
  touchAxis = 'pending'
  touchVelocity = 0
  touchChapter = null
}
const onTouchEnd = () => finishTouch(false)
const onTouchCancel = () => finishTouch(true)
const onKey = (event: KeyboardEvent) => {
  if (['Enter', 'ArrowRight', 'ArrowDown', 'PageDown'].includes(event.key)) target.value = clamp(Math.round(target.value) + 1)
  if (['ArrowLeft', 'ArrowUp', 'PageUp'].includes(event.key)) target.value = clamp(Math.round(target.value) - 1)
}
const openPrediction = () => window.open('https://prediction-exchange-lovat.vercel.app', '_blank', 'noopener,noreferrer')
const openWallet = () => window.open('https://testnet.wallet.eoeo.info/06bx', '_blank', 'noopener,noreferrer')

onMounted(() => {
  raf = requestAnimationFrame(animate)
  addEventListener('keydown', onKey)
  stageRef.value?.addEventListener('touchstart', onTouchStart, { passive: true })
  stageRef.value?.addEventListener('touchmove', onTouchMove, { passive: false })
  stageRef.value?.addEventListener('touchend', onTouchEnd, { passive: true })
  stageRef.value?.addEventListener('touchcancel', onTouchCancel, { passive: true })
})
onBeforeUnmount(() => {
  cancelAnimationFrame(raf)
  clearTimeout(snapTimer)
  removeEventListener('keydown', onKey)
  stageRef.value?.removeEventListener('touchstart', onTouchStart)
  stageRef.value?.removeEventListener('touchmove', onTouchMove)
  stageRef.value?.removeEventListener('touchend', onTouchEnd)
  stageRef.value?.removeEventListener('touchcancel', onTouchCancel)
})
</script>

<style scoped>
.timeline{--t:0;position:fixed;top:0;right:0;bottom:0;left:0;inset:0;width:100%;height:100%;height:100vh;min-height:100%;overflow:hidden;background:#020817;color:#f5f9ff;font-family:var(--aix-font);touch-action:none;user-select:none}.timeline :deep(.app-header){z-index:12000}.space-stage{position:absolute;top:0;right:0;bottom:0;left:0;inset:0;overflow:hidden;perspective:1200px}.space-camera{pointer-events:none}.space-camera{position:absolute;top:-10%;right:-10%;bottom:-10%;left:-10%;transform:perspective(900px) translateZ(var(--cam-z,0px)) rotate(var(--cam-rotate,0deg)) scale(var(--cam-scale,1.06));will-change:transform}.scene-image{position:absolute;top:0;right:0;bottom:0;left:0;inset:0;background-position:center;background-size:cover}.speed-lines{position:absolute;top:-20%;right:-20%;bottom:-20%;left:-20%;opacity:var(--speed-opacity,0);background:repeating-conic-gradient(from 0deg at 50% 50%,transparent 0 3deg,rgba(69,170,255,.13) 3.2deg 3.4deg);transform:scale(1.4)}.vignette{position:absolute;top:0;right:0;bottom:0;left:0;inset:0;background:radial-gradient(circle,transparent 30%,rgba(0,4,16,.78))}
.opening{position:fixed;z-index:10000;top:0;right:0;bottom:0;left:0;inset:0;display:flex;align-items:center;justify-content:center;overflow:hidden;background:#020817;pointer-events:none;transition:opacity .85s ease,visibility .85s}.opening-space,.opening-tunnel,.opening-energy,.opening-vignette{position:absolute;top:0;right:0;bottom:0;left:0;inset:0}.opening-space{overflow:hidden;animation:opening-flight 2.65s cubic-bezier(.18,.7,.22,1) both}.opening-tunnel{inset:-8%;background:url('/assets/timeline-00-time-tunnel.png') center/cover no-repeat;filter:saturate(1.12) contrast(1.08);animation:tunnel-roll 2.65s cubic-bezier(.18,.7,.22,1) both;will-change:transform,filter}.opening-energy{inset:-25%;background:repeating-conic-gradient(from 12deg at 50% 50%,transparent 0 5.5deg,rgba(61,174,255,.13) 5.7deg 5.95deg,transparent 6.1deg 12deg);mix-blend-mode:screen;animation:energy-spin 2.65s cubic-bezier(.18,.7,.22,1) both}.opening-vignette{background:radial-gradient(circle at center,transparent 12%,rgba(1,7,20,.12) 39%,rgba(1,7,20,.84) 100%),linear-gradient(180deg,rgba(1,7,20,.36),transparent 34%,rgba(1,7,20,.5))}.opening-copy{position:relative;z-index:2;display:flex;align-items:center;flex-direction:column;gap:18px;padding:28px;text-align:center;animation:opening-copy 2.3s ease both}.timeline:not(.intro-complete) :deep(.app-header){opacity:0;visibility:hidden}.timeline :deep(.app-header){transition:opacity .45s ease,visibility .45s}.opening p,.opening h1{margin:0;color:#f7fbff;font-weight:700;letter-spacing:.1em;text-shadow:0 4px 18px #020817,0 12px 42px #000}.opening p{font-size:clamp(18px,2vw,28px)}.opening h1{font-size:clamp(48px,8vw,112px);letter-spacing:.04em}.opening-copy span{width:clamp(70px,9vw,140px);height:2px;background:#168bff;box-shadow:0 0 24px #168bff;animation:scan 1.25s .18s ease-in-out both}.intro-complete .opening{opacity:0;visibility:hidden;pointer-events:none}.intro-complete .opening-tunnel{transform:scale(1.68)}
@keyframes scan{from{transform:scaleX(0);opacity:.2}to{transform:scaleX(1);opacity:1}}
@keyframes opening-flight{0%{filter:brightness(.55)}42%{filter:brightness(.92)}100%{filter:brightness(1.08)}}
@keyframes tunnel-roll{0%{transform:scale(1.02) rotate(-5deg)}55%{transform:scale(1.18) rotate(2deg)}100%{transform:scale(1.48) rotate(8deg);filter:saturate(1.25) contrast(1.12) blur(1px)}}
@keyframes energy-spin{0%{opacity:0;transform:scale(.7) rotate(-18deg)}45%{opacity:.44}100%{opacity:.08;transform:scale(1.55) rotate(34deg)}}
@keyframes opening-copy{0%{opacity:0;transform:translateY(18px) scale(.96)}22%,72%{opacity:1;transform:none}100%{opacity:.12;transform:translateY(-12px) scale(1.04)}}
.chapter{position:absolute;top:0;right:0;bottom:0;left:0;inset:0;z-index:4;opacity:var(--chapter-opacity,0);pointer-events:none;filter:none;transform:translate3d(var(--chapter-x,7vw),var(--chapter-y,0),0) scale(var(--chapter-scale,.94));transition:none;will-change:transform,opacity}.chapter.active{opacity:var(--chapter-opacity,1);filter:none;transform:translate3d(var(--chapter-x,0),var(--chapter-y,0),0) scale(var(--chapter-scale,1))}.chapter.interactive{pointer-events:auto}.content-surface{background:linear-gradient(135deg,rgba(3,14,37,.9),rgba(4,20,48,.63));border:1px solid rgba(119,190,255,.2);box-shadow:0 28px 100px rgba(0,4,18,.52),inset 0 1px rgba(235,248,255,.07);backdrop-filter:blur(20px) saturate(1.15);clip-path:polygon(18px 0,100% 0,100% calc(100% - 18px),calc(100% - 18px) 100%,0 100%,0 18px)}.section-label{display:flex;align-items:center;gap:12px}.section-label span{display:grid;width:32px;height:32px;place-items:center;background:#0a78ff;color:#fff;font:700 11px/1 monospace}.section-label small{color:#66baff;font:700 11px/1.3 monospace;letter-spacing:.16em}.future-content,.ecosystem-copy{position:absolute;left:max(5vw,32px);top:50%;width:min(650px,48vw);padding:clamp(26px,3vw,46px);transform:translateY(-48%)}h2{margin:24px 0 18px;color:#f7fbff;font-size:clamp(40px,4.8vw,72px);font-weight:750;line-height:1.08;letter-spacing:-.045em;text-wrap:balance}h2 em{color:#72bdff;font-style:normal}.lead,.thesis,.ecosystem-copy>p,.market-intro>p{margin:0;color:#c5d5e9;font-size:clamp(14px,1.2vw,17px);font-weight:500;line-height:1.75;text-shadow:0 2px 16px rgba(0,0,0,.9)}.protocol-axis{display:flex;align-items:stretch;gap:8px;margin-top:24px}.protocol-axis div{flex:1;padding:13px;background:rgba(5,23,54,.7);border:1px solid rgba(104,185,255,.18)}.protocol-axis b,.protocol-axis span{display:block}.protocol-axis b{color:#fff;font-size:13px}.protocol-axis span{margin-top:5px;color:#91a9c5;font-size:11px}.protocol-axis i{align-self:center;color:#46a9ff;font-style:normal}.thesis{margin-top:18px;padding-left:15px;border-left:2px solid #168bff}.primary-action{display:flex;align-items:center;justify-content:space-between;width:250px;height:64px;margin-top:26px;padding:0 8px 0 24px;border:1px solid #55b4ff;background:linear-gradient(135deg,#087eff,#0044dc);color:#fff;font:750 15px/1 inherit;cursor:pointer;clip-path:polygon(14px 0,100% 0,100% calc(100% - 14px),calc(100% - 14px) 100%,0 100%,0 14px);box-shadow:inset 0 1px rgba(255,255,255,.35),0 16px 40px rgba(0,82,255,.34);transition:transform .25s,filter .25s}.primary-action b{display:grid;width:48px;height:48px;place-items:center;background:#eff8ff;color:#096eff;font-size:22px;clip-path:polygon(9px 0,100% 0,100% calc(100% - 9px),calc(100% - 9px) 100%,0 100%,0 9px)}.primary-action:hover{transform:translateY(-4px) perspective(500px) rotateX(4deg);filter:brightness(1.12)}
.market-intro{position:absolute;z-index:2;left:4vw;top:14%;width:min(330px,25vw);padding:24px}.market-intro h2{margin:16px 0 10px;font-size:clamp(32px,3.5vw,52px)}.embedded-panel{position:absolute;right:4vw;top:12%;width:min(980px,67vw);height:78%;overflow:hidden;background:rgba(2,10,27,.78);border:1px solid rgba(104,185,255,.22);box-shadow:0 36px 120px rgba(0,3,17,.62);clip-path:polygon(18px 0,100% 0,100% calc(100% - 18px),calc(100% - 18px) 100%,0 100%,0 18px)}.embedded-panel :deep(.market-section){height:100%;padding:22px!important;background:transparent!important;border:0!important}.embedded-panel :deep(.market-shell){height:100%;display:flex;flex-direction:column}.embedded-panel :deep(.market-header){flex:0 0 auto}.embedded-panel :deep(.market-header>div:first-child .market-label),.embedded-panel :deep(.market-header .kicker){display:none}.embedded-panel :deep(.market-stats){flex:0 0 auto;margin-top:14px}.embedded-panel :deep(.terminal){flex:1;min-height:0;margin-top:10px;display:flex;flex-direction:column}.embedded-panel :deep(.chart-wrap){flex:1;min-height:220px;aspect-ratio:auto}.embedded-panel :deep(.chart){height:100%;min-height:220px}.embedded-panel :deep(.market-section::before){display:none}
.ecosystem-copy{width:min(440px,34vw)}.ecosystem-copy dl{display:grid;grid-template-columns:repeat(3,1fr);gap:8px;margin:24px 0 0}.ecosystem-copy dl div{padding:13px;background:rgba(4,19,47,.76);border:1px solid rgba(104,185,255,.18)}dt{color:#8ea5bf;font-size:10px}dd{margin:7px 0 0;color:#fff;font:700 12px/1.2 monospace}.partners-panel{position:absolute;right:4vw;top:20%;width:min(820px,57vw);height:62%;overflow:hidden;background:rgba(2,10,27,.74);border:1px solid rgba(104,185,255,.2);box-shadow:0 35px 110px rgba(0,3,18,.58);clip-path:polygon(18px 0,100% 0,100% calc(100% - 18px),calc(100% - 18px) 100%,0 100%,0 18px)}.partners-panel :deep(.partners-wall){min-height:100%;padding:22px!important;background:transparent!important;border:0!important}
.hero-copy{position:absolute;left:5vw;top:47%;width:min(670px,58vw);transform:translateY(-50%)}.eyebrow{display:flex;align-items:center;gap:18px;color:#4eafff;font:700 12px/1 monospace;letter-spacing:.2em}.eyebrow i{width:58px;height:5px;background:#188dff;box-shadow:0 0 18px #188dff}.hero-copy h2{margin:54px 0 28px;font-size:clamp(60px,7vw,108px);line-height:.92}.hero-copy>p{color:#b8c8dc;font-size:clamp(18px,2vw,28px)}.status{display:flex;gap:54px;margin-top:42px;padding-top:18px;border-top:1px solid rgba(108,183,255,.2);color:#899db5;font:600 11px/1 monospace;letter-spacing:.1em}.status strong{margin-left:10px;color:#fff}.hero-actions{position:absolute;left:5vw;right:5vw;bottom:4vh;display:grid;grid-template-columns:repeat(3,1fr);gap:16px}.hero-actions button{position:relative;height:96px;padding:0 30px;border:1px solid rgba(104,185,255,.26);background:rgba(3,15,38,.8);color:#eef7ff;text-align:left;font:700 18px/1 inherit;cursor:pointer;clip-path:polygon(16px 0,100% 0,100% calc(100% - 16px),calc(100% - 16px) 100%,0 100%,0 16px);backdrop-filter:blur(16px)}.hero-actions button.featured{background:linear-gradient(135deg,#168bff,#0052ff);box-shadow:0 18px 48px rgba(0,82,255,.32)}.hero-actions small{position:absolute;right:20px;top:18px;color:#fff;font:700 10px/1 monospace}.timeline-ui{position:absolute;z-index:12;right:18px;top:50%;display:flex;align-items:center;flex-direction:column;gap:10px;color:#76bfff;font:700 10px/1 monospace;transform:translateY(-50%)}.timeline-ui div{width:2px;height:120px;background:rgba(99,178,255,.2)}.timeline-ui i{display:block;width:100%;height:25%;background:#168bff;box-shadow:0 0 12px #168bff;transform:translateY(calc(var(--node) * 100%))}.control-hint{position:absolute;z-index:10;right:28px;bottom:20px;margin:0;color:#8aa0ba;font:600 9px/1 monospace;letter-spacing:.14em}
@media(max-width:1100px){.future-content{width:56vw}.market-intro{left:3vw;width:28vw}.embedded-panel{right:3vw;width:66vw}.ecosystem-copy{left:3vw;width:38vw}.partners-panel{right:3vw;width:55vw}}
@media(max-width:540px){.space-camera{inset:-6% -42%}.scene-image{background-position:62% center}.future-content,.ecosystem-copy{left:18px;right:30px;top:12%;width:auto;padding:20px;transform:none}.future-content h2,.ecosystem-copy h2{margin:15px 0 10px;font-size:clamp(30px,8.5vw,40px)}.lead,.thesis,.ecosystem-copy>p{font-size:13px;line-height:1.55}.protocol-axis{margin-top:13px;gap:4px}.protocol-axis div{padding:8px}.protocol-axis b{font-size:11px}.protocol-axis span{font-size:9px}.thesis{margin-top:10px}.primary-action{width:210px;height:52px;margin-top:14px}.primary-action b{width:38px;height:38px}.market-intro{left:18px;right:30px;top:10%;width:auto;padding:14px}.market-intro p{font-size:12px;line-height:1.45}.market-intro h2{margin:9px 0 6px;font-size:30px}.embedded-panel{left:18px;right:30px;top:31%;width:auto;height:61%}.embedded-panel :deep(.market-section){padding:10px!important}.embedded-panel :deep(.market-header h2){font-size:25px}.embedded-panel :deep(.price-block strong){font-size:23px}.embedded-panel :deep(.market-stats){grid-template-columns:repeat(4,1fr)}.embedded-panel :deep(.market-stats>div){padding:8px 5px}.embedded-panel :deep(.market-stats span){font-size:6px}.embedded-panel :deep(.market-stats strong){font-size:9px}.embedded-panel :deep(.toolbar){min-height:40px;padding:5px}.embedded-panel :deep(.ohlc){display:none}.embedded-panel :deep(.chart-wrap),.embedded-panel :deep(.chart){min-height:190px}.ecosystem-copy{top:10%}.ecosystem-copy dl{margin-top:12px}.partners-panel{left:18px;right:30px;top:48%;width:auto;height:42%}.partners-panel :deep(.partners-wall){padding:10px!important}.hero-copy{left:22px;top:39%;width:84vw}.hero-copy h2{margin:30px 0 18px;font-size:clamp(48px,14vw,68px)}.hero-copy>p{font-size:17px}.status{gap:18px;margin-top:22px}.hero-actions{left:14px;right:14px;bottom:3vh;gap:5px}.hero-actions button{height:70px;padding:0 10px;font-size:12px}.hero-actions small{right:7px;top:8px}.timeline-ui{right:6px}.control-hint{display:none}}
@media(max-width:479px){.protocol-axis div:nth-of-type(2),.protocol-axis i:nth-of-type(2){display:none}.future-content{top:11%}.partners-panel{top:50%;height:39%}.status{font-size:9px}.opening h1{font-size:52px}}
@media(max-height:680px){.future-content{top:11%;padding:16px}.future-content h2{font-size:32px}.lead{line-height:1.45}.protocol-axis{margin-top:9px}.thesis{display:none}.market-intro{top:9%}.embedded-panel{top:29%;height:62%}.ecosystem-copy{top:9%;padding:16px}.ecosystem-copy h2{font-size:32px}.partners-panel{top:47%;height:43%}.hero-copy{top:37%}.hero-copy h2{font-size:50px;margin:22px 0 12px}.hero-copy .primary-action{margin-top:10px}.status{margin-top:14px}.hero-actions button{height:58px}}
@media(min-width:1600px){.future-content,.hero-copy{left:7vw}.market-intro{left:7vw}.embedded-panel{right:7vw;width:60vw}.ecosystem-copy{left:7vw}.partners-panel{right:7vw;width:54vw}}
@media(max-width:540px){.opening-tunnel{inset:-4% -34%;background-position:center}.opening-energy{inset:-18% -70%}.opening-copy{gap:14px}.opening h1{font-size:clamp(48px,15vw,72px)}}
@media(prefers-reduced-motion:reduce){.space-camera,.scene-image,.chapter,.opening,.opening-space,.opening-tunnel,.opening-energy,.opening-copy{transform:none!important;transition:none!important;animation:none!important;filter:none!important}.speed-lines,.opening-energy{display:none}}

/* Dense information architecture: content follows the scene instead of floating in empty frames. */
.future-content{position:absolute;left:5vw;right:7vw;top:51%;width:auto;max-width:1180px;padding:0;transform:translateY(-48%);text-shadow:0 2px 18px #020817}.future-content::before{content:"";position:absolute;inset:-28px -34px;background:linear-gradient(105deg,rgba(2,10,28,.92),rgba(3,19,46,.76) 70%,rgba(3,19,46,.15));border-left:2px solid #168bff;clip-path:polygon(0 0,96% 0,100% 18%,100% 100%,0 100%);backdrop-filter:blur(14px);z-index:-1}.future-layout{display:grid;grid-template-columns:minmax(0,1.05fr) minmax(300px,.72fr);gap:clamp(28px,5vw,76px);align-items:end;margin-top:24px}.future-kicker{margin:0 0 10px;color:#61b9ff;font:700 11px/1.2 monospace;letter-spacing:.18em}.future-content h2{margin:0 0 16px;font-size:clamp(43px,5vw,72px)}.future-content h2 em{display:block;white-space:nowrap}.future-content .lead{max-width:680px}.future-story{display:flex;flex-direction:column;gap:14px;color:#c8d8eb;font-size:14px;line-height:1.75}.future-story p{margin:0}.definition{display:flex;flex-direction:column;gap:5px;padding:15px 17px;border:1px solid rgba(96,185,255,.3);background:linear-gradient(135deg,rgba(10,45,92,.76),rgba(3,18,43,.58));box-shadow:inset 3px 0 #168bff}.definition small{color:#60b8ff;font:700 9px/1 monospace;letter-spacing:.14em}.definition strong{color:#fff;font-size:16px}.definition span{color:#91a9c4;font:600 11px/1.4 monospace}.future-content .protocol-axis{margin-top:22px}.future-content .protocol-axis div{position:relative;padding:13px 16px;background:linear-gradient(145deg,rgba(8,31,68,.86),rgba(3,17,40,.74));clip-path:polygon(10px 0,100% 0,100% calc(100% - 10px),calc(100% - 10px) 100%,0 100%,0 10px)}.future-content .protocol-axis small{color:#4dafff;font:700 9px/1 monospace}.future-content .protocol-axis b{margin-top:6px;font-size:14px}.future-footer{display:flex;align-items:center;justify-content:space-between;gap:24px;margin-top:18px}.future-footer .thesis{flex:1;margin:0}.future-footer .primary-action{flex:0 0 250px;margin:0}.primary-action{position:relative;overflow:visible;border-color:#56b4ff;background:linear-gradient(135deg,#087eff 0%,#0052ee 58%,#003db4 100%);box-shadow:0 16px 38px rgba(0,82,255,.34),inset 0 1px rgba(255,255,255,.28),inset 0 -7px rgba(0,26,102,.24);transform:perspective(500px) rotateX(-2deg)}.primary-action::before{content:"";position:absolute;inset:4px;border:1px solid rgba(194,229,255,.18);clip-path:inherit;pointer-events:none}.primary-action:hover{filter:brightness(1.12);transform:perspective(500px) rotateX(0) translateY(-3px)}.primary-action b{background:#eef8ff;color:#086df4;clip-path:polygon(10px 0,100% 0,100% calc(100% - 10px),calc(100% - 10px) 100%,0 100%,0 10px)}
.partners-panel{top:27%;height:auto;min-height:0;overflow:hidden;background:linear-gradient(135deg,rgba(3,15,37,.84),rgba(3,18,42,.5));border:0;border-left:2px solid #168bff;box-shadow:0 26px 75px rgba(0,3,18,.42);clip-path:none}.partners-panel :deep(.partners-wall){min-height:0;margin:0;padding:14px 0!important}.hero-actions{display:flex;right:auto;gap:8px}.hero-actions button{width:clamp(150px,18vw,250px);height:66px;padding:0 20px;border-color:rgba(91,181,255,.38);background:linear-gradient(145deg,rgba(6,29,67,.94),rgba(2,13,34,.94));box-shadow:inset 0 1px rgba(255,255,255,.08),0 12px 32px rgba(0,4,18,.3);font-size:15px}.hero-actions button.featured{width:clamp(190px,24vw,320px)}
@media(max-width:900px){.future-content{left:30px;right:42px;top:52%}.future-content::before{inset:-20px}.future-layout{grid-template-columns:1fr 1fr;gap:26px}.future-content h2{font-size:42px}.future-story{font-size:13px}.future-content .protocol-axis b{font-size:12px}.future-content .protocol-axis span{font-size:10px}.future-footer .primary-action{flex-basis:220px}}
@media(max-width:540px){.future-content{left:18px;right:30px;top:11%;transform:none}.future-content::before{inset:-12px;background:linear-gradient(150deg,rgba(2,10,28,.94),rgba(3,19,46,.73));clip-path:polygon(10px 0,100% 0,100% calc(100% - 10px),calc(100% - 10px) 100%,0 100%,0 10px)}.future-layout{grid-template-columns:1fr;gap:12px;margin-top:12px}.future-content h2{margin-bottom:9px;font-size:clamp(31px,8vw,39px)}.future-content .lead{font-size:13px;line-height:1.55}.future-story{display:grid;grid-template-columns:1fr 1fr;gap:8px;font-size:11px;line-height:1.45}.future-story .definition{grid-column:1/-1;grid-row:1;padding:10px 12px}.future-story p{padding:0 4px}.future-content .protocol-axis{margin-top:11px}.future-content .protocol-axis div{padding:8px}.future-content .protocol-axis b{font-size:10px}.future-content .protocol-axis span{font-size:9px}.future-footer{gap:10px;margin-top:11px}.future-footer .thesis{font-size:11px;line-height:1.45}.future-footer .primary-action{flex-basis:210px;width:210px;height:48px;font-size:13px}.partners-panel{top:51%;height:auto}.hero-actions{left:14px;right:14px;bottom:18px;display:grid;grid-template-columns:1fr 1.2fr 1fr}.hero-actions button,.hero-actions button.featured{width:auto;height:58px;padding:0 8px;font-size:11px}}
@media(max-width:479px){.future-story p:last-child{display:none}.future-story{grid-template-columns:1fr}.future-story .definition{grid-column:auto}.future-content .protocol-axis div:nth-of-type(2),.future-content .protocol-axis i:nth-of-type(2){display:block}.future-content .protocol-axis b{font-size:9px}.future-content .protocol-axis span{display:none}.future-footer{align-items:stretch;flex-direction:column}.future-footer .primary-action{flex-basis:46px}.future-footer .thesis{display:block}.opening-tunnel{background-position:52% center}}
@media(max-height:700px){.future-layout{margin-top:10px}.future-content h2{font-size:32px}.future-content .lead{line-height:1.42}.future-story{gap:7px}.future-content .protocol-axis{margin-top:8px}.future-footer{margin-top:8px}.partners-panel{top:49%}.hero-actions{bottom:10px}}

/* Narrative typography — restrained display type, readable CJK body, precise mixed-script rhythm. */
.future-content{font-family:var(--aix-font);font-kerning:normal;text-rendering:optimizeLegibility}.future-content .section-label{gap:16px}.future-content .section-label span{width:42px;height:42px;font-family:'AixDisplay',sans-serif;font-size:12px;font-weight:650;letter-spacing:.04em}.future-content .section-label small{font-family:'AixDisplay',sans-serif;font-size:10px;font-weight:620;letter-spacing:.22em}.future-kicker{font-family:'AixDisplay',sans-serif;font-size:10px;font-weight:560;letter-spacing:.2em}.future-content h2{font-family:var(--aix-font-display);font-weight:500;line-height:1.04;letter-spacing:-.055em}.future-content h2 em{font-weight:650;letter-spacing:-.07em}.future-content .lead{max-width:720px;color:#d5e1ee;font-size:clamp(14px,1.15vw,17px);font-weight:400;line-height:1.82;letter-spacing:.015em;text-wrap:pretty}.future-story{color:#c7d5e5;font-size:clamp(13px,1vw,15px);font-weight:350;line-height:1.78;letter-spacing:.01em}.definition{gap:7px;padding:17px 19px}.definition small{font-family:'AixDisplay',sans-serif;font-size:9px;font-weight:650;letter-spacing:.18em}.definition strong{font-family:var(--aix-font-display);font-size:clamp(15px,1.25vw,18px);font-weight:560;line-height:1.35;letter-spacing:-.02em}.definition span{font-family:'AixSans',sans-serif;font-size:11px;font-weight:550;letter-spacing:.045em}.future-content .protocol-axis small{font-family:'AixDisplay',sans-serif;font-weight:600;letter-spacing:.12em}.future-content .protocol-axis b{font-family:var(--aix-font-display);font-weight:520;line-height:1.35;letter-spacing:-.015em}.future-content .protocol-axis span{font-weight:350;letter-spacing:.04em}.future-footer .thesis{color:#d3dfed;font-weight:400;letter-spacing:.02em}.future-footer .primary-action span{font-family:var(--aix-font);font-size:15px;font-weight:500;letter-spacing:.045em}.future-footer .primary-action b{font-family:'AixDisplay',sans-serif;font-weight:450}
@media(max-width:900px){.future-content .section-label span{width:38px;height:38px}.future-content h2{font-weight:520;letter-spacing:-.045em}.future-content .lead{line-height:1.7}.future-story{line-height:1.65}}
@media(max-width:540px){.future-content .section-label{gap:12px}.future-content .section-label span{width:34px;height:34px}.future-content .section-label small{font-size:8px;letter-spacing:.14em}.future-content h2{font-size:clamp(30px,8vw,38px);font-weight:520;line-height:1.08;letter-spacing:-.04em}.future-content h2 em{letter-spacing:-.05em}.future-content .lead{font-size:13px;line-height:1.68;letter-spacing:0}.future-story{font-size:11px;line-height:1.58}.definition{gap:4px;padding:10px 12px}.definition strong{font-size:13px}.definition span{font-size:9px}.future-footer .primary-action span{font-size:13px}}

/* Opening is a real timeline node: ambient while parked, cinematic only during travel. */
.opening{opacity:1;visibility:visible;background:#020711;transition:opacity .55s ease,visibility .55s}.opening-space{animation:none}.opening-tunnel{inset:-5%;background-image:url('/assets/timeline-00-time-tunnel-v2.png');background-position:center;background-size:cover;filter:saturate(.9) contrast(1.08) brightness(.78);animation:opening-breathe 9s ease-in-out infinite alternate}.opening-energy{opacity:.2;animation:opening-orbit 18s linear infinite}.opening-vignette{background:radial-gradient(circle at 50% 48%,transparent 8%,rgba(1,8,24,.12) 34%,rgba(1,6,18,.84) 100%),linear-gradient(90deg,rgba(1,7,20,.42),transparent 42%,rgba(1,7,20,.18))}.opening-copy{align-items:flex-start;width:min(82vw,760px);gap:0;padding:36px;text-align:left;animation:none}.opening-copy>p{margin:0 0 22px;color:#7ec5ff;font-family:'AixDisplay',sans-serif;font-size:10px;font-weight:550;line-height:1;letter-spacing:.34em;text-shadow:none}.opening-copy h1{display:flex;flex-direction:column;margin:0;color:#f4f8fd;font-family:var(--aix-font-display);font-size:clamp(58px,8.2vw,116px);font-weight:400;line-height:.94;letter-spacing:-.065em;text-shadow:0 8px 44px rgba(0,2,12,.9)}.opening-copy h1 span{width:auto;height:auto;margin-bottom:10px;background:none;color:#8fabc7;font-family:var(--aix-font);font-size:.23em;font-weight:300;line-height:1;letter-spacing:.34em;box-shadow:none;animation:none}.opening-enter{display:flex;align-items:center;justify-content:space-between;width:238px;height:58px;margin-top:38px;padding:0 8px 0 22px;border:1px solid rgba(94,184,255,.52);background:rgba(4,25,57,.72);color:#eff8ff;font-family:var(--aix-font);font-size:14px;font-weight:450;letter-spacing:.1em;cursor:pointer;clip-path:polygon(12px 0,100% 0,100% calc(100% - 12px),calc(100% - 12px) 100%,0 100%,0 12px);backdrop-filter:blur(14px);box-shadow:inset 0 1px rgba(255,255,255,.08),0 18px 48px rgba(0,6,22,.4);transition:transform .25s,background .25s,border-color .25s}.opening-enter:hover{transform:translateY(-3px);border-color:#8ed0ff;background:rgba(7,61,126,.8)}.opening-enter b{display:grid;width:42px;height:42px;place-items:center;background:#eaf6ff;color:#086ef4;font-family:'AixDisplay',sans-serif;font-size:19px;font-weight:450;clip-path:polygon(8px 0,100% 0,100% calc(100% - 8px),calc(100% - 8px) 100%,0 100%,0 8px)}.opening-copy>small{margin-top:15px;color:#67819d;font-family:'AixDisplay',sans-serif;font-size:8px;font-weight:550;letter-spacing:.22em}.intro-complete .opening{opacity:0;visibility:hidden;pointer-events:none}.intro-complete .opening-tunnel{transform:none}.timeline-ui i{height:20%;transform:translateY(calc(var(--node) * 100%))}
@keyframes opening-breathe{from{transform:scale(1.02);filter:saturate(.86) contrast(1.08) brightness(.72)}to{transform:scale(1.08);filter:saturate(.96) contrast(1.12) brightness(.84)}}
@keyframes opening-orbit{to{transform:rotate(360deg)}}
@media(max-width:540px){.opening-tunnel{inset:-4% -28%;background-position:50% center}.opening-copy{width:100%;padding:28px}.opening-copy h1{font-size:clamp(54px,17vw,76px)}.opening-copy h1 span{font-size:15px}.opening-enter{width:210px;height:54px;margin-top:30px}}

/* Terminal finale: one centered command deck, no dead zone. */
.chapter--hero::before{content:"";position:absolute;inset:0;background:linear-gradient(90deg,rgba(1,7,20,.88) 0%,rgba(1,9,25,.56) 52%,rgba(1,9,25,.14) 100%),linear-gradient(180deg,rgba(1,7,20,.2),transparent 45%,rgba(1,7,20,.58));pointer-events:none}.hero-command{position:absolute;left:max(5vw,34px);top:50%;width:min(930px,82vw);transform:translateY(-46%);perspective:1200px}.hero-copy{position:relative;left:auto;top:auto;width:auto;transform:none}.eyebrow{gap:14px;font-family:'AixDisplay',sans-serif;font-size:9px;font-weight:560;letter-spacing:.28em}.eyebrow i{width:44px;height:2px}.hero-copy h2{margin:24px 0 14px;font-family:var(--aix-font-display);font-size:clamp(54px,6.4vw,94px);font-weight:420;line-height:.92;letter-spacing:-.065em}.hero-copy h2 em{display:inline;color:#76beff;font-style:normal;font-weight:600}.hero-copy>p{margin:0;color:#aebfd2;font-size:clamp(14px,1.5vw,19px);font-weight:350;letter-spacing:.11em}.status{width:min(620px,100%);gap:34px;margin-top:22px;padding-top:13px;font-family:'AixDisplay',sans-serif;font-size:9px;font-weight:560}.status span{display:flex;align-items:center;gap:8px}.status span i{width:6px;height:6px;border-radius:50%;background:#38d6b4;box-shadow:0 0 14px #38d6b4}.status strong{margin-left:4px;letter-spacing:.15em}.hero-actions{position:relative;left:auto;right:auto;bottom:auto;display:grid;grid-template-columns:.86fr 1.25fr .96fr;gap:14px;margin-top:28px;transform-style:preserve-3d}.hero-actions button{position:relative;display:flex;height:104px;align-items:flex-start;justify-content:flex-end;flex-direction:column;gap:8px;padding:18px 62px 18px 20px;overflow:visible;border:1px solid rgba(103,188,255,.48);background:linear-gradient(145deg,rgba(9,39,79,.96),rgba(2,15,37,.98));color:#eef7ff;text-align:left;font-family:var(--aix-font);font-weight:450;cursor:pointer;clip-path:polygon(14px 0,calc(100% - 8px) 0,100% 8px,100% calc(100% - 16px),calc(100% - 16px) 100%,0 100%,0 14px);box-shadow:0 16px 0 -9px rgba(0,35,85,.95),0 24px 42px rgba(0,3,17,.56),inset 0 1px rgba(255,255,255,.14),inset 0 -12px rgba(0,8,28,.34);transform:rotateX(2deg) translateZ(0);transition:transform .25s ease,filter .25s ease,border-color .25s ease}.hero-actions button::before{content:"";position:absolute;left:8px;right:8px;top:7px;height:1px;background:linear-gradient(90deg,#79c7ff,transparent)}.hero-actions button:hover{z-index:2;filter:brightness(1.15);border-color:#8ed1ff;transform:rotateX(0) translateY(-6px) translateZ(18px)}.hero-actions button.featured{background:linear-gradient(145deg,#1495ff 0%,#075dea 56%,#063795 100%);border-color:#8bd3ff;box-shadow:0 18px 0 -9px #062766,0 30px 55px rgba(0,74,220,.42),inset 0 1px rgba(255,255,255,.35),inset 0 -15px rgba(0,30,115,.28);transform:rotateX(2deg) translateY(-5px) translateZ(14px)}.hero-actions button.featured:hover{transform:rotateX(0) translateY(-10px) translateZ(28px)}.hero-actions small{position:static;color:#70baff;font-family:'AixDisplay',sans-serif;font-size:8px;font-weight:600;letter-spacing:.18em}.hero-actions .featured small{color:#d4edff}.hero-actions button>span{font-family:var(--aix-font-display);font-size:clamp(14px,1.35vw,18px);font-weight:520;letter-spacing:-.01em}.hero-actions button>b{position:absolute;right:14px;bottom:14px;display:grid;width:38px;height:38px;place-items:center;background:rgba(226,245,255,.96);color:#0873ee;font-family:'AixDisplay',sans-serif;font-size:17px;font-weight:450;clip-path:polygon(8px 0,100% 0,100% calc(100% - 8px),calc(100% - 8px) 100%,0 100%,0 8px);box-shadow:inset 0 -5px rgba(115,187,235,.2)}.hero-primary{width:224px;height:54px;margin-top:18px}.hero-primary span{font-size:13px}.hero-primary b{width:40px;height:40px}
/* Shared dimensional controls: every homepage CTA uses physical depth. */
.primary-action,.opening-enter{border-bottom-color:#00419e;box-shadow:0 10px 0 -6px #031c50,0 20px 42px rgba(0,31,105,.44),inset 0 1px rgba(255,255,255,.32),inset 0 -9px rgba(0,24,99,.3);transform:perspective(600px) rotateX(2deg);transition:transform .22s ease,filter .22s ease,box-shadow .22s ease}.primary-action:hover,.opening-enter:hover{filter:brightness(1.12);transform:perspective(600px) rotateX(0) translateY(-5px);box-shadow:0 14px 0 -7px #031c50,0 26px 50px rgba(0,55,190,.46),inset 0 1px rgba(255,255,255,.36),inset 0 -8px rgba(0,24,99,.22)}.primary-action:active,.opening-enter:active,.hero-actions button:active{transform:translateY(3px);filter:brightness(.96)}
@media(max-width:540px){.hero-command{left:22px;right:30px;top:52%;width:auto;transform:translateY(-48%)}.hero-copy h2{margin:18px 0 10px;font-size:clamp(44px,13vw,62px)}.hero-copy>p{font-size:13px}.status{gap:16px;margin-top:14px}.hero-actions{grid-template-columns:1fr;gap:8px;margin-top:18px}.hero-actions button,.hero-actions button.featured{height:66px;padding:11px 54px 11px 15px;transform:none}.hero-actions button>span{font-size:14px}.hero-actions button>b{right:10px;bottom:10px;width:34px;height:34px}.hero-actions small{font-size:7px}.hero-primary{width:190px;height:48px;margin-top:12px}}
@media(max-height:700px){.hero-command{top:53%}.hero-copy h2{margin:14px 0 8px;font-size:45px}.hero-copy>p{font-size:12px}.status{margin-top:10px;padding-top:9px}.hero-actions{margin-top:14px}.hero-actions button,.hero-actions button.featured{height:58px}.hero-primary{height:44px;margin-top:9px}}

/* Responsive narrative system: spacious desktop, edited tablet, readable mobile. */
.future-content::before{inset:-36px -44px;background:linear-gradient(108deg,rgba(2,10,28,.84),rgba(3,19,46,.62) 68%,rgba(3,19,46,.08));border-left-width:1px;backdrop-filter:blur(11px)}.future-content .section-label span{width:38px;height:38px;background:linear-gradient(145deg,#168bff,#0758c9);font-size:10px;font-weight:560;box-shadow:0 8px 22px rgba(0,86,220,.28)}.future-content .section-label small{font-size:9px;font-weight:520;letter-spacing:.25em}.future-layout{align-items:start;gap:clamp(48px,6vw,96px);margin-top:32px}.future-kicker{margin-bottom:16px;font-size:9px;font-weight:500;letter-spacing:.28em}.future-content h2{margin-bottom:24px;font-size:clamp(42px,4.35vw,66px);font-weight:430;line-height:1.08;letter-spacing:-.045em}.future-content h2 em{margin-top:5px;font-weight:560;letter-spacing:-.055em}.future-content .lead{max-width:660px;font-size:clamp(14px,1.05vw,16px);font-weight:350;line-height:1.9;letter-spacing:.025em}.future-story{gap:20px;font-size:clamp(13px,.94vw,14px);font-weight:320;line-height:1.85}.definition{gap:8px;padding:18px 20px;background:linear-gradient(135deg,rgba(8,41,85,.68),rgba(3,18,43,.4));box-shadow:inset 1px 0 #168bff}.definition strong{font-size:clamp(15px,1.15vw,17px);font-weight:480}.future-content .protocol-axis{gap:14px;margin-top:30px}.future-content .protocol-axis div{padding:16px 18px}.future-content .protocol-axis b{margin-top:8px;font-size:13px;font-weight:460}.future-content .protocol-axis span{margin-top:7px;font-size:10px}.future-footer{gap:36px;margin-top:25px}.future-footer .thesis{font-size:13px;font-weight:350;line-height:1.7}.future-footer .primary-action{flex-basis:230px;width:230px;height:58px}.future-footer .primary-action span{font-size:13px;font-weight:430}
@media(min-width:1440px){.future-content{left:8vw;right:11vw;max-width:1380px}.future-content::before{inset:-48px -58px}.future-layout{grid-template-columns:minmax(0,1.12fr) minmax(330px,.68fr);gap:110px;margin-top:38px}.future-content h2{font-size:68px}.future-content .lead{font-size:16px}.future-content .protocol-axis{margin-top:36px}.future-footer{margin-top:30px}}
@media(min-width:1024px) and (max-width:1439px){.future-content{left:6vw;right:9vw;max-width:1120px}.future-layout{grid-template-columns:minmax(0,1.05fr) minmax(300px,.72fr);gap:48px}.future-content h2{font-size:clamp(44px,4.5vw,60px)}.future-story{font-size:13px}.future-content .protocol-axis{margin-top:25px}}
@media(min-width:760px) and (max-width:1023px){.future-content{left:5vw;right:7vw;top:51%;max-width:none}.future-content::before{inset:-28px}.future-layout{grid-template-columns:1.08fr .92fr;gap:32px;margin-top:25px}.future-content h2{font-size:clamp(38px,5.4vw,50px)}.future-content .lead{font-size:14px;line-height:1.75}.future-story{gap:14px;font-size:12px;line-height:1.68}.definition{padding:14px 16px}.future-content .protocol-axis{gap:8px;margin-top:22px}.future-content .protocol-axis div{padding:12px}.future-footer{margin-top:19px}}
@media(min-width:480px) and (max-width:759px){.future-content{left:30px;right:48px;top:50%;transform:translateY(-47%)}.future-content::before{inset:-24px -20px;background:linear-gradient(112deg,rgba(2,10,28,.86),rgba(3,19,46,.52) 78%,transparent);clip-path:polygon(0 0,96% 0,100% 7%,100% 100%,0 100%)}.future-content .section-label{gap:15px}.future-content .section-label span{width:36px;height:36px}.future-content .section-label small{font-size:8px;letter-spacing:.2em}.future-layout{display:block;margin-top:24px}.future-kicker{margin-bottom:13px}.future-content h2{margin-bottom:20px;font-size:clamp(34px,7.1vw,43px);font-weight:420;line-height:1.08}.future-content .lead{max-width:530px;font-size:13px;line-height:1.75}.future-story{display:block;margin-top:22px}.future-story>p{display:none}.future-story .definition{display:flex;max-width:520px;padding:14px 17px}.definition strong{font-size:14px}.future-content .protocol-axis{gap:8px;margin-top:22px}.future-content .protocol-axis div{padding:12px 13px}.future-content .protocol-axis b{font-size:11px}.future-content .protocol-axis span{font-size:9px}.future-footer{align-items:center;gap:18px;margin-top:20px}.future-footer .thesis{font-size:11px;line-height:1.55}.future-footer .primary-action{flex-basis:190px;width:190px;height:52px}.future-footer .primary-action span{font-size:12px}}
@media(max-width:479px){.future-content{left:22px;right:38px;top:50%;transform:translateY(-46%)}.future-content::before{inset:-18px -14px}.future-content .section-label span{width:32px;height:32px}.future-content .section-label small{max-width:210px;font-size:7px;line-height:1.5}.future-layout{display:block;margin-top:18px}.future-kicker{margin-bottom:10px;font-size:8px}.future-content h2{margin-bottom:14px;font-size:clamp(31px,10.5vw,40px);font-weight:420}.future-content .lead{font-size:12px;line-height:1.65}.future-story{display:block;margin-top:15px}.future-story>p{display:none}.future-story .definition{display:flex;padding:11px 13px}.definition strong{font-size:12px}.definition span{font-size:8px}.future-content .protocol-axis{gap:5px;margin-top:15px}.future-content .protocol-axis div{padding:9px 8px}.future-content .protocol-axis i{font-size:10px}.future-content .protocol-axis b{font-size:9px}.future-content .protocol-axis span{display:none}.future-footer{align-items:stretch;flex-direction:column;gap:12px;margin-top:15px}.future-footer .thesis{font-size:10px;line-height:1.55}.future-footer .primary-action{flex-basis:48px;width:190px;height:48px}.future-footer .primary-action span{font-size:11px}}
@media(max-height:680px) and (min-width:480px){.future-content{top:52%}.future-content .section-label span{width:30px;height:30px}.future-layout{margin-top:14px}.future-kicker{margin-bottom:8px}.future-content h2{margin-bottom:11px;font-size:32px}.future-content .lead{line-height:1.55}.future-story{margin-top:12px}.future-story .definition{padding:9px 12px}.future-content .protocol-axis{margin-top:11px}.future-footer{margin-top:11px}.future-footer .thesis{display:none}}

/* Layout correction: preserve the original composition and refine only its finish. */
.opening-copy{box-sizing:border-box;width:min(86vw,700px);padding:clamp(20px,4vw,36px)}.opening-copy>p{max-width:100%;font-size:clamp(8px,1.6vw,10px);line-height:1.4;letter-spacing:clamp(.18em,.7vw,.34em);white-space:nowrap}.opening-copy h1{width:100%;font-size:clamp(46px,9vw,104px);line-height:1;letter-spacing:-.045em;white-space:normal}.opening-copy h1 span{font-size:clamp(13px,2vw,20px);letter-spacing:.28em}.opening-enter{width:220px;height:56px;margin-top:30px;border:1px solid rgba(104,190,255,.62);background:linear-gradient(180deg,rgba(9,48,96,.92),rgba(3,24,59,.96));box-shadow:0 8px 0 -4px #031b49,0 18px 36px rgba(0,11,38,.42),inset 0 1px rgba(255,255,255,.16);transform:none}.opening-enter:hover{transform:translateY(-3px)}
.chapter--hero::before{background:linear-gradient(90deg,rgba(1,7,20,.9),rgba(1,9,25,.42) 58%,rgba(1,9,25,.08))}.hero-copy{position:absolute;left:5vw;top:50%;width:min(670px,58vw);transform:translateY(-50%)}.hero-copy h2{margin:30px 0 18px;font-size:clamp(54px,6.5vw,94px);font-weight:430;line-height:.94}.hero-copy>p{font-size:clamp(15px,1.5vw,21px)}.hero-primary{width:220px;height:56px;margin-top:24px}.status{width:min(620px,100%);gap:34px;margin-top:26px;padding-top:14px}.hero-actions{position:absolute;left:5vw;right:5vw;bottom:4vh;display:grid;grid-template-columns:1fr 1.18fr 1fr;gap:14px;margin:0;perspective:900px}.hero-actions button,.hero-actions button.featured{box-sizing:border-box;width:auto;height:78px;padding:16px 54px 14px 20px;justify-content:center;gap:5px;border:1px solid rgba(103,188,255,.4);background:linear-gradient(180deg,rgba(8,35,75,.94),rgba(3,17,42,.98));box-shadow:0 9px 0 -5px rgba(0,34,82,.92),0 18px 34px rgba(0,3,17,.38),inset 0 1px rgba(255,255,255,.12);transform:none;clip-path:polygon(10px 0,100% 0,100% calc(100% - 10px),calc(100% - 10px) 100%,0 100%,0 10px)}.hero-actions button.featured{background:linear-gradient(180deg,#168cf7,#0756d8);box-shadow:0 9px 0 -5px #052e7d,0 20px 38px rgba(0,64,190,.36),inset 0 1px rgba(255,255,255,.28)}.hero-actions button:hover,.hero-actions button.featured:hover{transform:translateY(-4px);filter:brightness(1.1)}.hero-actions small{position:absolute;left:18px;top:12px;color:#6fbcff;font-size:8px}.hero-actions button>span{font-size:clamp(13px,1.25vw,17px);font-weight:460}.hero-actions button>b{right:13px;bottom:13px;width:38px;height:38px}.primary-action,.opening-enter{transform:none}.primary-action:hover,.opening-enter:hover{transform:translateY(-3px)}
@media(min-width:760px) and (max-width:1023px){.hero-copy{left:6vw;top:39%;width:72vw}.hero-copy h2{font-size:clamp(48px,7.4vw,66px)}.hero-actions{left:4vw;right:6vw}.hero-actions button,.hero-actions button.featured{height:72px}}
@media(max-width:540px){.opening-copy{width:100%;padding:24px 28px}.opening-copy>p{font-size:8px;letter-spacing:.2em}.opening-copy h1{font-size:clamp(42px,13.5vw,66px);line-height:1.03}.opening-copy h1 span{font-size:13px}.opening-enter{width:194px;height:50px;margin-top:25px}.opening-copy>small{font-size:7px;letter-spacing:.16em}.hero-copy{left:22px;right:34px;top:34%;width:auto}.hero-copy h2{margin:18px 0 10px;font-size:clamp(42px,11vw,58px)}.hero-copy>p{font-size:13px}.hero-primary{width:190px;height:48px;margin-top:16px}.status{gap:16px;margin-top:16px}.hero-actions{left:16px;right:28px;bottom:20px;grid-template-columns:1fr 1.16fr 1fr;gap:6px}.hero-actions button,.hero-actions button.featured{height:60px;padding:15px 34px 8px 10px}.hero-actions small{left:9px;top:7px;font-size:7px}.hero-actions button>span{font-size:10px}.hero-actions button>b{right:7px;bottom:7px;width:28px;height:28px;font-size:13px}}
@media(max-width:479px){.opening-copy{padding:22px}.opening-copy h1{font-size:clamp(40px,14vw,56px)}.hero-copy{top:32%}.hero-actions{left:10px;right:20px;bottom:14px}.hero-actions button,.hero-actions button.featured{height:56px}.hero-actions button>span{font-size:9px}.status{font-size:8px}}
@media(max-height:680px){.opening-copy h1{font-size:clamp(38px,10vh,58px)}.opening-copy>p{margin-bottom:14px}.opening-enter{margin-top:20px}.hero-copy{top:31%}.hero-copy h2{font-size:42px;margin:12px 0 8px}.status{margin-top:10px}.hero-actions{bottom:10px}.hero-actions button,.hero-actions button.featured{height:52px}}

/* Approved flat controls: blue CTA with white arrow block, flat three-card navigation. */
.primary-action{position:relative;display:flex;align-items:center;justify-content:space-between;box-sizing:border-box;width:286px;height:72px;margin-top:24px;padding:0 10px 0 28px;border:2px solid #126cff;border-radius:0;background:#126cff;color:#fff;clip-path:none;box-shadow:none;transform:none}.primary-action::before{content:none}.primary-action span{width:auto;height:auto;white-space:nowrap;background:none;color:#fff;font-size:18px;font-weight:650;box-shadow:none;animation:none}.primary-action b{display:grid;flex:0 0 60px;width:60px;height:56px;place-items:center;background:#f4f7fb;color:#086dff;font-size:22px;clip-path:none}.primary-action:hover{background:#075be8;border-color:#075be8;box-shadow:none;transform:none}.future-footer .primary-action{width:214px;min-width:214px;height:56px;flex:0 0 214px;margin:0;padding:0 7px 0 20px}.future-footer .primary-action span{font-size:16px;font-weight:650}.future-footer .primary-action b{flex-basis:42px;width:42px;height:42px;font-size:19px}.hero-primary{width:286px;height:72px;margin-top:24px}.hero-actions{position:relative;left:auto;right:auto;bottom:auto;width:min(1120px,90vw);display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;margin-top:24px}.hero-actions button,.hero-actions button.featured{position:relative;box-sizing:border-box;width:auto;height:78px;padding:16px 22px 14px;border:1px solid rgba(196,210,228,.72);border-radius:0;background:rgba(248,251,255,.96);color:#07101e;clip-path:none;box-shadow:none;transform:none;backdrop-filter:none}.hero-actions button::before{content:none}.hero-actions button.featured{border-color:#126cff;background:#126cff;color:#fff;box-shadow:none}.hero-actions small{position:static;order:2;color:inherit;font-size:12px;font-weight:700;line-height:1}.hero-actions button>span{order:1;font-size:21px;font-weight:700;line-height:1.2}.hero-actions button,.hero-actions button.featured{align-items:flex-start;justify-content:center;gap:8px;text-align:left}.hero-actions button>b{display:none}.hero-actions button:hover,.hero-actions button.featured:hover{filter:brightness(.96);transform:none}
@media(max-width:540px){.hero-copy{top:50%}.primary-action,.hero-primary{width:230px;height:58px;padding-left:20px}.primary-action span{font-size:15px}.primary-action b{flex-basis:48px;width:48px;height:42px;font-size:18px}.future-footer .primary-action{width:194px;min-width:194px;height:50px;flex-basis:194px;padding-left:16px}.future-footer .primary-action span{font-size:15px}.future-footer .primary-action b{flex-basis:38px;width:38px;height:36px;font-size:17px}.hero-actions{left:auto;right:auto;bottom:auto;width:calc(100vw - 50px);gap:6px;margin-top:16px}.hero-actions button,.hero-actions button.featured{height:62px;padding:12px 10px 10px}.hero-actions small{font-size:9px}.hero-actions button>span{font-size:14px}.hero-actions button,.hero-actions button.featured{gap:5px}}
@media(max-width:479px){.hero-actions button>span{font-size:12px}.hero-actions{margin-top:12px}}
@media(max-height:680px){.hero-primary{height:52px;margin-top:10px}.primary-action b{height:38px}.hero-actions{margin-top:10px}.hero-actions button,.hero-actions button.featured{height:58px}}

/* Mobile fit: compact every scene and keep all controls clear of adjacent content. */
@media(max-width:540px){.timeline{touch-action:none}.future-content{top:8%;left:14px;right:24px;padding:14px}.future-content::before{inset:-8px}.future-content .section-label span{width:32px;height:32px}.future-content .section-label small{font-size:8px}.future-kicker{margin-bottom:6px;font-size:8px}.future-content h2{margin-bottom:7px;font-size:clamp(26px,7.6vw,34px);line-height:1}.future-content .lead{font-size:11px;line-height:1.45}.future-layout{gap:8px;margin-top:8px}.future-story{gap:6px;font-size:10px}.future-story .definition{padding:7px 9px}.future-story .definition strong{font-size:12px}.future-content .protocol-axis{margin-top:7px}.future-content .protocol-axis div{padding:6px}.future-content .protocol-axis b{font-size:9px}.future-footer{align-items:center;flex-direction:row;gap:8px;margin-top:7px}.future-footer .thesis{font-size:9px;line-height:1.35}.future-footer .primary-action{width:160px;min-width:160px;height:42px;flex:0 0 160px;padding:0 5px 0 12px}.future-footer .primary-action span{font-size:12px}.future-footer .primary-action b{width:32px;height:30px;flex-basis:32px;font-size:15px}.ecosystem-copy{top:8%;padding:14px}.ecosystem-copy h2{font-size:30px}.ecosystem-copy>p{font-size:11px}.ecosystem-copy dl{gap:5px;margin-top:8px}.ecosystem-copy dl div{padding:7px}.partners-panel{top:42%;height:45%}.hero-copy{top:48%;left:18px;right:26px;width:auto}.hero-copy h2{font-size:clamp(38px,11vw,52px)}.hero-primary{width:200px;height:50px}.hero-actions{width:calc(100vw - 44px)}.hero-actions button,.hero-actions button.featured{height:56px}.hero-actions button>span{font-size:11px}.hero-actions small{font-size:8px}}
@media(max-width:479px){.future-story p{display:none}.future-content .protocol-axis{gap:3px}.future-content .protocol-axis i{font-size:9px}.future-footer .thesis{display:block}.market-intro{top:7%}.embedded-panel{top:27%;height:61%}.partners-panel{top:40%;height:47%}.hero-copy{top:47%}}

/* Final mobile composition: every scene fits below the fixed 60px header. */
@media(max-width:540px){.chapter{inset:60px 0 0;overflow:hidden}.future-content,.ecosystem-copy,.market-intro,.embedded-panel,.partners-panel{box-sizing:border-box;max-width:none}.future-content{top:16px;right:24px;bottom:auto;left:16px;width:auto;padding:10px;transform:none}.future-content::before{inset:-6px}.future-content .section-label{gap:10px}.future-content .section-label span{width:28px;height:28px}.future-content .section-label small{font-size:7px}.future-layout{margin-top:7px}.future-kicker{margin-bottom:4px}.future-content h2{margin-bottom:5px;font-size:clamp(25px,7vw,32px)}.future-content .lead{font-size:10px;line-height:1.35}.future-story{margin-top:6px}.future-story .definition{padding:6px 8px}.future-story .definition span{display:none}.future-content .protocol-axis{margin-top:6px}.future-content .protocol-axis div{padding:5px}.future-content .protocol-axis b{margin-top:3px;font-size:8px}.future-footer{gap:6px;margin-top:6px}.future-footer .thesis{font-size:8px;line-height:1.25}.future-footer .primary-action{width:148px;min-width:148px;height:38px;flex-basis:148px}.future-footer .primary-action span{font-size:11px}.market-intro{top:12px;right:24px;left:16px;width:auto;padding:10px}.market-intro .section-label span{width:28px;height:28px}.market-intro h2{margin:6px 0 3px;font-size:25px}.market-intro p{font-size:10px;line-height:1.3}.embedded-panel{top:128px;right:24px;bottom:12px;left:16px;width:auto;height:auto}.embedded-panel :deep(.market-section){box-sizing:border-box;width:100%;height:100%;padding:8px!important}.embedded-panel :deep(.market-header h2){font-size:20px}.embedded-panel :deep(.price-block strong){font-size:18px}.embedded-panel :deep(.market-stats){margin-top:7px}.embedded-panel :deep(.market-stats>div){padding:5px 4px}.embedded-panel :deep(.toolbar){min-height:34px}.embedded-panel :deep(.terminal){min-height:0;margin-top:6px}.embedded-panel :deep(.chart-wrap),.embedded-panel :deep(.chart){min-height:0;height:100%}.ecosystem-copy{top:12px;right:24px;left:16px;width:auto;padding:10px;transform:none}.ecosystem-copy .section-label span{width:28px;height:28px}.ecosystem-copy h2{margin:8px 0 6px;font-size:clamp(25px,7vw,31px)}.ecosystem-copy>p{font-size:10px;line-height:1.35}.ecosystem-copy dl{margin-top:7px}.ecosystem-copy dl div{padding:6px}.ecosystem-copy dt{font-size:8px}.ecosystem-copy dd{margin-top:4px;font-size:10px}.partners-panel{top:218px;right:24px;bottom:12px;left:16px;width:auto;height:auto}.partners-panel :deep(.partners-wall){box-sizing:border-box;width:100%;height:100%;margin:0;padding:4px 0!important}.partners-panel :deep(.wall-row .wall-track){gap:10px}.partners-panel :deep(.wall-row .wall-item){min-width:72px;min-height:40px;padding:5px 7px}.partners-panel :deep(.wall-row .wall-item>img){max-width:105px;max-height:24px}.partners-panel :deep(.brand-item){min-width:112px}.partners-panel :deep(.brand-name){font-size:11px}.hero-copy{top:50%;right:28px;left:18px;width:auto}.hero-actions{width:100%}}
@media(max-width:479px){.future-content .lead{max-height:42px;overflow:hidden}.future-footer .thesis{display:none}.market-intro p{max-height:27px;overflow:hidden}.embedded-panel{top:120px}.partners-panel{top:205px}}
@media(max-height:700px) and (max-width:540px){.future-content{top:8px}.future-content .lead{max-height:28px;overflow:hidden}.future-content .protocol-axis span{display:none}.market-intro,.ecosystem-copy{top:8px}.embedded-panel{top:112px;bottom:8px}.partners-panel{top:190px;bottom:8px}.hero-copy{top:49%}}

/* Narrow desktop/tablet: one-column scenes prevent the desktop panels from overlapping. */
@media(min-width:541px) and (max-width:900px){.chapter{inset:60px 0 0;overflow:hidden}.future-content{top:50%;right:30px;left:30px;width:auto;padding:0;transform:translateY(-50%)}.future-content::before{inset:-18px}.future-layout{grid-template-columns:minmax(0,1.08fr) minmax(220px,.72fr);gap:20px;margin-top:14px}.future-content h2{font-size:38px}.future-content .lead{font-size:12px;line-height:1.5}.future-story{gap:8px;font-size:11px;line-height:1.45}.future-story .definition{padding:9px 11px}.future-content .protocol-axis{margin-top:12px}.future-content .protocol-axis div{padding:8px 9px}.future-footer{gap:12px;margin-top:12px}.future-footer .thesis{font-size:10px;line-height:1.4}.future-footer .primary-action{width:200px;min-width:200px;height:46px;flex-basis:200px;padding-left:14px}.future-footer .primary-action span{font-size:13px;letter-spacing:0;white-space:nowrap}.future-footer .primary-action b{width:34px;height:32px;flex-basis:34px}.market-intro{top:18px;right:30px;left:30px;width:auto;padding:14px 18px}.market-intro .section-label span{width:30px;height:30px}.market-intro h2{margin:8px 0 4px;font-size:30px}.market-intro p{font-size:11px;line-height:1.4}.embedded-panel{top:164px;right:30px;bottom:16px;left:30px;width:auto;height:auto}.embedded-panel :deep(.market-section){padding:10px!important}.embedded-panel :deep(.market-header h2){font-size:25px}.embedded-panel :deep(.price-block strong){font-size:24px}.embedded-panel :deep(.market-stats){margin-top:8px}.embedded-panel :deep(.market-stats>div){padding:8px}.embedded-panel :deep(.terminal){margin-top:8px}.embedded-panel :deep(.toolbar){min-height:38px}.embedded-panel :deep(.chart-wrap),.embedded-panel :deep(.chart){min-height:0;height:100%}.ecosystem-copy{top:18px;right:30px;left:30px;width:auto;padding:14px 18px;transform:none}.ecosystem-copy .section-label span{width:30px;height:30px}.ecosystem-copy h2{margin:7px 0 4px;font-size:30px;line-height:.98}.ecosystem-copy>p{font-size:10px;line-height:1.35}.ecosystem-copy dl{gap:6px;margin-top:7px}.ecosystem-copy dl div{padding:5px 8px}.ecosystem-copy dd{margin-top:3px;font-size:9px}.partners-panel{top:242px;right:30px;bottom:16px;left:30px;width:auto;height:auto}.partners-panel :deep(.partners-wall){box-sizing:border-box;width:100%;height:100%;display:flex;flex-direction:column;justify-content:space-evenly;padding:6px 0!important}.partners-panel :deep(.wall-row){display:flex;align-items:center;min-height:0;flex:1}.partners-panel :deep(.wall-row .wall-track){gap:14px}.partners-panel :deep(.wall-row .wall-item){min-width:76px;min-height:42px;padding:5px 8px}.partners-panel :deep(.wall-row .wall-item>img){max-width:112px;max-height:25px}.partners-panel :deep(.brand-item){min-width:122px}.partners-panel :deep(.brand-name){font-size:12px}.hero-copy{top:50%;right:30px;left:30px;width:auto}.hero-copy h2{margin:24px 0 14px;font-size:54px}.hero-copy>p{font-size:16px}.hero-primary{width:184px;height:46px;margin-top:12px;padding:0 6px 0 16px}.hero-primary span{font-size:15px}.hero-primary b{width:36px;height:34px;flex-basis:36px;font-size:17px}.status{gap:24px;margin-top:16px}.hero-actions{width:100%;grid-template-columns:repeat(3,minmax(0,1fr));gap:8px;margin-top:14px}.hero-actions button,.hero-actions button.featured{min-width:0;height:62px;padding:10px 12px}.hero-actions button>span{font-size:14px;line-height:1.15;white-space:nowrap}.hero-actions small{font-size:9px}}
@media(min-width:541px) and (max-width:900px) and (max-height:700px){.market-intro{top:10px}.embedded-panel{top:142px;bottom:10px}.ecosystem-copy{top:8px}.partners-panel{top:218px;bottom:10px}.hero-copy{top:49%}.hero-copy h2{font-size:48px}.hero-actions button,.hero-actions button.featured{height:56px}}

/* Single source of truth for phones. Keep each active scene inside the visible stage. */
@media(max-width:540px){.chapter{inset:60px 0 0;width:auto;height:auto;overflow:hidden}.chapter.active{transform:none}.future-content{top:18px;right:30px;left:20px;width:auto;max-width:none;padding:10px;transform:none}.future-content::before{inset:-6px}.future-content .section-label span{width:28px;height:28px}.future-content .section-label small{font-size:7px}.future-kicker{margin-bottom:4px;font-size:7px}.future-content h2{margin-bottom:5px;font-size:29px;line-height:1}.future-content .lead{font-size:10px;line-height:1.35}.future-layout{display:block;margin-top:7px}.future-story{display:block;margin-top:6px}.future-story p{display:none}.future-story .definition{padding:7px 9px}.future-story .definition strong{font-size:12px}.future-story .definition span{display:none}.future-content .protocol-axis{gap:3px;margin-top:7px}.future-content .protocol-axis div{padding:6px}.future-content .protocol-axis small{font-size:7px}.future-content .protocol-axis b{margin-top:3px;font-size:9px}.future-content .protocol-axis span{display:none}.future-footer{display:flex;align-items:center;flex-direction:row;gap:7px;margin-top:7px}.future-footer .thesis{display:block;font-size:8px;line-height:1.25}.future-footer .primary-action{width:146px;min-width:146px;height:38px;flex:0 0 146px;padding:0 5px 0 11px}.future-footer .primary-action span{font-size:11px;letter-spacing:0;white-space:nowrap;word-spacing:0}.future-footer .primary-action b{width:30px;height:28px;flex-basis:30px;font-size:14px}.market-intro{top:14px;right:30px;left:20px;width:auto;padding:10px}.market-intro .section-label span{width:28px;height:28px}.market-intro h2{margin:6px 0 3px;font-size:25px}.market-intro p{max-height:28px;overflow:hidden;font-size:10px;line-height:1.35}.embedded-panel{top:132px;right:30px;bottom:auto;left:20px;width:auto;height:calc(100% - 146px)}.embedded-panel :deep(.market-section){box-sizing:border-box;width:100%;height:100%;padding:8px!important}.embedded-panel :deep(.market-header h2){font-size:20px}.embedded-panel :deep(.price-block strong){font-size:18px}.embedded-panel :deep(.price-block span){font-size:10px}.embedded-panel :deep(.market-stats){margin-top:6px}.embedded-panel :deep(.market-stats>div){gap:4px;padding:5px 4px}.embedded-panel :deep(.toolbar){min-height:34px;padding:4px 7px}.embedded-panel :deep(.intervals button){min-width:38px;padding:7px}.embedded-panel :deep(.terminal){min-height:0;margin-top:6px}.embedded-panel :deep(.chart-wrap){min-height:0;height:auto;aspect-ratio:auto;flex:1}.embedded-panel :deep(.chart){min-height:0;height:100%}.ecosystem-copy{top:14px;right:30px;left:20px;width:auto;padding:10px;transform:none}.ecosystem-copy .section-label span{width:28px;height:28px}.ecosystem-copy h2{margin:7px 0 5px;font-size:29px;line-height:1}.ecosystem-copy>p{font-size:10px;line-height:1.35}.ecosystem-copy dl{gap:5px;margin-top:7px}.ecosystem-copy dl div{padding:6px}.ecosystem-copy dt{font-size:7px}.ecosystem-copy dd{margin-top:3px;font-size:9px}.partners-panel{top:230px;right:30px;bottom:auto;left:20px;width:auto;height:250px}.partners-panel :deep(.partners-wall){box-sizing:border-box;width:100%;height:100%;min-height:0;margin:0;padding:3px 0!important}.partners-panel :deep(.wall-row){min-height:0;flex:1}.partners-panel :deep(.wall-row .wall-track){gap:8px}.partners-panel :deep(.wall-row .wall-item){min-width:64px;min-height:38px;padding:4px 6px}.partners-panel :deep(.wall-row .wall-item>img){max-width:92px;max-height:21px}.partners-panel :deep(.brand-item){min-width:102px}.partners-panel :deep(.brand-logo){width:24px;height:24px;flex-basis:24px}.partners-panel :deep(.brand-name){font-size:10px}.hero-copy{top:50%;right:30px;left:20px;width:auto;transform:translateY(-50%)}.hero-copy h2{margin:18px 0 10px;font-size:42px;line-height:.95}.hero-copy>p{font-size:13px}.hero-primary{width:166px;height:42px;margin-top:10px;padding-left:12px}.hero-primary span{font-size:13px}.hero-primary b{width:30px;height:28px;flex-basis:30px;font-size:14px}.status{gap:12px;margin-top:12px;padding-top:10px;font-size:8px}.status strong{margin-left:4px}.hero-actions{position:relative;inset:auto;width:100%;grid-template-columns:repeat(3,minmax(0,1fr));gap:5px;margin-top:10px}.hero-actions button,.hero-actions button.featured{height:52px;min-width:0;padding:8px}.hero-actions button>span{font-size:10px;line-height:1.15;white-space:normal}.hero-actions small{font-size:7px}.timeline-ui{right:5px}}
.embedded-panel :deep(.kline-chart),
.embedded-panel :deep(.kline-embed) {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  min-height: 220px;
}
@media(max-width:540px) and (max-height:720px){.future-content{top:9px}.future-content .lead{max-height:28px;overflow:hidden}.market-intro,.ecosystem-copy{top:8px}.embedded-panel{top:116px;height:calc(100% - 126px)}.partners-panel{top:205px;height:220px}.hero-copy h2{font-size:37px}.hero-actions button,.hero-actions button.featured{height:48px}}

/* Phone layout reset: natural vertical flow, no absolute-panel collisions. */
@media(max-width:540px){.timeline{touch-action:none}.space-stage{top:60px;overflow:hidden}.chapter{inset:0;display:flex;box-sizing:border-box;min-height:100%;overflow:auto;overscroll-behavior:contain;-webkit-overflow-scrolling:touch;opacity:var(--chapter-opacity,0)!important;filter:none!important;transform:translate3d(var(--chapter-x,0),var(--chapter-y,0),0) scale(var(--chapter-scale,1))!important;transition:none!important;will-change:transform,opacity}.chapter.active{opacity:var(--chapter-opacity,1)!important;transform:translate3d(var(--chapter-x,0),var(--chapter-y,0),0) scale(var(--chapter-scale,1))!important}.chapter--future,.chapter--market,.chapter--ecosystem{flex-direction:column;padding:14px 30px 14px 18px}.future-content,.market-intro,.embedded-panel,.ecosystem-copy,.partners-panel{position:relative;inset:auto;width:100%;height:auto;max-width:none;transform:none}.future-content{margin-block:auto;padding:10px}.future-content h2 em{display:block;width:max-content;max-width:100%;white-space:nowrap;word-spacing:0;letter-spacing:-.05em}.future-content .lead{max-height:none;overflow:visible}.future-footer{align-items:center;flex-direction:column}.future-footer .thesis{display:block;width:100%}.future-footer .primary-action{align-self:flex-start;width:172px!important;min-width:172px!important;height:40px!important;min-height:40px!important;flex:0 0 40px!important;margin:0!important;padding:0 5px 0 11px!important}.future-footer .primary-action b{width:30px!important;height:28px!important;flex:0 0 30px!important}.market-intro{margin-top:auto;padding:10px}.embedded-panel{height:min(480px,62vh);margin-top:10px;margin-bottom:auto}.embedded-panel :deep(.market-section){height:100%}.embedded-panel :deep(.market-shell){height:100%}.embedded-panel :deep(.chart-wrap){height:auto;min-height:280px;flex:1}.ecosystem-copy{margin-top:auto;padding:10px}.partners-panel{height:min(250px,34vh);margin-top:12px;margin-bottom:auto}.partners-panel :deep(.partners-wall){height:100%;overflow:hidden}.partners-panel :deep(.wall-row){height:33.333%;min-height:0}.partners-panel :deep(.wall-track){width:max-content;height:100%;align-items:center;justify-content:flex-start;gap:8px!important}.partners-panel :deep(.wall-item){min-width:64px!important;max-width:none;flex:0 0 auto}.chapter--hero{box-sizing:border-box;align-items:center;overflow:auto;padding:16px 30px 16px 18px}.hero-copy{position:relative;inset:auto;width:100%;margin-block:auto;transform:none}.hero-copy h2{margin:18px 0 10px;font-size:42px;line-height:.95}.hero-copy>p{font-size:13px}.hero-primary{width:166px!important;height:42px!important;margin-top:12px!important}.status{gap:12px;margin-top:14px}.hero-actions{position:relative;inset:auto;display:grid;width:100%;margin-top:18px;grid-template-columns:repeat(3,minmax(0,1fr));gap:5px}.hero-actions button,.hero-actions button.featured{box-sizing:border-box;width:100%!important;height:58px!important;min-width:0;padding:8px!important;border-color:rgba(91,181,255,.38)!important;background:linear-gradient(145deg,rgba(6,29,67,.96),rgba(2,13,34,.96))!important;color:#eef7ff!important;font-size:10px!important}.hero-actions button.featured{background:linear-gradient(135deg,#168bff,#0052ff)!important}.hero-actions button>span{color:inherit!important;font-size:10px!important;line-height:1.15}.hero-actions small{color:#eef7ff!important}.timeline-ui{position:fixed;right:5px}.control-hint{display:none}}
</style>
