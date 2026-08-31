<template>
  <Teleport to="body">
    <div v-if="visible" class="announcement-overlay" @click="handleClose">
      <div class="announcement-modal" @click.stop>
        <div class="modal-close" @click="handleClose">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 6L6 18M6 6l12 12" />
          </svg>
        </div>

        <h1 class="modal-title">{{ $t('announcement.title') }}</h1>

        <div class="modal-content">
         <p>{{ $t('announcement.content') }}</p>
        </div>

        <div class="modal-footer">
          <button class="confirm-btn" @click="handleClose">{{ $t('announcement.gotIt') }}</button>
          <p class="no-remind" @click="handleNoRemind">{{ $t('announcement.noRemind') }}</p>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'

const visible = ref(false)
const STORAGE_KEY = 'announcement_dismissed'

onMounted(() => {
  const dismissed = localStorage.getItem(STORAGE_KEY)
  if (!dismissed) {
    visible.value = true
  }
})

watch(() => visible.value, (val) => {
  document.body.style.overflow = val ? 'hidden' : ''
})

const handleClose = () => {
  visible.value = false
}

const handleNoRemind = () => {
  localStorage.setItem(STORAGE_KEY, 'true')
  visible.value = false
}
</script>

<style lang="scss" scoped>
@use '@/style/variables.scss' as *;

.announcement-overlay {
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.8);
  padding: 20px;
  backdrop-filter: blur(8px);
}

.announcement-modal {
  position: relative;
  width: 90%;
  max-width: 380px;
  max-height: 70vh;
  /* 公告面板改为 Base 的浅色浮层：纯白 + 发丝边 + 轻投影。
     原本三处都是深色专用写法：
       145° 近黑渐变（rgba(8,19,30,.98)→rgba(13,27,42,.95)）
       30% 白描边      —— 白底上完全不可见
       24px 白色外发光 —— 白底上不可见，且 Base 全站零发光
     注意外层遮罩 rgba(0,0,0,.8) 保持不动：
     浅色主题的弹窗遮罩同样应该是深色，那一处本来就是对的。 */
  background: var(--surface-1);
  border-radius: var(--r-lg);
  padding: 28px 24px;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--hair);
  box-shadow: var(--shadow-3);
  animation: modalSlideIn 0.3s ease;

  .modal-close {
    position: absolute;
    top: 16px;
    right: 16px;
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--surface-2);
    border-radius: 50%;
    color: var(--text-3);
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover {
      background: var(--surface-3);
      color: var(--text);
    }
  }

  /* 这里原本是 background-clip: text 的渐变文字 —— 和首页标题、资产数值
     用的是同一种手法，我在那两处已经移除：它用材质掩盖排版，而且渐变的暗端
     必然比纯白更接近背景，等于主动压低了标题对比度。改为实心近白。 */
  .modal-title {
    font-size: 20px;
    font-weight: 600;
    color: var(--text);
    text-align: center;
    margin: 0 0 20px;
    letter-spacing: -0.01em;
  }

  .modal-subtitle {
    font-size: 14px;
    font-weight: normal;
    color: $brand-primary;
    text-align: center;
    margin: 0 0 16px;
  }

  .modal-content {
    flex: 1;
    overflow-y: auto;
    max-height: 50vh;
    padding: 0 4px;
    margin-bottom: 20px;

    &::-webkit-scrollbar {
      width: 4px;
    }

    &::-webkit-scrollbar-thumb {
      background: var(--hair-3);
      border-radius: 2px;
    }

    p {
      margin: 0 0 12px;
      font-size: 14px;
      color: var(--text-2);
      line-height: 1.7;
      white-space: pre-wrap;
      text-align: center;
    }

    strong {
      color: $brand-primary;
      font-weight: 600;
    }
  }

  .modal-footer {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;

    .confirm-btn {
      width: 100%;
      padding: 14px 0;
      background: var(--accent);
      /* 又一处"蓝底配黑字"：--accent 现在是纯蓝 #0000FF，
         压黑字只有约 2.4:1，基本读不出来。改用配对色 --on-accent（8.6:1）。
         这个 bug 在 Modal.vue 里出现过同样一份 —— 说明"背景用令牌、
         前景写死"的写法在项目里成对复制过。 */
      color: var(--on-accent);
      font-size: 16px;
      font-weight: 600;
      letter-spacing: 0.5px;
      border: none;
      border-radius: var(--r-pill);
      cursor: pointer;
      /* transition: all 换成显式属性：all 会把 layout 属性也纳入动画，
         是低端机掉帧的常见来源。时长对齐 Base 实测的 0.15s。 */
      transition:
        background-color var(--t-fast) var(--ease),
        transform var(--t-fast) var(--ease);

      &:hover {
        /* 原本是"上浮 2px + 白色外发光"。白发光在白底不可见，
           且 Base 的按钮 hover 只做颜色加深，不做位移。 */
        background: var(--accent-deep);
      }

      &:active {
        transform: scale(0.98);
      }
    }

    .no-remind {
      margin: 0;
      font-size: 13px;
      color: var(--text-3);
      cursor: pointer;
      transition: color 0.2s ease;

      &:hover {
        color: var(--text-2);
      }
    }
  }
}

@keyframes modalSlideIn {
  from {
    transform: translateY(-20px) scale(0.95);
    opacity: 0;
  }
  to {
    transform: translateY(0) scale(1);
    opacity: 1;
  }
}
</style>
