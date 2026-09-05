<template>
  <div class="withdrawal-page">
    <van-nav-bar
      :title="$t('withdraw.title')"
      left-arrow
      :border="false"
      fixed
      @click-left="router.back()"
    />

    <div class="content">
      <div class="page-header">
        <h1 class="page-title">{{ $t('withdraw.usdtAvailableBalance') }}</h1>
        <p class="page-balance">
          {{ displayAmount(usdtBalance) }}<span class="page-balance-unit">USDT</span>
        </p>
        <p class="page-hint">{{ $t('withdraw.usdtWithdrawHint') }}</p>
      </div>

      <div class="withdraw-form">
        <div class="form-hint-row">
          <p class="form-hint">{{ $t('withdraw.amount') }}</p>
          <button type="button" class="all-btn" @click="handleAllAmount">
            {{ $t('withdraw.all') }}
          </button>
        </div>
        <div class="form-row">
          <input
            class="form-input"
            v-model="amountInput"
            @input="checkAmount"
            type="text"
            inputmode="decimal"
            :placeholder="$t('withdraw.enterAmount')"
          />
          <span class="asset-tag">USDT</span>
        </div>
        <p v-if="amountError" class="error-text">{{ amountError }}</p>

        <button
          class="subscribe-btn custom-btn"
          :disabled="!canSubmit || loading"
          @click="handleWithdrawal"
        >
          {{ loading ? $t('withdraw.processing') : $t('withdraw.confirm') }}
        </button>

        <div class="form-info">
          <p>{{ $t('withdraw.fee') }}: 0 USDT</p>
        </div>
      </div>

      <div class="record-section">
        <div class="section-title-wrap">
          <div class="title-bar"></div>
          <h3 class="section-title">{{ $t('withdraw.details') }}</h3>
        </div>

        <div class="table-card">
          <div class="table-header table-header-4">
            <span>{{ $t('node.amount') }}</span>
            <span>{{ $t('withdraw.received') }}</span>
            <span>{{ $t('withdraw.toAddressShort') }}</span>
            <span>{{ $t('withdraw.status') }}</span>
          </div>
          <div class="order-list" v-for="item in filteredRecords" :key="item.id">
            <div class="table-row table-row-4">
              <span class="amount-cell">
                <strong>{{ displayAmount(item.amount) }}</strong>
                <small>USDT</small>
              </span>
              <span class="amount-cell">
                <strong>{{ displayAmount(item.net_amount) }}</strong>
                <small>USDT</small>
              </span>
              <span class="address-cell">{{ formatAddress(item.to_address) }}</span>
              <span class="status-cell" :class="`is-${String(item.status || '').toLowerCase()}`">
                {{ withdrawStatusText(item.status) }}
                <small v-if="item.tx_hash" class="tx-hint">{{ String(item.tx_hash).slice(0, 10) }}…</small>
                <small class="muted">{{ formatTime(item.created_at) }}</small>
              </span>
            </div>
          </div>
          <div class="empty-state" v-if="!recordLoading && filteredRecords.length === 0">
            <p>{{ $t('withdraw.noRecords') }}</p>
          </div>
          <div class="state-box" v-if="recordLoading">
            <van-loading color="#0052ff" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import userPerson from '@/pinia/person'
import { getWinWithdrawRecords, withdrawUsdt } from '@/api/aix'
import type { WinWithdrawRecord } from '@/api/aix'
import { compareDecimals, displayDecimal, isPositiveDecimal } from '@/tools/decimal'
import { showToast } from 'vant'
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const { t: $t } = useI18n()
const router = useRouter()
const person = userPerson()

const amountInput = ref('')
const loading = ref(false)
const recordLoading = ref(false)
const amountList = ref<WinWithdrawRecord[]>([])
const pollTimer = ref<ReturnType<typeof setInterval> | null>(null)

const usdtBalance = computed(() => String(person.profile?.usdt_withdrawable || '0'))

const filteredRecords = computed(() =>
  amountList.value.filter((item) => String(item.asset || '').toUpperCase() === 'USDT')
)

const hasPendingRecords = computed(() => filteredRecords.value.some((item) => item.status === 'pending'))

const amountError = computed(() => {
  if (!amountInput.value) return ''
  if (!isPositiveDecimal(amountInput.value)) return $t('withdraw.enterAmount')
  if (compareDecimals(amountInput.value, usdtBalance.value) > 0) {
    return $t('withdraw.insufficientUsdt')
  }
  return ''
})

const canSubmit = computed(() => Boolean(amountInput.value) && !amountError.value)

const withdrawStatusText = (status: string) => {
  switch (status) {
    case 'review': return $t('withdraw.statusReview')
    case 'pending': return $t('withdraw.statusPending')
    case 'doing': return $t('withdraw.statusPaying')
    case 'completed': return $t('withdraw.statusCompleted')
    case 'rejected': return $t('withdraw.statusRejected')
    case 'failed': return $t('withdraw.statusFailed')
    default: return status || '-'
  }
}

const displayAmount = (value: unknown) => displayDecimal(value)

const formatAddress = (value?: string) => {
  if (!value) return '-'
  return value.length > 14 ? `${value.slice(0, 6)}...${value.slice(-4)}` : value
}

const formatTime = (value: number) => {
  const date = new Date(Number(value) * 1000)
  if (Number.isNaN(date.getTime())) return ''
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

const getAmountList = async () => {
  recordLoading.value = true
  try {
    const result = await getWinWithdrawRecords()
    amountList.value = result?.records || []
    syncWithdrawPolling()
  } catch {
    amountList.value = []
    stopWithdrawPolling()
  } finally {
    recordLoading.value = false
  }
}

const stopWithdrawPolling = () => {
  if (!pollTimer.value) return
  clearInterval(pollTimer.value)
  pollTimer.value = null
}

const syncWithdrawPolling = () => {
  if (!hasPendingRecords.value) {
    stopWithdrawPolling()
    return
  }
  if (pollTimer.value) return
  pollTimer.value = setInterval(async () => {
    try {
      const result = await getWinWithdrawRecords()
      amountList.value = result?.records || []
      await person.refreshProfile?.()
      if (!hasPendingRecords.value) {
        stopWithdrawPolling()
      }
    } catch {
      stopWithdrawPolling()
    }
  }, 10000)
}

const handleAllAmount = () => {
  if (!isPositiveDecimal(usdtBalance.value)) {
    showToast({
      message: $t('withdraw.insufficientUsdt'),
      position: 'middle',
    })
    return
  }
  amountInput.value = usdtBalance.value
}

const checkAmount = (e: Event) => {
  const target = e.target as HTMLInputElement
  let raw = String(target?.value ?? amountInput.value ?? '')
  raw = raw.replace(/[^\d.]/g, '')
  const parts = raw.split('.')
  if (parts.length > 2) raw = parts[0] + '.' + parts.slice(1).join('')
  if (parts[1] != null && parts[1].length > 18) raw = parts[0] + '.' + parts[1].slice(0, 18)
  amountInput.value = raw
}

const handleWithdrawal = async () => {
  if (loading.value || !canSubmit.value) return
  loading.value = true
  try {
    const result = await withdrawUsdt(amountInput.value)
    person.profile = {
      ...person.profile,
      usdt_withdrawable: result.usdt_withdrawable,
    }
    showToast({
      message: $t('withdraw.submittedProcessing'),
      position: 'middle',
      duration: 2000,
    })
    amountInput.value = ''
    await Promise.allSettled([person.refreshProfile?.(), getAmountList()])
    syncWithdrawPolling()
  } catch (error: any) {
    const code = error?.response?.data?.reason || error?.response?.data?.code
    const messageKey: Record<string, string> = {
      INVALID_AMOUNT: 'withdraw.enterAmount',
      INSUFFICIENT_USDT: 'withdraw.insufficientUsdt',
      FORBIDDEN: 'withdraw.insufficientUsdt',
      INVALID_ADDRESS: 'withdraw.invalidAddress',
    }
    showToast({
      message: messageKey[code]
        ? $t(messageKey[code])
        : (error?.response?.data?.message || $t('withdraw.failed')),
      position: 'middle',
      duration: 2000,
    })
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await Promise.allSettled([
    person.getUser?.(),
    person.refreshProfile?.(),
    getAmountList(),
  ])
})

onUnmounted(() => {
  stopWithdrawPolling()
})
</script>

<style lang="scss" scoped>
.withdrawal-page {
  --accent: #0052ff;
  --accent-bright: #0052ff;
  --accent-deep: #003ec4;
  --accent-dim: rgba(0, 82, 255, .10);
  min-height: 100vh;
  background: var(--ink);
}

.content {
  padding: 76px 20px 40px;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  text-align: left;
  margin-bottom: 16px;
  padding: 18px;
  border: 1px solid rgba(0, 82, 255, .20);
  border-radius: var(--r-lg);
  background: #f2f5ff;

  .page-title {
    font-size: 11.5px;
    font-weight: 500;
    letter-spacing: .04em;
    color: #0052ff;
    margin-bottom: 6px;
  }

  .page-balance {
    font-family: var(--aix-font-display);
    font-size: 30px;
    font-weight: 300;
    line-height: 1.1;
    letter-spacing: -0.02em;
    color: #0052ff;
    font-variant-numeric: tabular-nums;
    margin-top: 0;

    .page-balance-unit {
      margin-left: 7px;
      font-size: 13px;
      font-weight: 500;
      letter-spacing: 0;
      color: rgba(0, 82, 255, .72);
      vertical-align: 4px;
    }
  }

  .page-hint {
    margin-top: 10px;
    font-size: 12px;
    line-height: 1.5;
    color: rgba(0, 82, 255, .72);
  }
}

.withdraw-form {
  margin-bottom: 28px;
}

.form-hint-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.form-hint {
  font-size: 13px;
  color: var(--fog);
  margin: 0;
}

.all-btn {
  border: 0;
  background: transparent;
  color: #0052ff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  padding: 0;
}

.form-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 14px;
  height: 48px;
  border: 1px solid rgba(0, 82, 255, .18);
  border-radius: var(--r-md);
  background: #fff;
}

.form-input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  font-size: 16px;
  color: var(--paper);
}

.asset-tag {
  font-size: 13px;
  font-weight: 600;
  color: #0052ff;
  white-space: nowrap;
}

.error-text {
  margin-top: 8px;
  font-size: 12px;
  color: #e5484d;
}

.subscribe-btn {
  width: 100%;
  margin-top: 18px;
  height: 48px;
  border: 0;
  border-radius: var(--r-pill);
  background: #0052ff;
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;

  &:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
}

.form-info {
  margin-top: 12px;
  font-size: 12px;
  color: var(--fog);
}

.record-section {
  margin-top: 8px;
}

.section-title-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.title-bar {
  width: 3px;
  height: 14px;
  border-radius: 2px;
  background: #0052ff;
}

.section-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--paper);
}

.table-card {
  border: 1px solid rgba(0, 82, 255, .14);
  border-radius: var(--r-lg);
  overflow: hidden;
  background: #fff;
}

.table-header,
.table-row {
  display: grid;
  grid-template-columns: 1.1fr 1.1fr 1.2fr 1.2fr;
  gap: 8px;
  padding: 12px 14px;
  font-size: 12px;
}

.table-header {
  background: #f2f5ff;
  color: rgba(0, 82, 255, .78);
  font-weight: 500;
}

.table-row {
  border-top: 1px solid rgba(0, 82, 255, .08);
  color: var(--paper);
  align-items: start;
}

.amount-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;

  strong {
    font-variant-numeric: tabular-nums;
  }

  small {
    color: var(--fog);
  }
}

.address-cell {
  word-break: break-all;
  color: var(--fog);
}

.status-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;

  .tx-hint,
  .muted {
    color: var(--fog);
    font-size: 11px;
  }

  &.is-completed { color: #1a7f37; }
  &.is-failed,
  &.is-rejected { color: #e5484d; }
  &.is-pending,
  &.is-doing,
  &.is-review { color: #0052ff; }
}

.empty-state,
.state-box {
  padding: 28px 16px;
  text-align: center;
  color: var(--fog);
  font-size: 13px;
}
</style>


