<template>
  <div v-if="visible" class="notice-overlay" role="presentation" @click.self="requestClose">
    <article class="notice-modal" role="dialog" aria-modal="true" :aria-labelledby="titleId" :aria-describedby="forced ? instructionId : undefined">
      <header class="notice-head">
        <div>
          <span v-if="forced" class="notice-level notice-level--new">NEW {{ $t('announcement.latest') }}</span>
          <span v-else class="notice-level">{{ $t('announcement.notice') }}</span>
          <h2 :id="titleId">{{ current.title || $t('announcement.details') }}</h2>
          <time v-if="noticeTime">{{ noticeTime }}</time>
        </div>
        <button v-if="!forced" type="button" class="notice-close" :aria-label="$t('announcement.close')" @click="requestClose">×</button>
      </header>

      <p v-if="forced" :id="instructionId" class="notice-instruction">
        {{ $t('announcement.readBeforeConfirm') }}
      </p>

      <div class="notice-body">
        <div v-if="loadingDetail" class="notice-loading">{{ $t('announcement.loading') }}</div>
        <template v-else>
          <img
            v-if="current.image_url"
            class="notice-image"
            :src="current.image_url"
            :alt="current.title || $t('announcement.imageAlt')"
          />
          <div v-if="current.content" class="notice-content" v-html="current.content"></div>
          <p v-else-if="current.summary" class="notice-summary">{{ current.summary }}</p>
        </template>
      </div>

      <div v-if="totalPages > 1" class="notice-pager">
        <button type="button" class="pager-btn" :disabled="currentPage <= 1 || loadingDetail" @click="goPrev">
          {{ $t('announcement.prevPage') }}
        </button>
        <span class="pager-indicator">{{ $t('announcement.pageOf', { current: currentPage, total: totalPages }) }}</span>
        <button type="button" class="pager-btn" :disabled="currentPage >= totalPages || loadingDetail" @click="goNext">
          {{ $t('announcement.nextPage') }}
        </button>
      </div>
      <div v-else class="notice-pager notice-pager--single">
        <span class="pager-indicator">{{ $t('announcement.pageOf', { current: 1, total: 1 }) }}</span>
      </div>

      <footer>
        <button type="button" :disabled="!canConfirm" @click="acknowledge">
          {{ $t(canConfirm ? 'announcement.acknowledged' : 'announcement.keepReading') }}
        </button>
      </footer>
    </article>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getAnnouncementDetail, type AnnouncementItem } from '@/api/aix'

const { t: $t } = useI18n()
const props = withDefaults(
  defineProps<{
    items?: AnnouncementItem[]
    forced?: boolean
  }>(),
  { items: () => [], forced: false },
)
const emit = defineEmits<{ (event: 'close'): void; (event: 'acknowledge'): void }>()

const titleId = 'announcement-detail-title'
const instructionId = 'announcement-read-instruction'
const currentPage = ref(1)
const reachedLastPage = ref(false)
const loadingDetail = ref(false)
const detailCache = ref<Record<number, AnnouncementItem>>({})

const visible = computed(() => (props.items?.length || 0) > 0)
const totalPages = computed(() => Math.max(1, props.items?.length || 0))
const current = computed<AnnouncementItem>(() => {
  const list = props.items || []
  const base = list[currentPage.value - 1] || {}
  const id = Number(base.id || 0)
  if (id && detailCache.value[id]) {
    return { ...base, ...detailCache.value[id] }
  }
  return base
})
const noticeTime = computed(() => current.value.published_at || current.value.created_at || '')
const canConfirm = computed(() => {
  if (!visible.value || loadingDetail.value) return false
  if (totalPages.value <= 1) return true
  return reachedLastPage.value && currentPage.value >= totalPages.value
})

const requestClose = () => {
  if (!props.forced) emit('close')
}
const acknowledge = () => {
  if (!canConfirm.value) return
  emit('acknowledge')
}

const loadCurrentDetail = async () => {
  const list = props.items || []
  const item = list[currentPage.value - 1]
  const id = Number(item?.id || 0)
  if (!id) return
  if (detailCache.value[id]?.content) return
  loadingDetail.value = true
  try {
    const detail = await getAnnouncementDetail(id)
    detailCache.value = { ...detailCache.value, [id]: detail }
  } catch {
    // 列表内容可作降级
  } finally {
    loadingDetail.value = false
  }
}

const goPrev = async () => {
  if (currentPage.value <= 1) return
  currentPage.value -= 1
  await loadCurrentDetail()
}
const goNext = async () => {
  if (currentPage.value >= totalPages.value) return
  currentPage.value += 1
  if (currentPage.value >= totalPages.value) reachedLastPage.value = true
  await loadCurrentDetail()
}

const resetPager = async () => {
  currentPage.value = 1
  reachedLastPage.value = (props.items?.length || 0) <= 1
  detailCache.value = {}
  await loadCurrentDetail()
}

watch(
  () => props.items,
  (items) => {
    document.body.style.overflow = items && items.length ? 'hidden' : ''
    if (items && items.length) resetPager()
  },
  { immediate: true, deep: true },
)

watch(
  () => currentPage.value,
  (page) => {
    if (page >= totalPages.value) reachedLastPage.value = true
  },
)

onBeforeUnmount(() => {
  document.body.style.overflow = ''
})
</script>

<style scoped lang="scss">
.notice-overlay {
  position: fixed;
  z-index: 13000;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(5, 12, 28, 0.58);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
}

.notice-modal {
  display: flex;
  flex-direction: column;
  width: 92vw;
  max-width: 520px;
  max-height: 82vh;
  overflow: hidden;
  border: 1px solid rgba(0, 82, 255, 0.18);
  border-radius: 24px;
  background: #fff;
  color: #101827;
  box-shadow: 0 28px 80px rgba(3, 17, 48, 0.24);
  animation: notice-in 0.28s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.notice-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 24px 24px 18px;
  border-bottom: 1px solid rgba(17, 36, 76, 0.1);
}

.notice-instruction {
  margin: 0;
  padding: 10px 24px;
  background: #f2f5ff;
  color: #0052ff;
  font-size: 12px;
  font-weight: 650;
  letter-spacing: 0.02em;
}

.notice-level {
  display: inline-flex;
  margin-bottom: 10px;
  color: #627089;
  font-size: 11px;
  font-weight: 750;
  letter-spacing: 0.08em;
}

.notice-level--new {
  color: #0052ff;
}

h2 {
  margin: 0;
  color: #101827;
  font-size: 21px;
  line-height: 1.35;
  text-wrap: balance;
}

time {
  display: block;
  margin-top: 8px;
  color: #7e899c;
  font-size: 12px;
}

.notice-close {
  flex: 0 0 auto;
  width: 34px;
  height: 34px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: #f1f4f9;
  color: #566174;
  font-size: 24px;
  line-height: 30px;
  cursor: pointer;
}

.notice-body {
  flex: 1;
  min-height: 160px;
  max-height: 46vh;
  overflow-y: auto;
  padding: 18px 24px;
}

.notice-loading {
  color: #7e899c;
  font-size: 14px;
  text-align: center;
  padding: 24px 0;
}

.notice-image {
  display: block;
  width: 100%;
  height: auto;
  border-radius: 14px;
  background: #f3f6fa;
}

.notice-content,
.notice-summary {
  margin: 0;
  color: #334057;
  font-size: 15px;
  line-height: 1.75;
  word-break: break-word;
}

.notice-image + .notice-content,
.notice-image + .notice-summary {
  margin-top: 18px;
}

.notice-content :deep(img) {
  display: block;
  width: 100%;
  height: auto;
  border-radius: 12px;
}

.notice-content :deep(a) {
  color: #0052ff;
}

.notice-pager {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 22px 0;
}

.notice-pager--single {
  justify-content: center;
}

.pager-btn {
  min-width: 72px;
  height: 34px;
  padding: 0 12px;
  border: 1px solid rgba(0, 82, 255, 0.22);
  border-radius: 17px;
  background: #f2f5ff;
  color: #0052ff;
  font-size: 13px;
  font-weight: 650;
  cursor: pointer;
}

.pager-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.pager-indicator {
  color: #627089;
  font-size: 12px;
  font-weight: 600;
}

footer {
  padding: 18px 22px 24px;
}

footer button {
  width: 100%;
  min-height: 46px;
  border: 0;
  border-radius: 23px;
  background: #0052ff;
  color: #fff;
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
  transition: opacity 0.2s ease, transform 0.2s ease;
}

footer button:disabled {
  background: #d8deea;
  color: #7d8798;
  cursor: not-allowed;
}

footer button:not(:disabled):active {
  transform: scale(0.985);
}

@keyframes notice-in {
  from {
    opacity: 0;
    transform: translateY(14px) scale(0.97);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

@media (max-width: 540px) {
  .notice-overlay {
    align-items: flex-end;
    padding: 0;
  }

  .notice-modal {
    width: 100%;
    max-height: 88vh;
    max-height: 88dvh;
    border-radius: 24px 24px 0 0;
  }

  .notice-head {
    padding: 21px 20px 16px;
  }

  .notice-body {
    padding: 16px 20px;
    max-height: 48vh;
  }

  .notice-pager {
    padding: 12px 20px 0;
  }

  footer {
    padding: 16px 20px calc(18px + env(safe-area-inset-bottom));
  }
}

@media (prefers-reduced-motion: reduce) {
  .notice-modal {
    animation: none;
  }
}
</style>
