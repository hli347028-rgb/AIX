<template>
  <!-- Header 已经把本组件传送到 body。这里不再嵌套 Teleport：部分钱包的旧
       WebView 在同一目标上嵌套传送时会留下一个遮罩，却没有挂载弹窗内容。 -->
  <div v-if="announcement" class="notice-overlay" role="presentation" @click.self="requestClose">
    <article class="notice-modal" role="dialog" aria-modal="true" :aria-labelledby="titleId" :aria-describedby="forced ? instructionId : undefined">
      <header class="notice-head">
        <div>
          <span v-if="announcement.priority === 'important'" class="notice-level notice-level--important">!!! {{ $t('announcement.important') }}</span>
          <span v-else-if="announcement.priority === 'new'" class="notice-level notice-level--new">NEW {{ $t('announcement.latest') }}</span>
          <span v-else class="notice-level">{{ $t('announcement.notice') }}</span>
          <h2 :id="titleId">{{ announcement.title || $t('announcement.details') }}</h2>
          <time v-if="noticeTime">{{ noticeTime }}</time>
        </div>
        <button v-if="!forced" type="button" class="notice-close" :aria-label="$t('announcement.close')" @click="requestClose">×</button>
      </header>

      <p v-if="forced" :id="instructionId" class="notice-instruction">{{ $t('announcement.readBeforeConfirm') }}</p>
      <div class="notice-body">
        <img
          v-if="announcement.image_url"
          class="notice-image"
          :src="announcement.image_url"
          :alt="announcement.title || $t('announcement.imageAlt')"
        />
        <div v-if="announcement.content" class="notice-content" v-html="announcement.content"></div>
        <p v-else-if="announcement.summary" class="notice-summary">{{ announcement.summary }}</p>
      </div>
      <footer>
        <button type="button" @click="acknowledge">{{ $t('announcement.gotIt') }}</button>
      </footer>
    </article>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AnnouncementItem } from '@/api/aix'

const { t: $t } = useI18n()
const props = withDefaults(defineProps<{ announcement: AnnouncementItem | null; forced?: boolean }>(), { forced: false })
const emit = defineEmits<{ (event: 'close'): void; (event: 'acknowledge'): void }>()
const titleId = 'announcement-detail-title'
const instructionId = 'announcement-read-instruction'
const noticeTime = computed(() => props.announcement?.published_at || props.announcement?.created_at || '')
const requestClose = () => { if (!props.forced) emit('close') }
const acknowledge = () => emit('acknowledge')
watch(() => props.announcement, value => {
  document.body.style.overflow = value ? 'hidden' : ''
}, { immediate: true })
onBeforeUnmount(() => { document.body.style.overflow = '' })
</script>

<style scoped lang="scss">
.notice-overlay {
  position: fixed;
  z-index: 13000;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  /* inset 是增强项；四方向定位保证旧 Android WebView 也能铺满。 */
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

.notice-level--important {
  color: #df1f32;
  animation: urgent-pulse 1.3s ease-in-out infinite;
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
  min-height: 0;
  overflow-y: auto;
  padding: 20px 24px;
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

footer {
  padding: 24px 22px;
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

@keyframes urgent-pulse {
  50% {
    opacity: 0.48;
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
    padding: 18px 20px;
  }

  footer {
    padding: 20px 20px calc(18px + env(safe-area-inset-bottom));
  }
}

@media (prefers-reduced-motion: reduce) {
  .notice-modal,
  .notice-level--important {
    animation: none;
  }
}
</style>
