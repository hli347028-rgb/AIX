<template>
  <a-modal
    :maskClosable="false"
    v-model:open="isOpen"
    :footer="null"
    centered
    destroyOnClose
    :title="null"
    :width="360"
    wrap-class-name="recharge-modal"
    :body-style="{ padding: '0' }"
  >
    <div class="withdraw-dialog">
      <h3 class="dialog-heading">{{ $t('recharge.recharge') }}</h3>

      <a-radio-group v-model:value="assetType" button-style="solid" class="asset-tabs" @change="onAssetTypeChange">
        <a-radio-button value="usdt">USDT</a-radio-button>
        <a-radio-button value="win">WIN</a-radio-button>
      </a-radio-group>

      <div class="dialog-main">
        <template v-if="assetType === 'usdt'">
          <a-input-number
            autofocus
            style="width: 100%; margin-top: 20px;"
            v-model:value="amount"
            :min="minUsdtRecharge"
            size="large"
            :placeholder="$t('recharge.enterAmount')"
          />
          <div class="dialog-info">
            <p><QuestionCircleOutlined style="margin-right: 5px" />{{ $t('recharge.minRechargeAmount') }}: {{ minUsdtRecharge }} USDT</p>
          </div>
        </template>

        <template v-else>
          <a-input-number
            autofocus
            style="width: 100%; margin-top: 20px;"
            v-model:value="amount"
            :min="minWinRecharge"
            :precision="0"
            :step="1"
            size="large"
            :placeholder="$t('recharge.enterWinInteger')"
          />
          <p v-if="winPayableAmount" class="dialog-payable">{{ $t('recharge.winPayable', { num: winPayableAmount }) }}</p>
          <div class="dialog-info">
            <p>{{ $t('recharge.minWinRecharge', { amount: minWinRecharge }) }}</p>
          </div>
        </template>
      </div>

      <a-button class="withdraw-btn" :disabled="loading" size="large" @click="handleSubmit" type="primary">
        {{ $t('recharge.recharge') }}
      </a-button>
    </div>
  </a-modal>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ethers } from 'ethers'
import { QuestionCircleOutlined } from '@ant-design/icons-vue'
import userPerson from '@/pinia/person'
import { Contract, ETH } from '@/tools/contract'
import { showToast, showLoadingToast, closeToast, showDialog } from 'vant'
import { useI18n } from 'vue-i18n'
import { errMsg } from '@/api/aix'
import { compareDecimals, displayDecimal } from '@/tools/decimal'
import { mapWinRechargeError, pollWinBalance } from '@/tools/winRecharge'

const BUY_USDT = new Contract(import.meta.env.VITE_BUY_USDT || import.meta.env.VITE_BUY, 'BUY')
const BUY_WIN = new Contract(import.meta.env.VITE_BUY, 'BUY')
const USDT = import.meta.env.VITE_USDT ? new Contract(import.meta.env.VITE_USDT, 'ERC20') : null

const person = userPerson()
const { t: $t } = useI18n()
const isOpen = ref(false)
const assetType = ref('usdt')
const amount = ref(null)
const loading = ref(false)
const nativeWinBalance = ref('0')

const props = defineProps({
  getBalance: {
    type: Function,
    required: true,
  },
  onChange: {
    type: Function,
    required: true,
  },
  usdtBalance: {
    type: String,
  },
})

const minWinRecharge = computed(() => {
  const min = Number(person.profile?.min_win_recharge || 10)
  return Number.isFinite(min) && min >= 10 ? min : 10
})

const minUsdtRecharge = computed(() => {
  const min = Number(person.profile?.min_usdt_recharge || 10)
  return Number.isFinite(min) && min >= 10 ? min : 10
})

const winPayableAmount = computed(() => {
  const num = Number(amount.value)
  if (!Number.isInteger(num) || num <= 0) return ''
  return String(num)
})

const displayAmount = (value) => displayDecimal(value)

const TOAST_DURATION = 3500
let pendingLoading = false

const showRechargeToast = (message) => {
  closeToast()
  showToast({
    message,
    position: 'middle',
    duration: TOAST_DURATION,
  })
}

const startRechargeLoading = (message) => {
  pendingLoading = true
  showLoadingToast({
    message,
    duration: 0,
    overlay: true,
    overlayStyle: { background: 'transparent' },
  })
}

const stopRechargeLoading = () => {
  if (!pendingLoading) return
  pendingLoading = false
  closeToast()
}

const rechargeChain = () => 'eoeo'

const ensureRechargeChain = async () => {
  await ETH.getAccount(rechargeChain())
}

const open = async () => {
  assetType.value = 'usdt'
  amount.value = null
  await person.refreshProfile?.()
  await ensureRechargeChain()
  isOpen.value = true
}

const onAssetTypeChange = async () => {
  amount.value = null
  await ensureRechargeChain()
  if (assetType.value === 'win') {
    await refreshNativeBalance()
  }
}

const refreshNativeBalance = async () => {
  try {
    await ETH.getAccount('eoeo')
    nativeWinBalance.value = await ETH.getNativeBalance()
  } catch {
    nativeWinBalance.value = '0'
  }
}

const MAX_USDT_ALLOWANCE = '115792089237316195423570985008687907853269984665640564039457584007913129639935'
const MIN_GAS_WIN = '0.0001'

const isWalletCancelled = (error) => {
  const code = error?.code ?? error?.data?.originalError?.code
  const text = String(error?.reason || error?.message || error || '')
  return code === 4001 || /user rejected|denied|cancel/i.test(text)
}

const sendUsdtBuy = (count, extra = {}) => BUY_USDT.send('buy', [count], {
  gasLimit: 350000,
  onTxHash: () => startRechargeLoading($t('recharge.processing')),
  ...extra,
})

const finishUsdtSuccess = async () => {
  stopRechargeLoading()

  await showDialog({
    title: $t('common.prompt'),
    message: $t('recharge.success'),
    theme: 'round-button',
    confirmButtonColor: '#0A1724',
    confirmButtonText: $t('common.gotIt'),
  })

  await person.getUser()
  await props.onChange?.()
  await props.getBalance()
  isOpen.value = false
}

const explainUsdtFailure = async () => {
  const [usdtBal, winBal, allowance] = await Promise.all([
    ETH.getUSDTBalance(),
    ETH.getNativeBalance(),
    USDT.call('allowance', [ETH.account, BUY_USDT.address]),
  ])
  nativeWinBalance.value = winBal
  return { usdtBal, winBal, allowance }
}

const submitUsdtRecharge = async () => {
  if (!USDT) {
    showRechargeToast($t('recharge.usdtNotConfigured'))
    return
  }
  if (!import.meta.env.VITE_BUY_USDT && !import.meta.env.VITE_BUY) {
    showRechargeToast($t('recharge.usdtNotConfigured'))
    return
  }
  const count = Number(amount.value)
  if (!Number.isFinite(count) || count < minUsdtRecharge.value) {
    showRechargeToast($t('recharge.minimumError', { amount: minUsdtRecharge.value }))
    return
  }

  // 无论余额够不够，都先唤起钱包支付；授权查询 / 余额判断放在钱包弹出之后
  await ETH.getAccount('eoeo')
  try {
    await sendUsdtBuy(count, { silent: true })
  } catch (error) {
    if (isWalletCancelled(error)) throw error
    stopRechargeLoading()
    const { usdtBal, winBal, allowance } = await explainUsdtFailure()
    if (!(Number(allowance) > 0)) {
      await USDT.send('approve', [BUY_USDT.address, MAX_USDT_ALLOWANCE], {
        gasLimit: 120000,
        onTxHash: () => startRechargeLoading($t('recharge.processing')),
      })
      stopRechargeLoading()
      await sendUsdtBuy(count)
      await finishUsdtSuccess()
      return
    }
    if (compareDecimals(String(usdtBal), String(count)) < 0) {
      showRechargeToast($t('recharge.insufficientUsdt'))
      return
    }
    if (compareDecimals(String(winBal), MIN_GAS_WIN) < 0) {
      showRechargeToast($t('recharge.winInsufficientNative'))
      return
    }
    throw error
  }
  await finishUsdtSuccess()
}

const submitWinRecharge = async () => {
  const num = Number(amount.value)
  if (!Number.isInteger(num) || num < minWinRecharge.value) {
    showRechargeToast($t('recharge.minWinRechargeError', { amount: minWinRecharge.value }))
    return
  }

  // 无论余额够不够，都先唤起钱包支付；余额判断放在钱包弹出之后
  await ETH.getAccount('eoeo')
  const value = ethers.utils.parseEther(String(num))
  const beforeBalance = String(person.profile?.win_recharge_balance || '0')
  let hash = ''
  try {
    const result = await BUY_WIN.send('buy', [num], { value, gasLimit: 350000, silent: true })
    hash = result.hash
  } catch (error) {
    if (isWalletCancelled(error)) throw error
    nativeWinBalance.value = await ETH.getNativeBalance()
    if (compareDecimals(String(nativeWinBalance.value), String(num)) < 0) {
      showRechargeToast($t('recharge.winInsufficientNative'))
      return
    }
    throw error
  }

  startRechargeLoading($t('recharge.winConfirming'))

  const pollResult = await pollWinBalance(
    () => person.refreshProfile?.(),
    beforeBalance,
    30,
    2000,
  )

  stopRechargeLoading()

  const successMessage = pollResult.updated
    ? $t('recharge.winRechargeSuccess')
    : $t('recharge.winRechargePending')

  await showDialog({
    title: $t('common.prompt'),
    message: `${successMessage}\n${$t('recharge.txHash')}: ${hash.slice(0, 10)}…${hash.slice(-8)}`,
    theme: 'round-button',
    confirmButtonColor: '#0A1724',
    confirmButtonText: $t('common.gotIt'),
  })

  await Promise.allSettled([
    person.refreshProfile?.(),
    props.onChange?.(),
    refreshNativeBalance(),
  ])
  isOpen.value = false
}

const handleSubmit = async () => {
  if (loading.value) return

  loading.value = true
  try {
    if (assetType.value === 'usdt') {
      await submitUsdtRecharge()
    } else {
      await submitWinRecharge()
    }
  } catch (error) {
    console.error('充值失败:', error)
    pendingLoading = false
    // Contract.send / ETH.getAccount 内部已 showFailToast，避免 closeToast 覆盖后再弹一次
    if (typeof error === 'string') return
    const mapped = assetType.value === 'win'
      ? mapWinRechargeError(error, $t)
      : (isWalletCancelled(error) ? $t('recharge.userCancelled') : '')
    showRechargeToast(mapped || errMsg(error, $t('common.operationFailed')))
  } finally {
    loading.value = false
    stopRechargeLoading()
  }
}

defineExpose({ open })
</script>

<style lang="scss" scoped>
@use '@/style/variables.scss' as *;

.withdraw-dialog {
  height: 280px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  align-items: stretch;
}

.dialog-heading {
  margin: 0 0 16px;
  padding-bottom: 14px;
  text-align: center;
  font-size: 18px;
  font-weight: 600;
  line-height: 1.2;
  color: #fff;
  letter-spacing: 0.02em;
  background: linear-gradient(90deg, #fff 0%, $brand-primary-light 100%);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

.asset-tabs {
  width: 100%;
  display: flex;
  margin-bottom: 16px;

  :deep(.ant-radio-button-wrapper) {
    flex: 1;
    height: 38px;
    line-height: 36px;
    text-align: center;
    font-size: 14px;
    border-color: rgba(21, 151, 229, 0.25);
    background: rgba(3, 10, 17, 0.45);
    color: $text-muted;

    &::before {
      display: none;
    }

    &:first-child {
      border-radius: 10px 0 0 10px;
    }

    &:last-child {
      border-radius: 0 10px 10px 0;
    }
  }

  :deep(.ant-radio-button-wrapper-checked:not(.ant-radio-button-wrapper-disabled)) {
    background: $gradient-primary;
    border-color: transparent;
    color: $text-inverse;
    font-weight: 600;
  }
}

.dialog-main {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dialog-subtitle {
  margin: 0;
  min-height: 18px;
  font-size: 12px;
  line-height: 18px;
  color: $text-muted;
}

.dialog-subtitle--placeholder {
  visibility: hidden;
}

.dialog-payable {
  margin: 0;
  font-size: 13px;
  color: $brand-primary-light;
}

.dialog-info {
  display: flex;
  justify-content: flex-end;

  p {
    margin: 0;
    text-align: right;
    color: $text-muted;
    font-size: 12px;
  }
}

.withdraw-btn {
  width: 100%;
  height: 44px;
  margin-top: 10px;
  background: $gradient-primary;
  border: none;
  color: $text-inverse;
  border-radius: 22px;
  font-size: 15px;
  font-weight: 600;

  &:hover:not(:disabled) {
    background: linear-gradient(135deg, $brand-primary-light 0%, $brand-primary 100%);
  }

  &:disabled {
    opacity: 0.5;
  }
}
</style>

<style lang="scss">
.recharge-modal .ant-modal-content {
  border-radius: 16px;
  overflow: hidden;
}
</style>
