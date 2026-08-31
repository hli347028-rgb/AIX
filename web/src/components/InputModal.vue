<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div v-if="visible" class="modal-overlay" @click="handleOverlayClick">
        <div class="modal" @click.stop>
          <div class="modal-body">
            <p class="modal-message">{{ message }}</p>
            <div class="input-wrapper">
              <input
                v-model="inputValue"
                type="text"
                :placeholder="placeholder"
                class="modal-input"
              />
            </div>
          </div>
          <div class="modal-footer">
            <button class="modal-btn confirm-btn" @click="handleConfirm">{{ confirmText }}</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    visible: boolean
    message?: string
    placeholder?: string
    confirmText?: string
    closeOnOverlay?: boolean
  }>(),
  {
    message: '',
    placeholder: '',
    confirmText: '',
    closeOnOverlay: true
  }
)

const emit = defineEmits<{
  close: []
  confirm: [value: string]
}>()

const inputValue = ref('')

watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      inputValue.value = ''
    }
  }
)

const close = () => {
  emit('close')
}

const handleConfirm = () => {
  emit('confirm', inputValue.value)
  close()
}

const handleOverlayClick = () => {
  if (props.closeOnOverlay) {
    close()
  }
}
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
  /* 与 Modal.vue / AnnouncementModal.vue 统一为 Base 的浅色浮层。
     原本是近黑半透明面板 + 10% 白描边（白底上不可见）。 */
  background: var(--surface-1);
  border: 1px solid var(--hair);
  border-radius: var(--r-lg);
  width: 100%;
  max-width: 360px;
  box-shadow: var(--shadow-3);
  animation: modalSlideIn var(--t-slow) var(--ease-out-q);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--hair);

  .modal-title {
    font-size: 18px;
    font-weight: 600;
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
    font-size: 14px;
    color: var(--text-2);
    line-height: 1.6;
    margin: 0 0 16px 0;
  }

  .input-wrapper {
    .modal-input {
      width: 100%;
      padding: 12px 16px;
      background: var(--surface-2);
      border: 1px solid var(--hair);
      border-radius: 12px;
      color: var(--text);
      font-size: 14px;
      outline: none;
      transition: border-color var(--t-fast) var(--ease), background-color var(--t-fast) var(--ease);

      &::placeholder {
        color: var(--text-3);
      }

      &:focus {
        border-color: $brand-primary;
        background: var(--surface-2);
      }
    }
  }
}

.modal-footer {
  padding: 16px 24px 24px;
  display: flex;
  gap: 12px;

  .modal-btn {
    flex: 1;
    padding: 12px 0;
    border: none;
    border-radius: 12px;
    font-size: 16px;
    font-weight: 600;
    cursor: pointer;
    transition: border-color var(--t-fast) var(--ease), background-color var(--t-fast) var(--ease);
  }

  .cancel-btn {
    background: var(--surface-3);
    color: var(--text);

    &:hover {
      background: var(--surface-3);
    }
  }

  .confirm-btn {
    background: var(--accent);
    color: var(--on-accent);

    &:hover {
      transform: none;
    }

    &:active {
      transform: translateY(0);
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
