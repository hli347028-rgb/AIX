<template>
  <div class="withdrawal-page">
    <Header />

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
              <span>{{ displayAmount(item.amount) }} {{ recordAssetLabel(item.asset) }}</span>
              <span>{{ displayAmount(item.net_amount) }} {{ recordAssetLabel(item.asset) }}</span>
              <span class="address-cell">{{ formatAddress(item.to_address) }}</span>
              <span class="status-cell">
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
            <van-loading color="#8A9096" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Header from '@/components/Header.vue'
import userPerson from '@/pinia/person'
import { getWinWithdrawRecords, withdrawSdt, withdrawUsdt } from '@/api/aix'
import type { WinWithdrawRecord } from '@/api/aix'
import { compareDecimals, displayDecimal, isPositiveDecimal } from '@/tools/decimal'
import { showToast } from 'vant'
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'

type AssetType = 'SDT' | 'USDT'

const { t: $t } = useI18n()
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
@use '@/style/variables.scss' as *;

.withdrawal-page {
  min-height: 100vh;
  background: var(--ink);
}

.content {
  padding: 90px 20px 40px;
  max-width: 1200px;
  margin: 0 auto;
}

/* 资产切换：原本是三个各自独立、彼此有 10px 间距的方块按钮，
   和 node / wallet / transfer 里的分段控件是两种语言，同一个 App 里
   出现两种切换器就显得没人统一收口。这里改成同一套"凹槽 + 抬起滑块"。 */
.asset-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 18px;
  padding: 4px;
  border: 1px solid var(--hair);
  border-radius: 999px;
  background: var(--surface-2);
  box-shadow: inset 0 2px 6px rgba(0, 0, 0, 0.07);
}

.asset-tab {
  flex: 1;
  height: 38px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: transparent;
  color: var(--text-3);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition:
    color .18s var(--ease),
    background-color .18s var(--ease);

  &.active {
    background: linear-gradient(180deg, var(--surface-3) 0%, var(--surface-2) 100%);
    border-color: var(--hair-2);
    color: var(--text);
    font-weight: 600;
    box-shadow:
      0 2px 8px rgba(0, 0, 0, 0.07);
  }
}

/* 原本标题/余额/说明三行全部居中且字号接近，读起来没有主次。
   改成左对齐 + 余额放大：可提现余额是这一页唯一需要"一眼看到"的数字。 */
.page-header {
  text-align: left;
  margin-bottom: 24px;

  .page-title {
    font-size: 11.5px;
    font-weight: 500;
    letter-spacing: .04em;
    color: var(--text-3);
    margin-bottom: 6px;
  }

  .page-balance {
    font-family: var(--aix-font-display);
    font-size: 30px;
    font-weight: 300;
    line-height: 1.1;
    letter-spacing: -0.02em;
    color: var(--text);
    font-variant-numeric: tabular-nums;
    margin-top: 0;

    /* 单位跟在大数字后面，但明显降一级，避免和数值抢注意力 */
    .page-balance-unit {
      margin-left: 7px;
      font-family: var(--aix-font);
      font-size: 13px;
      font-weight: 500;
      letter-spacing: .02em;
      color: var(--text-3);
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
    color: $brand-primary;
    cursor: pointer;
  }
}

.withdraw-form {
  margin-bottom: 40px;

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
    padding: 0;
    border: none;
    background: transparent;
    color: $brand-primary;
    font-size: 13px;
    cursor: pointer;
  }

  .form-row {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .form-input {
    flex: 1;
    height: 44px;
    padding: 0 14px;
    border: 1px solid var(--hair);
    border-radius: var(--r-md);
    /* 原为 rgba(0,0,0,.25) 配 color: var(--text)（近黑）——
       25% 黑底压近黑字，这个提现金额输入框此前几乎读不出来。
       提现是资金操作，输入内容必须清楚可读。
       改为浅灰底（--surface-2）+ 近黑字，与全站输入框统一。 */
    background: var(--surface-2);
    color: var(--text);
    font-size: 15px;
    outline: none;
    caret-color: $brand-primary;
    -webkit-text-fill-color: var(--text);

    &::placeholder {
      color: var(--text-2);
      -webkit-text-fill-color: var(--text-2);
    }

    &:focus {
      border-color: $brand-primary;
    }
  }

  .asset-tag {
    flex-shrink: 0;
    color: var(--accent-bright);
    font-weight: 600;
  }

  .error-text {
    margin: 8px 0 0;
    color: #f17b7b;
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
  margin-top: 18px;
  padding: 8px 20px;
  background: $gradient-primary;
  color: $text-inverse;
  border: none;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 400;
  cursor: pointer;
  transition: all 0.3s ease;

  &:hover:not(:disabled) {
    background: linear-gradient(135deg, $brand-primary-light 0%, $brand-primary 100%);
    transform: translateY(-2px);
    box-shadow: var(--shadow-1);
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background: var(--surface-2);
    color: var(--text-2);
  }
}

.record-section {
  margin-top: 20px;

  .section-title-wrap {
    position: relative;
    margin-bottom: 10px;
    margin-left: 10px;
    display: flex;
    align-items: center;

    .title-bar {
      position: absolute;
      left: -10px;
      top: 50%;
      width: 4px;
      height: 16px;
      border-radius: 2px;
      background: linear-gradient(180deg, var(--accent) 0%, var(--text-3) 100%);
      transform: translateY(-50%);
    }

    .section-title {
      margin: 0 0 0 8px;
      font-size: 16px;
      font-weight: bold;
      color: var(--text);
    }
  }
}

.table-card {
  margin-top: 10px;
  min-height: 300px;
  overflow: hidden;
  border: 1px solid $border-color;
  border-radius: 11px;
  background: var(--surface-1);
  backdrop-filter: blur(10px);
  padding: 11px 0;

  .table-header {
    display: flex;
    align-items: center;
    background: var(--ink-deep);
    padding: 8px 0;
    margin: -11px 0 0;

    span {
      flex: 1;
      text-align: center;
      font-size: 10px;
      color: $text-muted;
    }
  }

  .order-list {
    .table-row {
      display: flex;
      align-items: center;
      padding: 12px 0;
      border-bottom: 1px solid $border-light;

      &:last-child {
        border-bottom: none;
      }

      span {
        flex: 1;
        text-align: center;
        font-size: 13px;
        color: $text-primary;
      }
    }
  }

  .address-cell {
    font-size: 11px !important;
    color: $text-muted !important;
  }

  .status-cell {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;

    .tx-hint {
      font-size: 10px;
      color: $text-muted;
    }
  }

  .muted {
    font-size: 11px;
    color: $text-muted;
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
      color: $text-muted;
    }
  }

  .state-box {
    height: 120px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
}
</style>
