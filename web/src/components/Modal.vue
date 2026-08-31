<template>
  <Transition name="modal-fade">
    <div v-if="visible" class="modal-overlay" @click="handleOverlayClick">
      <div class="modal" @click.stop>
        <div class="modal-header">
          <!-- <h3 class="modal-title">{{ title }}</h3> -->
          <button class="modal-close" @click="close">×</button>
        </div>
        <div class="modal-body">
          <p class="modal-message">{{ message }}</p>
        </div>
        <div class="modal-footer" v-if="showConfirmButton">
          <button class="modal-btn" @click="handleConfirm">{{ confirmText }}</button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { watch } from 'vue'

const props = withDefaults(
  defineProps<{
    visible: boolean
    title?: string
    message?: string
    confirmText?: string
    closeOnOverlay?: boolean
    showConfirmButton?: boolean
    autoCloseDelay?: number
  }>(),
  {
    title: '',
    message: '',
    confirmText: '',
    closeOnOverlay: true,
    showConfirmButton: true,
    autoCloseDelay: 0
  }
)

const emit = defineEmits<{
  close: []
  confirm: []
}>()

const close = () => {
  emit('close')
}

const handleConfirm = () => {
  emit('confirm')
  close()
}

const handleOverlayClick = () => {
  if (props.closeOnOverlay) {
    close()
  }
}

watch(() => props.visible, (newVal) => {
  if (newVal && props.autoCloseDelay > 0) {
    setTimeout(() => {
      close()
    }, props.autoCloseDelay)
  }
})
</script>

<style lang="scss" scoped>
@use '@/style/variables.scss' as *;

.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 3000;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.modal {
  /* 原本是近黑半透明面板（rgba(8,19,30,.95)）+ 白色描边。
     翻成 Base 的浅色浮层：纯白 + 发丝边 + 12px 圆角 + 轻投影。 */
  background: var(--surface-1);
  border: 1px solid var(--hair);
  border-radius: var(--r-lg);
  width: 100%;
  max-width: 320px;
  box-shadow: var(--shadow-3);
  animation: modalSlideIn 0.3s ease;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  // padding: 20px 24px;
  // border-bottom: 1px solid rgba(255, 255, 255, 0.1);

  .modal-title {
    font-size: 18px;
    font-weight: 700;
    color: var(--text);
    margin: 0;
  }

  .modal-close {
    width: 32px;
    height: 32px;
    background: transparent;
    border: none;
    color: var(--text-3);
    font-size: 24px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color var(--t-fast) var(--ease);

    &:hover {
      color: var(--text);
    }
  }
}

.modal-body {
  padding: 24px;

  .modal-message {
    font-size: 15px;
    color: var(--text-2);
    line-height: 1.6;
    margin: 0;
    text-align: center;
  }
}

.modal-footer {
  padding: 16px 24px 24px;
  display: flex;
  justify-content: center;

  /* Base 的按钮：纯蓝实心 + 白字 + 全圆角，无渐变无投影。
     原本写的是 `background: var(--accent); color: #000` ——
     令牌翻转后 --accent 成了纯蓝 #0000FF，黑字压在上面只有约 2.4:1，
     基本读不出来。这是"令牌值变了但配对的前景色没跟着变"的典型后果，
     所以前景色改用 --on-accent（跟着 accent 一起定义的配对色，8.6:1）。 */
  .modal-btn {
    padding: 12px 32px;
    background: var(--accent);
    color: var(--on-accent);
    border: none;
    border-radius: var(--r-pill);
    font-size: 16px;
    font-weight: 600;
    cursor: pointer;
    transition:
      background-color var(--t-fast) var(--ease),
      transform var(--t-fast) var(--ease);

    &:hover {
      background: var(--accent-deep);
    }

    &:active {
      transform: scale(0.98);
    }
  }
}

.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.3s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
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
