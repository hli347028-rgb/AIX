<template>
  <section class="oracle-showcase" aria-labelledby="oracle-title">
    <div class="oracle-shell">
      <header class="oracle-heading">
        <div>
          <p>ORACLE NETWORK / 01</p>
          <h2 id="oracle-title">{{ $t('oracle.title') }}</h2>
        </div>
        <p class="heading-copy">{{ $t('oracle.description') }}</p>
      </header>

      <div class="oracle-grid" role="list" :aria-label="$t('oracle.levels')">
        <template v-for="(oracle, index) in oracles" :key="oracle.id">
          <button
            class="oracle-card"
            :class="{ active: expandedIndex === index }"
            type="button"
            role="listitem"
            :aria-expanded="expandedIndex === index"
            :aria-controls="`oracle-profile-${oracle.id}`"
            :aria-label="`${expandedIndex === index ? $t('oracle.collapse') : $t('oracle.expand')} ${oracle.name} ${$t('oracle.details')}`"
            @click="toggleOracle(index)"
            @keydown.left.prevent="selectOracle((index + oracles.length - 1) % oracles.length)"
            @keydown.right.prevent="selectOracle((index + 1) % oracles.length)"
          >
            <span class="card-media">
              <img :src="oracle.image" :alt="`${oracle.name} ${oracle.en}`" loading="lazy" />
              <span class="scan" aria-hidden="true"></span>
              <span class="level">{{ oracle.id }}</span>
            </span>
            <span class="card-meta">
              <span><strong>{{ oracle.name }}</strong><small>{{ oracle.en }}</small></span>
              <i>{{ oracle.confidence }}</i>
            </span>
          </button>

          <Transition name="profile-fold">
            <div
              v-if="expandedIndex === index"
              :id="`oracle-profile-${oracle.id}`"
              class="oracle-profile"
              aria-live="polite"
            >
              <div class="profile-index">
                <span>{{ oracle.id }}</span>
                <small>ACTIVE ORACLE</small>
              </div>
              <div class="profile-copy">
                <p>{{ oracle.en }}</p>
                <h3>{{ oracle.name }}</h3>
                <p class="description">{{ oracle.description }}</p>
              </div>
              <ul class="profile-signals">
                <li v-for="signal in oracle.signals" :key="signal"><span></span>{{ signal }}</li>
              </ul>
              <div class="profile-score">
                <small>{{ $t('oracle.confidence') }}</small>
                <strong>{{ oracle.confidence }}</strong>
                <span><i :style="{ width: oracle.confidence }"></i></span>
              </div>
            </div>
          </Transition>
        </template>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t: $t, tm } = useI18n()
const oracleMeta = [
  { id: 'O1', confidence: '86%', image: '/oracles/oracle-o1.png' },
  { id: 'O2', confidence: '89%', image: '/oracles/oracle-o2.png' },
  { id: 'O3', confidence: '92%', image: '/oracles/oracle-o3.png' },
  { id: 'O4', confidence: '94%', image: '/oracles/oracle-o4.png' },
  { id: 'O5', confidence: '97%', image: '/oracles/oracle-o5.png' },
]
const oracles = computed(() => {
  const content = tm('oracle.items') as unknown as Array<{ name: string; en: string; description: string; signals: string[] }>
  return oracleMeta.map((item, index) => ({ ...item, ...content[index] }))
})

const expandedIndex = ref<number | null>(null)

const selectOracle = (index: number) => {
  expandedIndex.value = index
}

const toggleOracle = (index: number) => {
  if (expandedIndex.value === index) {
    expandedIndex.value = null
    return
  }
  selectOracle(index)
}

</script>

<style scoped lang="scss">
.oracle-showcase { position: relative; padding: 94px max(34px,calc((100% - 1120px)/2)); background: var(--paper,#f7f8fa); color: var(--ink,#0a0b0d); border-top: 1px solid var(--line,#dfe3ea); }
.oracle-showcase::before { content: ''; position: absolute; top: -1px; left: max(34px,calc((100% - 1120px)/2)); width: 112px; height: 8px; background: var(--blue,#0052ff); }
.oracle-shell { max-width: 1120px; margin: 0 auto; }
.oracle-showcase:not(.is-visible) .oracle-heading,
.oracle-showcase:not(.is-visible) .oracle-card,
.oracle-showcase:not(.is-visible) .oracle-profile { opacity: 0; transform: translateY(24px); }
.oracle-showcase.is-visible .oracle-heading { animation: oracle-rise .72s cubic-bezier(.16,1,.3,1) both; }
.oracle-showcase.is-visible .oracle-card { animation: oracle-rise .68s cubic-bezier(.16,1,.3,1) both; }
.oracle-showcase.is-visible .oracle-card:nth-child(1) { animation-delay: .1s; }
.oracle-showcase.is-visible .oracle-card:nth-child(2) { animation-delay: .16s; }
.oracle-showcase.is-visible .oracle-card:nth-child(3) { animation-delay: .22s; }
.oracle-showcase.is-visible .oracle-card:nth-child(4) { animation-delay: .28s; }
.oracle-showcase.is-visible .oracle-card:nth-child(5) { animation-delay: .34s; }
.oracle-showcase.is-visible .oracle-profile { animation: oracle-rise .76s .4s cubic-bezier(.16,1,.3,1) both; }
.oracle-heading { display: flex; justify-content: space-between; align-items: flex-end; gap: 32px; margin-bottom: 34px; }
.oracle-heading p { margin: 0 0 13px; color: var(--blue,#0052ff); font: 650 9px/1.2 var(--aix-font-display); letter-spacing: .17em; }
.oracle-heading h2 { margin: 0; color: var(--ink,#0a0b0d); font-family: var(--aix-font-sans); font-size: 32px; font-weight: 650; line-height: 1.15; letter-spacing: -.035em; }
.oracle-heading .heading-copy { max-width: 370px; margin: 0; color: var(--muted,#606775); font: 400 13px/1.7 var(--aix-font-sans); letter-spacing: 0; }
.oracle-grid { display: grid; grid-template-columns: repeat(5,minmax(0,1fr)); gap: 8px; }
.oracle-card { display: flex; min-width: 0; flex-direction: column; padding: 0; border: 1px solid var(--line,#dfe3ea); background: #fff; color: var(--ink,#0a0b0d); cursor: pointer; text-align: left; transition: transform .35s cubic-bezier(.2,.8,.2,1), border-color .25s ease, box-shadow .35s ease; }
.oracle-card:hover,.oracle-card:focus-visible,.oracle-card.active { z-index: 1; border-color: var(--blue,#0052ff); outline: none; transform: translateY(-5px); box-shadow: 0 16px 34px rgba(0,82,255,.12); }
.card-media { position: relative; display: block; aspect-ratio: 1; overflow: hidden; background: #071636; }
.card-media::after { content: ''; position: absolute; inset: 0; border-bottom: 3px solid transparent; transition: border-color .25s ease; }
.oracle-card.active .card-media::after { border-color: var(--blue,#0052ff); }
.card-media img { width: 100%; height: 100%; object-fit: cover; transform: scale(1.015); filter: saturate(.82) contrast(1.04); transition: transform .65s cubic-bezier(.2,.8,.2,1), filter .35s ease; }
.oracle-card:hover img,.oracle-card.active img { transform: scale(1.06); filter: saturate(1.08) contrast(1.05); }
.scan { position: absolute; inset: -30% 0 auto; height: 35%; opacity: 0; background: linear-gradient(180deg,transparent,rgba(59,220,255,.32),transparent); pointer-events: none; }
.oracle-card.active .scan { opacity: 1; animation: scan 2.8s ease-in-out infinite; }
.level { position: absolute; top: 10px; left: 10px; padding: 7px 8px; background: var(--blue,#0052ff); color: #fff; font: 700 10px/1 var(--aix-font-display); letter-spacing: .08em; }
.card-meta { display: flex; justify-content: space-between; align-items: flex-end; gap: 8px; min-height: 72px; padding: 12px; }
.card-meta > span { display: flex; min-width: 0; flex-direction: column; gap: 5px; }
.card-meta strong { color: var(--ink,#0a0b0d); font: 750 13px/1.2 var(--aix-font-display); }
.card-meta small { overflow: hidden; color: var(--muted,#606775); font: 600 7px/1 var(--aix-font-display); letter-spacing: .09em; text-overflow: ellipsis; white-space: nowrap; }
.card-meta i { color: var(--blue,#0052ff); font: 650 10px/1 var(--aix-font-display); font-style: normal; }
.oracle-profile { grid-column: 1/-1; display: grid; grid-template-columns: 116px minmax(0,1.35fr) minmax(180px,.85fr) 120px; align-items: center; gap: 28px; margin: 6px 0 14px; padding: 28px; border: 1px solid var(--blue,#0052ff); background: #fff; transform-origin: top center; }
.profile-fold-enter-active,.profile-fold-leave-active { overflow: hidden; transition: opacity .3s ease, transform .42s cubic-bezier(.16,1,.3,1), clip-path .42s cubic-bezier(.16,1,.3,1); }
.profile-fold-enter-from,.profile-fold-leave-to { opacity: 0; transform: translateY(-8px) scaleY(.94); clip-path: inset(0 0 100% 0); }
.profile-fold-enter-to,.profile-fold-leave-from { opacity: 1; transform: translateY(0) scaleY(1); clip-path: inset(0); }
.profile-index { display: flex; flex-direction: column; gap: 8px; }
.profile-index span { color: var(--blue,#0052ff); font: 700 42px/1 var(--aix-font-display); letter-spacing: -.06em; }
.profile-index small,.profile-score small { color: var(--muted,#606775); font: 600 7px/1 var(--aix-font-display); letter-spacing: .12em; }
.profile-copy { animation: reveal .4s ease both; }
.profile-copy > p:first-child { margin: 0 0 7px; color: var(--blue,#0052ff); font: 650 8px/1 var(--aix-font-display); letter-spacing: .14em; }
.profile-copy h3 { margin: 0; font: 650 22px/1.2 var(--aix-font-sans); letter-spacing: -.025em; }
.profile-copy .description { max-width: 490px; margin: 10px 0 0; color: var(--muted,#606775); font-size: 12px; line-height: 1.7; }
.profile-signals { display: flex; flex-direction: column; gap: 9px; margin: 0; padding: 0; list-style: none; animation: reveal .4s .05s ease both; }
.profile-signals li { display: flex; align-items: center; gap: 8px; color: var(--ink,#0a0b0d); font-size: 11px; }
.profile-signals span { width: 5px; height: 5px; background: var(--blue,#0052ff); }
.profile-score { display: flex; flex-direction: column; align-items: flex-end; gap: 8px; }
.profile-score strong { color: var(--blue,#0052ff); font: 700 25px/1 var(--aix-font-display); }
.profile-score > span { width: 100%; height: 3px; background: var(--line,#dfe3ea); }
.profile-score i { display: block; height: 100%; background: var(--blue,#0052ff); transition: width .6s cubic-bezier(.2,.8,.2,1); }
@keyframes oracle-rise { from { opacity: 0; transform: translateY(24px); } }
@keyframes scan { 0% { transform: translateY(-30%); } 100% { transform: translateY(370%); } }
@keyframes reveal { from { opacity: 0; transform: translateY(5px); } }
@container(max-width:759px) {
  .oracle-showcase { padding: 78px 22px; }
  .oracle-showcase::before { left: 22px; width: 82px; }
  .oracle-heading { align-items: flex-start; flex-direction: column; gap: 14px; margin-bottom: 26px; }
  .oracle-heading h2 { font-size: 28px; }
  .oracle-heading .heading-copy { max-width: 470px; }
  .oracle-grid { grid-template-columns: repeat(2,minmax(0,1fr)); }
  .oracle-card:last-child { grid-column: 1/-1; flex-direction: row; }
  .oracle-card:last-child .card-media { width: 50%; aspect-ratio: 1.35; }
  .oracle-card:last-child .card-meta { flex: 1; }
  .card-meta { min-height: 66px; }
  .oracle-profile { grid-template-columns: 72px minmax(0,1fr); gap: 20px; padding: 22px; }
  .profile-signals { grid-column: 2; }
  .profile-score { grid-column: 1/-1; align-items: flex-start; }
  .profile-index span { font-size: 34px; }
}
@container(max-width:430px) {
  .oracle-grid { grid-template-columns: 1fr; }
  .oracle-card,.oracle-card:last-child { display: grid; grid-template-columns: 35% 1fr; }
  .oracle-card:last-child { grid-column: auto; }
  .oracle-card .card-media,.oracle-card:last-child .card-media { width: auto; aspect-ratio: 1; }
  .card-meta { min-width: 0; min-height: auto; padding: 14px 12px; }
  .card-meta strong { font-size: 12px; white-space: nowrap; }
  .oracle-profile { grid-template-columns: 1fr; }
  .profile-signals,.profile-score { grid-column: auto; }
}
@media(prefers-reduced-motion:reduce) {
  .oracle-heading,.oracle-card,.oracle-profile,.profile-copy,.profile-signals {
    opacity: 1!important;
    transform: none!important;
    animation: none!important;
  }
  .oracle-card,.card-media img,.profile-score i,.profile-fold-enter-active,.profile-fold-leave-active { transition: none; }
  .scan { animation: none!important; }
}
</style>
