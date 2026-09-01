<template>
  <Teleport to="body">
    <div v-if="visible" class="sidebar-overlay" @click="close">
      <aside class="sidebar" :aria-label="$t('futurefi.navigation')" @click.stop>
        <div class="ambient-logo" aria-hidden="true">
          <img src="/assets/aix-orbit-logo.jpeg" alt="" />
        </div>

        <header class="sidebar-head">
          <div class="brand">
            <span class="brand-mark-wrap">
              <img src="/assets/aix-logo-sm.png" alt="AIX" class="brand-mark" />
            </span>
          <div class="brand-copy">
              <span class="brand-name">AI PREDICTION EXCHANGE</span>
              <span class="brand-meta">FUTUREFI PROTOCOL</span>
            </div>
          </div>

          <button class="close-btn" type="button" :aria-label="$t('common.close')" @click="close">
            <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M7 7l10 10M17 7 7 17" stroke="currentColor" stroke-width="1.5" />
            </svg>
          </button>
        </header>

        <div class="nav-heading">
          <span>{{ $t('common.explore') }}</span>
          <span>{{ String(liveItems.length).padStart(2, '0') }}</span>
        </div>

        <nav class="sidebar-nav">
          <a
            v-for="(item, index) in liveItems"
            :key="item.key"
            class="nav-row"
            :class="{ active: item.path && isActive(item.path) }"
            :href="item.href || item.path"
            :target="item.href ? '_blank' : undefined"
            :rel="item.href ? 'noopener noreferrer' : undefined"
            @click="handleNav(item, $event)"
          >
            <span class="nav-index">{{ String(index + 1).padStart(2, '0') }}</span>
            <span class="nav-label">{{ item.label }}</span>
            <span class="nav-direction" :class="{ external: item.href }" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path v-if="item.href" d="M8 16 16 8M16 8H9M16 8v7" stroke="currentColor" stroke-width="1.5" />
                <path v-else d="M5 12h13m-5-5 5 5-5 5" stroke="currentColor" stroke-width="1.5" />
              </svg>
            </span>
          </a>
        </nav>

        <section v-if="upcomingItems.length" class="sidebar-parked">
          <div class="parked-head">
            <span>{{ $t('common.comingSoon') }}</span>
            <span class="parked-line"></span>
          </div>
          <div class="parked-grid">
            <button
              v-for="item in upcomingItems"
              :key="item.key"
              class="parked-row"
              type="button"
              @click="handleNav(item, $event)"
            >
              <span>{{ item.label }}</span>
              <span class="parked-dot" aria-hidden="true"></span>
            </button>
          </div>
        </section>
      </aside>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { showToast } from 'vant'

const router = useRouter()
const route = useRoute()
const { t: $t } = useI18n()

const props = defineProps({ visible: Boolean })
const emit = defineEmits(['close'])

const unlockPage = () => {
  document.body.style.removeProperty('overflow')
  document.body.style.removeProperty('overscroll-behavior')
  document.documentElement.style.removeProperty('overflow')
}

watch(() => props.visible, (open) => {
  if (!open) {
    unlockPage()
    return
  }
  document.body.style.overflow = 'hidden'
  document.body.style.overscrollBehavior = 'none'
  document.documentElement.style.overflow = 'hidden'
}, { immediate: true })

onUnmounted(unlockPage)

const BRIDGE_URL = 'https://bridge.winbit.win/'
const BITWINEX_URL = 'https://bitwinex.cloud'

const navItems = computed(() => [
  { key: 'home', label: $t('tab.home'), path: '/' },
  { key: 'project-intro', label: $t('futurefi.projectIntro'), path: '/futurefi' },
  { key: 'team', label: $t('tab.myTeam'), path: '/community' },
  { key: 'assets', label: $t('tab.myAssets'), path: '/wallet' },
  { key: 'recharge', label: $t('tab.rechargeZone'), path: '/recharge' },
  { key: 'order', label: $t('tab.orderZone'), path: '/node' },
  { key: 'rules', label: $t('index.rulesSummaryTitle'), path: '/rules' },
  { key: 'bridge', label: $t('tab.crossChain'), href: BRIDGE_URL },
  { key: 'bitwinex', label: $t('tab.bitwinex'), href: BITWINEX_URL },
  { key: 'announcements', label: $t('announcement.title'), path: '/announcements' },
  { key: 'support', label: $t('tab.customerService'), path: '/support' },
  { key: 'launch', label: $t('tab.globalLaunch'), upcoming: true },
  { key: 'bank', label: $t('tab.digitalBank'), upcoming: true },
  { key: 'games', label: $t('tab.chainGameZone'), upcoming: true },
  { key: 'predict', label: $t('tab.predictionZone'), upcoming: true },
  { key: 'chat', label: $t('tab.cloudChat'), upcoming: true },
  { key: 'faq', label: $t('tab.faq'), upcoming: true },
])

const liveItems = computed(() => navItems.value.filter((item) => !item.upcoming))
const upcomingItems = computed(() => navItems.value.filter((item) => item.upcoming))
const isActive = (path) => route.path === path
const close = () => emit('close')

const go = (path) => {
  if (route.path !== path) router.push(path)
  close()
}

const handleNav = (item, event) => {
  if (item.path) {
    event?.preventDefault()
    go(item.path)
    return
  }
  if (item.href) {
  close()
    return
  }
  showToast($t('common.comingSoon'))
}

</script>

<style lang="scss" scoped>
.sidebar-overlay {
  position: fixed;
  inset: 0;
  z-index: 2000;
  overflow: hidden;
  overscroll-behavior: none;
  touch-action: none;
  background: rgba(10, 11, 13, 0.28);
  backdrop-filter: blur(12px) saturate(0.9);
  animation: overlay-in 0.35s ease both;
}

.sidebar {
  --panel-text: #0a0b0d;
  --panel-muted: #606775;
  --panel-line: #dfe3ea;
  --panel-blue: #0052ff;
  position: absolute;
  inset: 0 auto 0 max(0px, calc(50% - 207px));
  width: min(100%, 414px);
  max-width: 100%;
  box-sizing: border-box;
  padding: 22px 24px 32px;
  display: flex;
  flex-direction: column;
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  overscroll-behavior-x: none;
  touch-action: pan-y;
  scrollbar-width: none;
  color: var(--panel-text);
  background: #f7f8fa;
  border-right: 1px solid var(--panel-line);
  box-shadow: 36px 0 90px rgba(10, 11, 13, 0.16);
  animation: panel-in 0.55s cubic-bezier(0.2, 0.75, 0.2, 1) both;

  &::after {
    content: '';
    position: fixed;
    inset: 0 auto 0 max(0px, calc(50% - 207px));
    width: min(100%, 414px);
    pointer-events: none;
    opacity: 1;
    background: linear-gradient(to bottom, var(--panel-blue) 0 4px, transparent 4px);
  }

  &::-webkit-scrollbar { display: none; }
}

.ambient-logo {
  position: absolute;
  top: 86px;
  right: 0;
  width: 220px;
  pointer-events: none;
  opacity: 0.08;
  mix-blend-mode: multiply;
  mask-image: radial-gradient(circle, black 30%, transparent 72%);

  img { display: block; width: 100%; }
}

.sidebar-head,
.account-row,
.nav-heading,
.sidebar-nav,
.sidebar-parked { position: relative; z-index: 1; }

.sidebar-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--panel-line);
}

.brand { display: flex; align-items: center; gap: 12px; min-width: 0; }
.brand-mark-wrap {
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  display: grid;
  place-items: center;
  overflow: hidden;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(26, 75, 124, 0.18), transparent 72%);
}
.brand-mark {
  width: 42px;
  height: 42px;
  display: block;
  object-fit: contain;
  mix-blend-mode: screen;
  filter: contrast(1.22) saturate(0.82) brightness(0.92);
  -webkit-mask-image: radial-gradient(circle, #000 48%, rgba(0, 0, 0, 0.94) 60%, transparent 78%);
  mask-image: radial-gradient(circle, #000 48%, rgba(0, 0, 0, 0.94) 60%, transparent 78%);
}
.brand-copy { display: flex; min-width: 0; flex-direction: column; gap: 4px; }
.brand-name { overflow: hidden; color: var(--panel-text); font-size: 10px; font-weight: 750; letter-spacing: 0.17em; text-overflow: ellipsis; white-space: nowrap; }
.brand-meta { color: var(--panel-muted); font-size: 8px; font-weight: 600; letter-spacing: 0.24em; }

.close-btn {
  width: 38px;
  height: 38px;
  padding: 0;
  flex: 0 0 auto;
  display: grid;
  place-items: center;
  border: 1px solid var(--panel-line);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.025);
  color: var(--panel-muted);
  cursor: pointer;
  transition: color 0.25s ease, border-color 0.25s ease, background 0.25s ease, transform 0.35s ease;

  svg { width: 18px; height: 18px; }
  &:hover { color: var(--panel-text); border-color: rgba(101, 184, 243, 0.48); background: rgba(101, 184, 243, 0.08); transform: rotate(4deg); }
  &:focus-visible { outline: 2px solid var(--panel-blue); outline-offset: 3px; }
}

.account-row {
  width: 100%;
  margin-top: 18px;
  padding: 14px;
  display: flex;
  align-items: center;
  gap: 12px;
  border: 1px solid var(--panel-line);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.035);
  color: inherit;
  cursor: pointer;
  text-align: left;
  transition: border-color 0.25s ease, background 0.25s ease, transform 0.25s ease;

  &:hover { border-color: rgba(101, 184, 243, 0.42); background: rgba(101, 184, 243, 0.07); transform: translateY(-1px); }
  &:focus-visible { outline: 2px solid var(--panel-blue); outline-offset: 2px; }
}

.account-status { width: 7px; height: 7px; flex: 0 0 auto; border-radius: 50%; background: var(--panel-blue); box-shadow: 0 0 14px rgba(101, 184, 243, 0.7); }
.account-copy { min-width: 0; flex: 1; display: flex; flex-direction: column; gap: 3px; }
.account-label { color: var(--panel-muted); font-size: 8px; font-weight: 650; letter-spacing: 0.18em; }
.address-value { color: #dce7f3; font-family: var(--aix-font-display); font-size: 13px; font-weight: 650; font-variant-numeric: tabular-nums; letter-spacing: 0.05em; }
.copy-action { width: 30px; height: 30px; display: grid; place-items: center; border-radius: 8px; color: var(--panel-blue); background: rgba(101, 184, 243, 0.08); }
.copy-action svg { width: 15px; height: 15px; }

.nav-heading {
  margin-top: 26px;
  padding-bottom: 9px;
  display: flex;
  justify-content: space-between;
  color: var(--panel-muted);
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.2em;
  border-bottom: 1px solid var(--panel-line);
}

.sidebar-nav { display: flex; flex-direction: column; }
.nav-row {
  min-height: 57px;
  padding: 0 4px;
  display: flex;
  align-items: center;
  gap: 14px;
  border-bottom: 1px solid var(--panel-line);
  color: #aebaca;
  text-decoration: none;
  transition: color 0.25s ease, padding 0.35s cubic-bezier(0.2, 0.75, 0.2, 1), background 0.25s ease;
  animation: item-in 0.5s both;
}

.nav-row:nth-child(1) { animation-delay: 0.08s; }
.nav-row:nth-child(2) { animation-delay: 0.12s; }
.nav-row:nth-child(3) { animation-delay: 0.16s; }
.nav-row:nth-child(4) { animation-delay: 0.2s; }
.nav-row:nth-child(5) { animation-delay: 0.24s; }
.nav-row:nth-child(6) { animation-delay: 0.28s; }
.nav-row:nth-child(7) { animation-delay: 0.32s; }
.nav-row:nth-child(8) { animation-delay: 0.36s; }
.nav-row:nth-child(9) { animation-delay: 0.4s; }
.nav-row:nth-child(10) { animation-delay: 0.44s; }
.nav-row:nth-child(11) { animation-delay: 0.48s; }
.nav-row:nth-child(12) { animation-delay: 0.52s; }
.nav-row:hover { padding-left: 10px; color: var(--panel-text); background: linear-gradient(90deg, rgba(101, 184, 243, 0.08), transparent 78%); }
.nav-row.active { color: var(--panel-text); background: linear-gradient(90deg, rgba(101, 184, 243, 0.12), transparent 80%); }
.nav-row.active .nav-index { color: var(--panel-blue); }
.nav-row.active .nav-label { font-weight: 750; }
.nav-index { width: 20px; flex: 0 0 auto; color: #536274; font-family: var(--aix-font-display); font-size: 9px; font-weight: 600; letter-spacing: 0.08em; }
.nav-label { flex: 1; font-size: 15px; font-weight: 550; letter-spacing: 0.02em; }
.nav-direction { width: 28px; height: 28px; display: grid; place-items: center; color: #607186; transition: color 0.25s ease, transform 0.3s ease; }
.nav-direction svg { width: 16px; height: 16px; }
.nav-row:hover .nav-direction { color: var(--panel-blue); transform: translateX(3px); }
.nav-row:hover .nav-direction.external { transform: translate(2px, -2px); }

.sidebar-parked { margin-top: 25px; }
.parked-head { display: flex; align-items: center; gap: 12px; color: #59697c; font-size: 9px; font-weight: 700; letter-spacing: 0.12em; }
.parked-line { height: 1px; flex: 1; background: var(--panel-line); }
.parked-grid { margin-top: 14px; display: grid; grid-template-columns: 1fr 1fr; gap: 5px 16px; }
.parked-row { min-height: 31px; padding: 0; display: flex; align-items: center; justify-content: space-between; gap: 8px; border: 0; background: transparent; color: #59697c; font-size: 12px; font-weight: 550; text-align: left; cursor: pointer; transition: color 0.2s ease; }
.parked-row:hover { color: #8999ab; }
.parked-dot { width: 3px; height: 3px; flex: 0 0 auto; border-radius: 50%; background: currentColor; }

.brand-mark-wrap { background: #f2f5ff; }
.brand-mark { mix-blend-mode: multiply; filter: saturate(1.05) brightness(1.08); }
.close-btn { background: #fff; }
.close-btn:hover { color: #fff; border-color: var(--panel-blue); background: var(--panel-blue); }
.account-row { background: #fff; }
.account-row:hover { border-color: rgba(0, 82, 255, 0.45); background: #f2f5ff; }
.account-status { box-shadow: none; }
.address-value { color: var(--panel-text); }
.copy-action { color: var(--panel-blue); background: #f2f5ff; }
.nav-row { color: #303744; }
.nav-row:hover { color: var(--panel-text); background: linear-gradient(90deg, #f2f5ff, transparent 82%); }
.nav-row.active { color: #fff; background: var(--panel-blue); }
.nav-row.active .nav-index,
.nav-row.active .nav-direction { color: #fff; }
.nav-index,
.nav-direction { color: #737b89; }
.parked-head,
.parked-row { color: #737b89; }
.parked-row:hover { color: var(--panel-blue); }

@keyframes overlay-in { from { opacity: 0; } }
@keyframes panel-in { from { opacity: 0; transform: translateX(-42px); } }
@keyframes item-in { from { opacity: 0; transform: translateX(-10px); } }

@media (max-width: 374px) {
  .sidebar { width: 100%; padding-inline: 18px; }
  .brand-name { max-width: 190px; font-size: 9px; }
}

@media (prefers-reduced-motion: reduce) {
  .sidebar-overlay,
  .sidebar,
  .nav-row { animation: none; }
  .nav-row,
  .close-btn,
  .account-row,
  .nav-direction { transition: none; }
}
</style>
