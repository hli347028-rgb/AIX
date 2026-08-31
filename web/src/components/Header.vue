<template>
  <header class="app-header">
    <div class="header-container">
      <button class="brand" :aria-label="$t('common.openNavigation')" @click="handleSidebarClick">
        <img src="/assets/aix-orbit-logo.jpeg" alt="AIX" />
        <span class="brand-menu" aria-hidden="true">
          <svg viewBox="0 0 18 18" fill="none">
            <rect x="3" y="3" width="4" height="4" rx="1" />
            <rect x="11" y="3" width="4" height="4" rx="1" />
            <rect x="3" y="11" width="4" height="4" rx="1" />
            <path d="M11 13h4m-2-2 2 2-2 2" />
          </svg>
          <i></i>
        </span>
      </button>
      <div class="header-actions">
        <button
          type="button"
          class="announcement-trigger"
          :class="`announcement-trigger--${noticePriority}`"
          :aria-label="noticeAriaLabel"
          @click="openAnnouncement"
        >
          <svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M5 10.5v3l2.2.8L9 19h2l-1.3-4H13l5 3V6l-5 3H7.2L5 10.5Z"/><path d="M20 9v6"/></svg>
          <span v-if="noticePriority === 'important'" class="announcement-alert">!!!</span>
          <span v-else-if="noticePriority === 'new'" class="announcement-dot">NEW</span>
        </button>
        <button
          class="wallet"
          :disabled="Boolean(address) || isWalletSwitching"
          :aria-busy="isWalletSwitching"
          :aria-label="$t('common.switchWallet')"
          :title="$t('common.switchWallet')"
          @click="handleWalletClick"
        >
          {{ address ? formatAddress(address) : $t('common.connectWallet') }}
        </button>
        <button class="language" :aria-label="$t('common.selectLanguage')" @click="handleLanguageClick">
          <svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><circle cx="12" cy="12" r="8.5"/><path d="M3.5 12h17M12 3.5c2.2 2.4 3.4 5.3 3.4 8.5S14.2 18.1 12 20.5C9.8 18.1 8.6 15.2 8.6 12S9.8 5.9 12 3.5Z"/></svg>
        </button>
      </div>
    </div>
  </header>

  <Teleport to="body">
    <div v-if="showLangDrawer" class="lang-overlay" @click="showLangDrawer = false">
      <div class="lang-drawer" @click.stop>
        <p>{{ $t('common.selectLanguage') }}</p>
        <button v-for="lang in languages" :key="lang.code" :class="{ active: currentLanguage === lang.code }" @click="selectLanguage(lang.code)">{{ lang.name }}</button>
      </div>
    </div>
    <Sidebar :visible="showSidebar" @close="showSidebar = false" />
    <AnnouncementDetailModal
      :announcement="selectedAnnouncement"
      :forced="isForcedAnnouncement"
      @close="closeAnnouncement"
      @acknowledge="acknowledgeAnnouncement"
    />
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import userPerson from '@/pinia/person'
import Sidebar from '@/components/Sidebar.vue'
import AnnouncementDetailModal from '@/components/AnnouncementDetailModal.vue'
import { getAnnouncementDetail, listAnnouncements, type AnnouncementItem, type AnnouncementPriority } from '@/api/aix'
import { userLanguageOptions } from '@/i18n/languages'
import { restartCurrentApp } from '@/tools/plaocRuntime'

const person = userPerson()
const acknowledgedKeys = ref<Set<string>>(new Set())
const noticeStorageKey = (notice: AnnouncementItem) => {
  const baseKey = `aix-announcement-read:${notice.id || notice.title || 'latest'}`
  const accountKey = String(person.address || localStorage.getItem('account') || 'anonymous').toLowerCase()
  return `${baseKey}:${accountKey}`
}
const hasAcknowledged = (key: string) =>
  localStorage.getItem(key) === '1' || sessionStorage.getItem(key) === '1'
const isAcknowledged = (notice: AnnouncementItem) =>
  acknowledgedKeys.value.has(noticeStorageKey(notice)) || hasAcknowledged(noticeStorageKey(notice))
const markAcknowledged = (key: string) => {
  localStorage.setItem(key, '1')
  sessionStorage.setItem(key, '1')
  acknowledgedKeys.value = new Set(acknowledgedKeys.value).add(key)
}

const { locale, t: $t } = useI18n()
const currentLanguage = ref(locale.value)
const showLangDrawer = ref(false)
const showSidebar = ref(false)
const isWalletSwitching = ref(false)
const announcements = ref<AnnouncementItem[]>([])
const selectedAnnouncement = ref<AnnouncementItem | null>(null)
const isForcedAnnouncement = ref(false)
const address = computed(() => person.address)
const noticePriority = computed<AnnouncementPriority>(() => {
  const unreadAnnouncements = announcements.value.filter(item => !isAcknowledged(item))
  if (unreadAnnouncements.some(item => item.priority === 'important')) return 'important'
  if (unreadAnnouncements.some(item => item.priority === 'new')) return 'new'
  return 'normal'
})
const noticeAriaLabel = computed(() => $t(`announcement.${noticePriority.value === 'important' ? 'viewImportant' : noticePriority.value === 'new' ? 'viewLatest' : 'view'}`))
const languages = userLanguageOptions
watch(locale, value => {
  currentLanguage.value = value
  document.documentElement.lang = value
}, { immediate: true })
const formatAddress = (value: string) => `${value.slice(0, 6)}...${value.slice(-6)}`
const normalizeAnnouncements = (items: AnnouncementItem[] = []) =>
  items
    .filter((item) => item.status !== 'draft' && item.status !== 'archived')
    .map((item, index) => ({
      ...item,
      // 后台未设优先级时：最新一条标 new，便于铃铛提示
      priority: (item.priority || (index === 0 ? 'new' : 'normal')) as AnnouncementPriority,
      status: item.status || 'published',
    }))
const handleSidebarClick = () => { showSidebar.value = true }
const handleLanguageClick = () => { showLangDrawer.value = true }
const selectLanguage = (code: string) => { locale.value = code; localStorage.setItem('lan', code); document.documentElement.lang = code; showLangDrawer.value = false }
const handleWalletClick = async () => {
  if (address.value || isWalletSwitching.value) return

  isWalletSwitching.value = true
  const token = localStorage.getItem('token')
  const account = localStorage.getItem('account')
  try {
    // 复用项目既有的钱包登录入口：清除当前授权后，由应用初始化流程
    // 重新获取钱包账户、签名挑战并换取登录 token。
    localStorage.removeItem('token')
    localStorage.removeItem('account')
    await restartCurrentApp()
  } catch (error) {
    if (token) localStorage.setItem('token', token)
    if (account) localStorage.setItem('account', account)
    console.error('[handleWalletClick] Failed to restart wallet login', error)
    isWalletSwitching.value = false
  }
}
let announcementRequest = 0
const showAnnouncement = async (notice: AnnouncementItem, forced: boolean) => {
  const requestId = ++announcementRequest
  isForcedAnnouncement.value = forced
  selectedAnnouncement.value = notice
  if (!notice.id) return
  try {
    const detail = await getAnnouncementDetail(notice.id)
    if (requestId !== announcementRequest) return
    selectedAnnouncement.value = detail
  } catch {
    // 列表数据可作为详情接口临时不可用时的降级内容。
  }
}
const openAnnouncement = async () => {
  const topNotice = announcements.value[0]
  if (topNotice) await showAnnouncement(topNotice, false)
}
const dismissAnnouncement = () => {
  announcementRequest += 1
  isForcedAnnouncement.value = false
  selectedAnnouncement.value = null
}
const closeAnnouncement = () => {
  if (isForcedAnnouncement.value) return
  dismissAnnouncement()
}
const acknowledgeAnnouncement = () => {
  const notice = selectedAnnouncement.value
  if (notice) markAcknowledged(noticeStorageKey(notice))
  dismissAnnouncement()
}

onMounted(async () => {
  try {
    const result = await listAnnouncements({ page: 1, page_size: 20 })
    const rank: Record<AnnouncementPriority, number> = { important: 3, new: 2, normal: 1 }
    announcements.value = normalizeAnnouncements(result.list || []).sort(
      (left, right) => rank[right.priority || 'normal'] - rank[left.priority || 'normal'],
    )

    const unreadAnnouncement = announcements.value.find((item) => !isAcknowledged(item))
    if (unreadAnnouncement) {
      // 未读公告强制确认；内容来自管理后台公告列表
      await showAnnouncement(unreadAnnouncement, true)
    }
  } catch {
    announcements.value = []
  }
})
</script>

<style lang="scss" scoped>
.app-header{position:fixed;z-index:12000;top:0;left:50%;width:100%;max-width:414px;transform:translateX(-50%);background:rgba(255,255,255,.88);border-bottom:1px solid rgba(0,82,255,.12);backdrop-filter:blur(18px);-webkit-backdrop-filter:blur(18px)}
.header-container{display:flex;align-items:center;justify-content:space-between;min-height:60px;padding:0 20px}.brand,.header-actions{display:flex;align-items:center}.announcement-trigger{position:relative;display:grid;place-items:center;width:34px;height:34px;padding:0;border:1px solid rgba(0,82,255,.2);border-radius:50%;background:#f2f5ff;color:#0052ff;cursor:pointer;transition:transform .2s ease,background-color .2s ease,color .2s ease}.announcement-trigger svg{width:18px;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round}.announcement-trigger:hover,.announcement-trigger:focus-visible{outline:none;transform:translateY(-1px);background:#0052ff;color:#fff}.announcement-dot,.announcement-alert{position:absolute;top:-8px;right:-9px;display:grid;place-items:center;min-width:25px;height:16px;padding:0 4px;border:2px solid #fff;border-radius:9px;background:#0052ff;color:#fff;font-size:8px;font-weight:850;line-height:1;letter-spacing:-.02em;box-shadow:0 3px 10px rgba(0,82,255,.3)}.announcement-alert{min-width:27px;background:#df1f32;font-size:11px;letter-spacing:-.08em;box-shadow:0 3px 12px rgba(223,31,50,.42);animation:announcement-urgent 1.2s ease-in-out infinite}.announcement-trigger--important{border-color:rgba(223,31,50,.32);background:#fff4f5;color:#df1f32}@keyframes announcement-urgent{50%{transform:scale(1.12);box-shadow:0 3px 18px rgba(223,31,50,.68)}}.brand{gap:7px;padding:0;border:0;background:transparent;color:#0052ff;cursor:pointer}.brand img{width:29px;height:29px;object-fit:cover;border-radius:50%;opacity:1;mix-blend-mode:multiply;filter:saturate(1.1) brightness(1.12);transition:opacity .25s ease,transform .35s cubic-bezier(.2,.8,.2,1)}.brand-menu{position:relative;display:grid;place-items:center;width:30px;height:30px;border:1px solid rgba(0,82,255,.28);border-radius:8px;background:#f2f5ff;color:#0052ff;transition:color .25s ease,border-color .25s ease,background .25s ease,transform .35s cubic-bezier(.2,.8,.2,1)}.brand-menu svg{width:17px;stroke:currentColor;stroke-width:1.3;stroke-linecap:round;stroke-linejoin:round}.brand-menu svg rect{fill:currentColor;stroke:none}.brand-menu i{position:absolute;top:-3px;right:-3px;width:7px;height:7px;border:2px solid #070c16;border-radius:50%;background:#72a9df;box-shadow:0 0 8px rgba(114,169,223,.65)}.brand:hover img,.brand:focus-visible img{opacity:1;transform:rotate(-8deg) scale(1.06)}.brand:hover .brand-menu,.brand:focus-visible .brand-menu{color:#fff;border-color:rgba(180,213,246,.82);background:rgba(98,151,207,.28);transform:translateX(3px)}.brand:focus-visible{outline:1px solid rgba(221,234,249,.5);outline-offset:5px;border-radius:16px}.header-actions{gap:13px}.wallet{padding:7px 12px;border:1px solid rgba(0,82,255,.24);border-radius:18px;background:#f2f5ff;color:#0052ff;font-size:11px;font-weight:600;cursor:pointer;transition:border-color .25s ease,color .25s ease,background .25s ease}.wallet:hover:not(:disabled),.wallet:focus-visible:not(:disabled){outline:none;border-color:#0052ff;background:#0052ff;color:#fff}.wallet:disabled{cursor:default}.wallet[aria-busy="true"]{cursor:wait;opacity:.65}.language{display:grid;place-items:center;padding:0;border:0;background:transparent;color:#0052ff;cursor:pointer;transition:color .25s ease,transform .25s ease}.language:hover,.language:focus-visible{outline:none;color:#003ec4;transform:translateY(-1px)}.language svg{width:18px;stroke:currentColor;stroke-width:1.25}
.lang-overlay{position:fixed;z-index:1000;inset:0;background:rgba(11,18,32,.22);backdrop-filter:blur(7px);-webkit-backdrop-filter:blur(7px)}.lang-drawer{position:absolute;top:0;right:0;width:min(76%,290px);height:100%;padding:32px 24px;background:#fbfaf7;border-left:1px solid rgba(12,23,48,.12);box-shadow:-18px 0 48px rgba(12,23,48,.1)}.lang-drawer::before{content:"";position:absolute;top:0;left:0;width:72px;height:4px;background:#0052ff}.lang-drawer p{margin:0 0 24px;font:650 10px/1 var(--aix-font-display);letter-spacing:.14em;color:#0052ff}.lang-drawer button{position:relative;display:block;width:100%;padding:17px 12px;border:0;border-bottom:1px solid rgba(12,23,48,.11);background:transparent;color:#263247;font-size:15px;font-weight:520;text-align:left;cursor:pointer;transition:color .2s ease,background .2s ease,padding-left .2s ease}.lang-drawer button:hover,.lang-drawer button:focus-visible{outline:none;background:#f1f5ff;color:#0052ff;padding-left:16px}.lang-drawer button.active{background:#0052ff;color:#fff;font-weight:680;padding-left:16px}.lang-drawer button.active::after{content:"✓";position:absolute;right:12px;color:#fff;font-size:12px}
@media(max-width:420px){.header-container{padding:0 14px}.brand{gap:5px}.brand img{width:25px;height:25px}.header-actions{gap:9px}.announcement-trigger{width:31px;height:31px}.wallet{max-width:128px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}}
@media(prefers-reduced-motion:reduce){.announcement-alert{animation:none}}
</style>
