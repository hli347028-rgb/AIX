<template>
  <div class="support-page">
    <van-nav-bar
      :title="$t('support.title')"
      left-arrow
      :border="false"
      fixed
      @click-left="router.back()"
    />

    <main class="page-main">
      <p class="lead">{{ $t('support.intro') }}</p>

      <form class="feedback-form" @submit.prevent="onSubmit">
        <label class="field-label" for="feedback-content">{{ $t('support.contentLabel') }}</label>
        <textarea
          id="feedback-content"
          v-model="content"
          class="feedback-input"
          maxlength="500"
          rows="8"
          :placeholder="$t('support.contentPlaceholder')"
        />
        <p class="char-count">{{ content.trim().length }} / 500</p>

        <button
          type="submit"
          class="aix-btn feedback-submit"
          :disabled="!canSubmit || submitting"
        >
          {{ submitting ? $t('support.submitting') : $t('support.submit') }}
        </button>
      </form>

      <p class="notice">{{ $t('support.notice') }}</p>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { showFailToast, showSuccessToast, showToast } from 'vant'
import { errMsg, submitFeedback } from '@/api/aix'
import userPerson from '@/pinia/person'

const router = useRouter()
const { t: $t } = useI18n()
const person = userPerson()

const content = ref('')
const submitting = ref(false)
const canSubmit = computed(() => content.value.trim().length >= 4)

const onSubmit = async () => {
  const text = content.value.trim()
  if (!text) {
    showToast($t('support.contentRequired'))
    return
  }
  if (text.length < 4) {
    showToast($t('support.contentTooShort'))
    return
  }
  if (submitting.value) return
  submitting.value = true
  try {
    await submitFeedback({
      content: text,
      address: person.address || '',
    })
    content.value = ''
    showSuccessToast($t('support.success'))
  } catch (error) {
    showFailToast(errMsg(error, $t('support.failed')))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped lang="scss">
@use '@/style/variables.scss' as *;

.support-page {
  min-height: 100vh;
  background: var(--ink);
  color: var(--text);
}

.page-main {
  padding: 76px 20px 40px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.lead {
  margin: 0;
  color: var(--text-2);
  font-size: 14px;
  line-height: 1.7;
}

.feedback-form {
  padding: 18px;
  border: 1px solid rgba(0, 82, 255, 0.18);
  border-radius: var(--r-lg);
  background: var(--surface-1);
}

.field-label {
  display: block;
  margin-bottom: 9px;
  color: var(--text);
  font-size: 12px;
}

.feedback-input {
  display: block;
  box-sizing: border-box;
  width: 100%;
  min-height: 168px;
  padding: 12px 14px;
  border: 1px solid var(--hair);
  border-radius: 12px;
  background: var(--surface-1);
  color: var(--text);
  font: inherit;
  font-size: 14px;
  line-height: 1.65;
  resize: vertical;

  &:focus {
    outline: none;
    border-color: #0052ff;
  }

  &::placeholder {
    color: var(--text-2);
  }
}

.char-count {
  margin: 8px 0 0;
  color: var(--text-2);
  font-size: 12px;
  text-align: right;
}

.feedback-submit {
  width: 100%;
  margin-top: 18px;
  min-height: 44px;
}

.notice {
  margin: 0;
  color: var(--text-2);
  font-size: 12px;
  line-height: 1.65;
}
</style>
