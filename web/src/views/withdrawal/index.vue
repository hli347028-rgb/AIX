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
      <div class="asset-tabs">
        <button
          type="button"
          class="asset-tab"
          :class="{ active: assetType === 'SDT' }"
          @click="switchAsset('SDT')"
        >
          AIX-USDT
        </button>
        <button
          type="button"
          class="asset-tab"
          :class="{ active: assetType === 'USDT' }"
          @click="switchAsset('USDT')"
        >
          USDT
        </button>
      </div>

      <div class="page-header">
        <!-- 标签走小号，数字才放大：原先整行（含资产名称和“余额”）被一起放大，
             标签和数字同样醒目，等于没有层级。 -->
        <h1 class="page-title">{{ balanceLabel }}</h1>
        <p class="page-balance">
          {{ displayAmount(currentBalance) }}<span class="page-balance-unit">{{ assetLabel }}</span>
        </p>
        <p v-if="assetType === 'USDT'" class="page-hint">{{ $t('withdraw.usdtWithdrawHint') }}</p>
        <p v-else class="page-hint">{{ $t('withdraw.sdtExchangeHint') }}</p>
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
          <span class="asset-tag">{{ assetLabel }}</span>
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
          <p>{{ $t('withdraw.fee') }}: 0 {{ assetLabel }}</p>
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
                <small>{{ recordAssetLabel(item.asset) }}</small>
              </span>
              <span class="amount-cell">
                <strong>{{ displayAmount(item.net_amount) }}</strong>
                <small>{{ recordAssetLabel(item.asset) }}</small>
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
import { getWinWithdrawRecords, withdrawSdt, withdrawUsdt } from '@/api/aix'
import type { WinWithdrawRecord } from '@/api/aix'
import { compareDecimals, displayDecimal, isPositiveDecimal } from '@/tools/decimal'
import { showToast } from 'vant'
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

type AssetType = 'SDT' | 'USDT'

const { t: $t } = useI18n()
const router = useRouter()
const person = userPerson()

const assetType = ref<AssetType>('SDT')
const amountInput = ref('')
const loading = ref(false)
const recordLoading = ref(false)
const amountList = ref<WinWithdrawRecord[]>([])
const pollTimer = ref<ReturnType<typeof setInterval> | null>(null)

const assetLabel = computed(() => {
  if (assetType.value === 'USDT') return 'USDT'
  return 'AIX-USDT'
})
const balanceLabel = computed(() => {
  if (assetType.value === 'USDT') return $t('withdraw.usdtAvailableBalance')
  return $t('withdraw.sdtAvailableBalance')
})

const sdtBalance = computed(() => String(person.profile?.points || person.userinfo?.points || '0'))
const usdtBalance = computed(() => String(person.profile?.usdt_withdrawable || '0'))
const currentBalance = computed(() => {
  if (assetType.value === 'USDT') return usdtBalance.value
  return sdtBalance.value
})

const filteredRecords = computed(() =>
  amountList.value.filter((item) => {
    const asset = String(item.asset || '').toUpperCase()
    if (assetType.value === 'SDT') return asset === 'SDT'
    return asset === 'USDT'
  })
)

const hasPendingRecords = computed(() => filteredRecords.value.some((item) => item.status === 'pending'))

const amountError = computed(() => {
  if (!amountInput.value) return ''
  if (!isPositiveDecimal(amountInput.value)) return $t('withdraw.enterAmount')
  if (compareDecimals(amountInput.value, currentBalance.value) > 0) {
    if (assetType.value === 'USDT') return $t('withdraw.insufficientUsdt')
    return $t('withdraw.insufficientSdt')
  }
  return ''
})

const canSubmit = computed(() => Boolean(amountInput.value) && !amountError.value)

const recordAssetLabel = (asset?: string) => {
  const a = String(asset || '').toUpperCase()
  if (a === 'SDT') return 'AIX-USDT'
  if (a === 'USDT') return 'USDT'
  return a || '-'
}

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

const switchAsset = (next: AssetType) => {
  if (assetType.value === next) return
  assetType.value = next
  amountInput.value = ''
  syncWithdrawPolling()
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
  if (!isPositiveDecimal(currentBalance.value)) {
    const msg =
      assetType.value === 'USDT'
        ? $t('withdraw.insufficientUsdt')
        : $t('withdraw.insufficientSdt')
    showToast({
      message: msg,
      position: 'middle',
    })
    return
  }
  amountInput.value = currentBalance.value
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
    let result: Awaited<ReturnType<typeof withdrawSdt>>
    if (assetType.value === 'USDT') {
      result = await withdrawUsdt(amountInput.value)
    } else {
      result = await withdrawSdt(amountInput.value)
    }
    person.profile = {
      ...person.profile,
      ...(assetType.value === 'USDT'
        ? { usdt_withdrawable: result.usdt_withdrawable }
        : { points: result.points }),
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
      INSUFFICIENT_SDT: 'withdraw.insufficientSdt',
      INSUFFICIENT_USDT: 'withdraw.insufficientUsdt',
      FORBIDDEN: 'withdraw.insufficientUsdt',
      INVALID_ADDRESS: 'withdraw.invalidAddress',
      AIX_WITHDRAW_FORBIDDEN: 'withdraw.aixExchangeHint',
      WIN_WITHDRAW_DISABLED: 'withdraw.winWithdrawDisabled',
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

/* 资产切换沿用顶栏 wallet 的浅蓝底、蓝字、蓝框。 */
.asset-tabs {
  display: flex;
  gap: 6px;
  margin-bottom: 18px;
  padding: 4px;
  border: 1px solid rgba(0, 82, 255, .18);
  border-radius: 999px;
  background: #f2f5ff;
}

.asset-tab {
  flex: 1;
  height: 38px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: transparent;
  color: #0052ff;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition:
    color .18s var(--ease),
    background-color .18s var(--ease);

  &.active {
    background: #0052ff;
    border-color: #0052ff;
    color: var(--on-accent);
    font-weight: 600;
  }

  &:focus-visible {
    outline: 2px solid #0052ff;
    outline-offset: 2px;
  }
}

/* 原本标题/余额/说明三行全部居中且字号接近，读起来没有主次。
   改成左对齐 + 余额放大：可提现余额是这一页唯一需要"一眼看到"的数字。 */
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

    /* 单位跟在大数字后面，但明显降一级，避免和数值抢注意力 */
    .page-balance-unit {
      margin-left: 7px;
      font-family: var(--aix-font);
      font-size: 13px;
      font-weight: 500;
      letter-spacing: .02em;
      color: var(--text-2);
    }
  }

  .page-hint {
    margin: 10px 0 0;
    font-size: 12px;
    color: var(--text-2);
    line-height: 1.6;
  }

  .link-btn {
    margin-left: 4px;
    padding: 0;
    border: 0;
    background: transparent;
    color: #0052ff;
    cursor: pointer;
  }
}

.withdraw-form {
  margin-bottom: 40px;
  padding: 18px;
  border: 1px solid rgba(0, 82, 255, .18);
  border-radius: var(--r-lg);
  background: var(--surface-1);

  .form-hint-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 10px;
  }

  .form-hint {
    margin: 0;
    font-size: 13px;
    color: var(--text);
  }

  .all-btn {
    min-height: 28px;
    padding: 0 12px;
    border: 1px solid rgba(0, 82, 255, .24);
    border-radius: 14px;
    background: #f2f5ff;
    color: #0052ff;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;

    &:hover,
    &:focus-visible {
      outline: none;
      border-color: #0052ff;
      background: #0052ff;
      color: var(--on-accent);
    }
  }

  .form-row {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .form-input {
    flex: 1;
    min-width: 0;
    height: 44px;
    padding: 0 14px;
    border: 1px solid rgba(0, 82, 255, .24);
    border-radius: var(--r-md);
    /* 原为 rgba(0,0,0,.25) 配 color: var(--text)（近黑）——
       25% 黑底压近黑字，这个提现金额输入框此前几乎读不出来。
       提现是资金操作，输入内容必须清楚可读。
       改为浅灰底（--surface-2）+ 近黑字，与全站输入框统一。 */
    background: #f7f9ff;
    color: var(--text);
    font-size: 15px;
    outline: none;
    caret-color: #0052ff;
    -webkit-text-fill-color: var(--text);

    &::placeholder {
      color: var(--text-2);
      -webkit-text-fill-color: var(--text-2);
    }

    &:focus {
      border-color: #0052ff;
      box-shadow: 0 0 0 3px rgba(0, 82, 255, .10);
    }
  }

  .asset-tag {
    flex-shrink: 0;
    padding: 7px 10px;
    border: 1px solid rgba(0, 82, 255, .18);
    border-radius: 14px;
    background: #f2f5ff;
    color: #0052ff;
    font-size: 11px;
    font-weight: 600;
  }

  .error-text {
    margin: 8px 0 0;
    color: var(--down);
    font-size: 12px;
  }

  .form-info {
    margin-top: 12px;
    display: flex;
    flex-direction: column;
    gap: 4px;

    p {
      margin: 0;
      font-size: 12px;
      color: var(--text-2);
    }
  }
}

.subscribe-btn {
  width: 100%;
  min-height: 44px;
  margin-top: 18px;
  padding: 8px 20px;
  background: #0052ff;
  color: var(--on-accent);
  border: 1px solid #0052ff;
  border-radius: 18px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background-color var(--t-fast) var(--ease),
    border-color var(--t-fast) var(--ease),
    transform var(--t-fast) var(--ease);

  &:hover:not(:disabled) {
    border-color: #003ec4;
    background: #003ec4;
  }

  &:disabled {
    opacity: 1;
    cursor: not-allowed;
    border-color: var(--hair);
    background: var(--surface-3);
    color: var(--text-2);
  }

  &:active:not(:disabled) { transform: scale(.985); }
}

.record-section {
  margin-top: 20px;

  .section-title-wrap {
    margin: 0 0 12px;
  }
}

.table-card {
  margin-top: 10px;
  min-height: 220px;
  overflow-x: auto;
  overflow-y: hidden;
  border: 1px solid rgba(0, 82, 255, .18);
  border-radius: var(--r-lg);
  background: var(--surface-1);
  padding: 0;

  .table-header {
    display: grid;
    grid-template-columns: 1fr 1fr .9fr 1.1fr;
    gap: 6px;
    align-items: center;
    min-height: 42px;
    padding: 0 8px;
    background: #f2f5ff;
    border-bottom: 1px solid rgba(0, 82, 255, .18);

    span {
      min-width: 0;
      text-align: center;
      font-size: 10px;
      color: #0052ff;
      font-weight: 600;
      letter-spacing: .04em;
    }
  }

  .order-list {
    .table-row {
      display: grid;
      grid-template-columns: 1fr 1fr .9fr 1.1fr;
      gap: 6px;
      align-items: center;
      min-height: 62px;
      padding: 10px 8px;
      border-bottom: 1px solid var(--hair);
      transition: background-color var(--t-fast) var(--ease);

      &:hover { background: #f7f9ff; }

      span {
        min-width: 0;
        text-align: center;
        font-size: 12px;
        color: var(--text);
      }
    }

    &:last-of-type .table-row { border-bottom: 0; }
  }

  .address-cell {
    font-size: 11px !important;
    color: var(--text-2) !important;
    overflow-wrap: anywhere;
  }

  .amount-cell {
    font-family: var(--aix-font-display);
    font-variant-numeric: tabular-nums;

    strong,
    small {
      display: block;
    }

    strong {
      font-size: 12px;
      font-weight: 600;
      color: var(--text);
      white-space: nowrap;
    }

    small {
      margin-top: 3px;
      font-family: var(--aix-font);
      font-size: 10px;
      font-weight: 500;
      color: var(--text-3);
      white-space: nowrap;
    }
  }

  .status-cell {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    color: var(--text-2) !important;
    font-size: 11px !important;

    &.is-review,
    &.is-pending,
    &.is-doing { color: #0052ff !important; }

    &.is-completed { color: var(--up-readable) !important; }

    &.is-rejected,
    &.is-failed { color: var(--down) !important; }

    .tx-hint {
      font-size: 10px;
      color: var(--text-2);
    }
  }

  .muted {
    font-size: 11px;
    color: var(--text-3);
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 250px;

    p {
      margin-top: 8px;
      font-size: 12px;
      color: var(--text-3);
    }
  }

  .state-box {
    height: 120px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
}

@media (max-width: 420px) {
  .table-card {
    .table-header,
    .table-row {
      min-width: 460px;
      grid-template-columns: minmax(92px, 1fr) minmax(92px, 1fr) minmax(86px, .9fr) minmax(108px, 1.1fr);
    }
  }
}
</style>
