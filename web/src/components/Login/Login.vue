<template>
    <div class="login-view">
        <div class="loading" :class="{ 'loading--error': person.authError }">
            <div v-if="!person.authError" class="loading-spinner"></div>
            <div class="loading-text">
                <div v-if="person.authError" class="error-text">{{ person.authError }}</div>
                <div v-else-if="person.authStage === 'connecting'">{{lang('common.walletConnecting')}}</div>
                <div v-else-if="person.authStage === 'authenticating'">{{lang('common.authorizing')}}</div>
                <div v-else>{{lang('common.contractVerifying')}}</div>
                <p v-if="!person.authError && showRetry" class="loading-hint">{{ lang('common.walletWaitingHint') }}</p>
            </div>
            <button v-if="person.authError || showRetry" class="retry-button" type="button" @click="retry">
                {{ lang('common.retry') }}
            </button>
        </div>
    </div>
</template>
<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
const person = userPerson();
const showRetry = ref(false)
let retryTimer = 0

const armRetry = () => {
  showRetry.value = Boolean(person.authError)
  clearTimeout(retryTimer)
  if (person.authError || person.isLogin) return
  retryTimer = window.setTimeout(() => {
    showRetry.value = true
  }, 8000)
}

const retry = () => {
    showRetry.value = false
    void person.retryAuth().catch(() => undefined)
}

watch(() => [person.authStage, person.authError, person.isLogin], armRetry, { immediate: true })
onBeforeUnmount(() => clearTimeout(retryTimer))
</script>
<style scoped lang="less">
.login-view {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    background: #000;
}

.loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 20px;
}

.loading-spinner {
    width: 40px;
    height: 40px;
    border: 3px solid var(--hair);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 1s linear infinite;
}

@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}

.loading-text {
    color: var(--text);
    font-size: 14px;
    text-align: center;
}

.error-text {
    max-width: 300px;
    line-height: 1.6;
}

.loading-hint {
    max-width: 280px;
    margin: 10px 0 0;
    color: var(--text-2);
    font-size: 12px;
    line-height: 1.6;
}

.retry-button {
    min-width: 96px;
    padding: 9px 18px;
    border: 1px solid var(--accent);
    border-radius: 18px;
    background: transparent;
    color: var(--accent);
    cursor: pointer;
}
</style>
