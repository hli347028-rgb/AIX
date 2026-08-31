<template>
  <div class="node-page">
    <Header />

    <div class="content">
      <nav class="home-return-bar" :aria-label="$t('common.pageNavigation')">
        <RouterLink to="/" class="home-return-link" :aria-label="$t('common.backHome')">
          <span aria-hidden="true">‹</span>
          {{ $t('common.backHome') }}
        </RouterLink>
      </nav>

      <div class="page-header">
        <div class="mode-tabs" role="radiogroup">
          <label class="mode-option" :class="{ active: tabMode === 'mint', disabled: submitting }">
            <input
              type="radio"
              name="subscribe-tab"
              value="mint"
              :checked="tabMode === 'mint'"
              :disabled="submitting"
              @change="switchTab('mint')"
            />
            <span class="radio-dot" aria-hidden="true"></span>
            <strong>{{ $t('node.reportOrder') }}</strong>
          </label>
          <label class="mode-option" :class="{ active: tabMode === 'reward', disabled: submitting }">
            <input
              type="radio"
              name="subscribe-tab"
              value="reward"
              :checked="tabMode === 'reward'"
              :disabled="submitting"
              @change="switchTab('reward')"
            />
            <span class="radio-dot" aria-hidden="true"></span>
            <strong>{{ $t('node.reinvest') }}</strong>
          </label>
        </div>

        <div v-if="tabMode === 'mint'" class="funding-switch">
          <span class="funding-label">{{ $t('node.fundingSource') }}</span>
          <div class="funding-options" role="radiogroup">
            <button
              type="button"
              class="funding-option"
              :class="{ active: mintFunding === 'recharge' }"
              :disabled="submitting"
              @click="switchMintFunding('recharge')"
            >
              {{ $t('node.rechargeWallet') }}
            </button>
            <button
              type="button"
              class="funding-option"
              :class="{ active: mintFunding === 'win' }"
              :disabled="submitting"
              @click="switchMintFunding('win')"
            >
              {{ $t('node.winPay') }}
            </button>
          </div>
        </div>

        <div class="balance-card">
          <div class="balance-summary">
            <span>{{ balanceLabel }}</span>
            <strong>{{ displayAmount(accountBalance) }} <small>{{ balanceUnit }}</small></strong>
          </div>
          <div class="custom-amount">
            <div class="custom-heading">
              <label class="custom-hint" for="custom-amount-input">
                {{ $t('node.customAmountHint', { amount: minAmountText, unit: amountUnit }) }}
              </label>
              <button type="button" class="all-btn" :disabled="submitting" @click="fillAll">
                {{ $t('node.all') }}
              </button>
            </div>
            <div class="custom-row">
              <input
                id="custom-amount-input"
                v-model="customAmount"
                class="custom-input"
                type="number"
                :min="minAmountText"
                step="any"
                :placeholder="$t('node.minPlaceholder', { amount: minAmountText })"
              />
              <button class="subscribe-btn custom-btn" :disabled="submitting" @click="handleCustomSubscribe">
                {{ actionText }}
              </button>
            </div>
          </div>
        </div>
        <p v-if="activeMode === 'win' && winPrice > 0" class="mode-tip win-cost-tip">
          {{ $t('node.winPayHint', { price: winPrice, cost: minAmountText }) }}
        </p>
        <p class="mode-tip">{{ modeTip }}</p>
      </div>

      <!-- <div class="node-tiers">
        <div
          v-for="tier in nodeTiers"
          :key="tier.price"
          class="tier-card"
          :class="{ active: selectedTier === tier.price }"
          @click="selectedTier = tier.price"
        >
          <div class="tier-header">
            <div class="tier-price">{{ tierAmountText(tier.price) }}</div>
            <div class="tier-unit">{{ amountUnit }}</div>
          </div>
          <button class="subscribe-btn" :disabled="submitting" @click.stop="handleSubscribe(String(tier.price))">
            {{ actionText }}
          </button>
        </div>
      </div> -->

      <div class="record-section">
        <div class="section-title-wrap">
          <div class="title-bar"></div>
          <h3 class="section-title">{{ $t('node.orderList') }}</h3>
        </div>

        <div class="table-card">
          <div class="table-header">
            <span>{{ $t('node.amount') }}</span>
            <span>{{ $t('node.status') }}</span>
            <span>{{ $t('node.fundingSource') }}</span>
            <span>{{ $t('node.time') }}</span>
          </div>
          <div class="order-list" v-for="(item, index) in orderList" :key="item.id || index">
            <div class="table-row">
              <span>
                {{ item.total_amount ?? item.amount }} USDT
                <small v-if="item.from_win" class="win-deduct">-{{ displayAmount(item.from_win) }} WIN</small>
              </span>
              <span>{{ orderStatusText(item.status) }}</span>
              <span>{{ fundingSourceText(item.fund_source || item.product_name) }}</span>
              <span>{{ item.created_at ?? item.createdAt }}</span>
            </div>
          </div>
          <div class="empty-state" v-if="orderList.length === 0">
            <p>{{ $t('common.noData') }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Header from '@/components/Header.vue'
import request from '@/tools/request'
import { showSuccessToast, showFailToast, showLoadingToast, closeToast, showConfirmDialog } from 'vant'
import userPerson from '@/pinia/person'
import { errMsg, listAixOrders, subscribeAix } from '@/api/aix'
import { compareDecimals, displayDecimal, divDecimal, isPositiveDecimal, mulDecimal } from '@/tools/decimal'

const { t: $t } = useI18n()
const person = userPerson()

type SubscribeMode = 'recharge' | 'reward' | 'win'
type TabMode = 'mint' | 'reward'
type MintFunding = 'recharge' | 'win'

const tabMode = ref<TabMode>('mint')
const mintFunding = ref<MintFunding>('recharge')
const activeMode = computed<SubscribeMode>(() =>
  tabMode.value === 'reward' ? 'reward' : mintFunding.value
)
const winPrice = computed(() => Number(person.profile?.win_price || 0))
const accountBalance = computed(() => {
  const profile = person.profile
  if (activeMode.value === 'win') return profile?.win_recharge_balance || '0.00'
  if (activeMode.value === 'recharge') return profile?.usdt_recharge || '0.00'
  return profile?.usdt_reward || '0.00'
})
const balanceLabel = computed(() => {
  if (activeMode.value === 'win') return $t('node.winWalletBalance')
  if (activeMode.value === 'recharge') return $t('node.rechargeWalletBalance')
  return $t('node.rewardWalletBalance')
})
const amountUnit = computed(() => activeMode.value === 'win' ? 'WIN' : 'USDT')
const balanceUnit = amountUnit
const modeTip = computed(() => activeMode.value === 'win' ? $t('node.winReferralTip') : $t('node.referralRewardTip'))
const minAmountText = computed(() =>
  activeMode.value === 'win' ? calcWinCost(minSubscribe.value) : String(minSubscribe.value)
)

function tierAmountText(price: number) {
  return activeMode.value === 'win' ? calcWinCost(price) : String(price)
}

function calcWinCost(usdtAmount: number | string) {
  const needWin = calcNeedWin(String(usdtAmount))
  return needWin ? displayAmount(needWin) : '-'
}

function displayAmount(value: unknown) {
  return displayDecimal(value)
}

function calcNeedWin(usdtAmount: string) {
  if (!winPrice.value || winPrice.value <= 0 || !isPositiveDecimal(usdtAmount)) return null
  return divDecimal(usdtAmount, String(winPrice.value))
}

const actionText = computed(() => {
  if (activeMode.value === 'reward') return $t('node.reinvestNow')
  if (activeMode.value === 'win') return $t('node.winPayNow')
  return $t('node.reportNow')
})

interface NodeTier {
  price: number
}

const selectedTier = ref<number | null>(null)
const orderList = ref<any[]>([])
const nodeTiers = ref<NodeTier[]>([])
const minSubscribe = ref(100)
const customAmount = ref('')
const submitting = ref(false)

const switchTab = (tab: TabMode) => {
  if (submitting.value) return
  tabMode.value = tab
  selectedTier.value = null
  customAmount.value = ''
}

const switchMintFunding = (funding: MintFunding) => {
  if (submitting.value) return
  mintFunding.value = funding
  selectedTier.value = null
  customAmount.value = ''
}

const getSubscribeTiers = async () => {
  try {
    const res: any = await request.get('app_server/subscribe_tiers')
    minSubscribe.value = Math.max(100, Number(res?.min_subscribe_amount || 100))
    nodeTiers.value = (res?.tiers || []).map((value: string | number) => ({
      price: Number(value),
    })).filter((tier: NodeTier) => tier.price > 0)
  } catch {
    nodeTiers.value = [100, 500, 1000, 3000].map((price) => ({ price }))
    minSubscribe.value = 100
  }
}

const getOrderList = async () => {
  try {
    const res: any = await listAixOrders()
    orderList.value = res.orders || []
  } catch {
    orderList.value = []
  }
}

const fundingSourceText = (source: string) => {
  const map: Record<string, string> = {
    reward: $t('node.rewardSource'),
    win: $t('node.winSource'),
    win_a: $t('node.winASource'),
    recharge: $t('node.rechargeSource'),
    'recharge+win': $t('node.mixSourceRechargeWin'),
    'recharge+win_a': $t('node.mixSourceRechargeWinA'),
    'win+win_a': $t('node.mixSourceWinWinA'),
    'win+recharge': $t('node.mixSourceWinRecharge'),
    'win_a+recharge': $t('node.mixSourceWinARecharge'),
    'win_a+win': $t('node.mixSourceWinAWin'),
  }
  return map[source] || $t('node.unknownSource')
}

/* 状态文案映射。
   修两个 bug：
   1. 漏了 'completed' —— 后端确实会返回这个值（截图里第三行就直接显示了
      英文 "completed"）。而 node.statusCompleted 这个键在 7 种语言里
      全都已经存在，只是这里忘了接上。
   2. 兜底 `return s` 会把任何未知的原始状态串直接漏到界面上。
      未知状态应该显示为中性占位，绝不该把后端的内部值暴露给用户 ——
      前者是小瑕疵，后者是会让用户看到 "completed"/"pending_review"
      这类开发术语的信息泄漏。 */
const orderStatusText = (status: string | number) => {
  const s = String(status ?? '').trim().toLowerCase()
  if (s === '1' || s === 'active') return $t('node.statusActive')
  if (s === '2' || s === 'exited') return $t('node.statusExited')
  if (s === '3' || s === 'completed') return $t('node.statusCompleted')
  return '—'
}

const handleSubscribe = async (usdtAmount: string) => {
  if (submitting.value) return

  if (!isPositiveDecimal(usdtAmount) || compareDecimals(usdtAmount, String(minSubscribe.value)) < 0) {
    showFailToast($t('node.minSubscribeAmount', { amount: minAmountText.value, unit: amountUnit.value }))
    return
  }
  const mode = activeMode.value
  let needWin: string | null = null
  if (mode === 'win') {
    if (!winPrice.value || winPrice.value <= 0) {
      showFailToast($t('node.winPriceMissing'))
      return
    }
    needWin = calcNeedWin(usdtAmount)
    if (!needWin || !isPositiveDecimal(needWin)) {
      showFailToast($t('node.winPriceMissing'))
      return
    }
    if (compareDecimals(needWin, accountBalance.value) > 0) {
      showFailToast($t('node.insufficientWin'))
      return
    }
  } else if (compareDecimals(usdtAmount, accountBalance.value) > 0) {
    showFailToast($t('common.insufficientBalance'))
    return
  }

  const confirmMessage = mode === 'reward'
    ? $t('node.confirmReinvest', { amount: usdtAmount })
    : mode === 'win'
      ? $t('node.confirmWinPay', { cost: displayAmount(needWin) })
      : $t('node.confirmReport', { amount: usdtAmount })
  try {
    await showConfirmDialog({
      title: $t('common.prompt'),
      message: confirmMessage,
      confirmButtonText: $t('common.agree'),
      cancelButtonText: $t('common.reject'),
    })
  } catch {
    return
  }

  selectedTier.value = Number(usdtAmount)
  submitting.value = true
  showLoadingToast({ message: $t('common.loading'), duration: 0 })
  try {
    await subscribeAix(usdtAmount, mode)
    closeToast()
    const okMsg = mode === 'reward'
      ? $t('node.reinvestSuccess')
      : mode === 'win'
        ? $t('node.winPaySuccess')
        : $t('node.reportSuccess')
    showSuccessToast(okMsg)
    customAmount.value = ''
    selectedTier.value = null
    await Promise.all([person.refreshProfile(), getOrderList()])
  } catch (error: any) {
    closeToast()
    const code = error?.response?.data?.reason || error?.response?.data?.code
    const messageKey: Record<string, string> = {
      MIN_SUBSCRIBE_LIMIT: 'node.minSubscribeAmount',
      WIN_PRICE_NOT_CONFIGURED: 'node.winPriceMissing',
      INSUFFICIENT_WIN: 'node.insufficientWin',
      INSUFFICIENT_BALANCE: 'common.insufficientBalance',
      INVALID_AMOUNT: 'node.enterSubscribeAmount',
    }
    const failMsg = mode === 'reward'
      ? $t('node.reinvestFailed')
      : mode === 'win'
        ? $t('node.winPayFailed')
        : $t('node.reportFailed')
    const mapped = messageKey[code]
      ? $t(messageKey[code], { amount: minAmountText.value, unit: amountUnit.value })
      : ''
    showFailToast(mapped || errMsg(error, failMsg))
  } finally {
    submitting.value = false
  }
}

const fillAll = () => {
  if (submitting.value) return
  if (activeMode.value === 'win') {
    if (!winPrice.value || winPrice.value <= 0) {
      showFailToast($t('node.winPriceMissing'))
      return
    }
    if (!isPositiveDecimal(accountBalance.value)) {
      showFailToast($t('node.insufficientWin'))
      return
    }
    customAmount.value = displayDecimal(accountBalance.value)
    return
  }
  if (!isPositiveDecimal(accountBalance.value)) {
    showFailToast($t('common.insufficientBalance'))
    return
  }
  customAmount.value = displayDecimal(accountBalance.value)
}

const handleCustomSubscribe = () => {
  const input = String(customAmount.value ?? '').trim()
  if (!isPositiveDecimal(input)) {
    showFailToast($t('node.enterSubscribeAmount'))
    return
  }
  if (activeMode.value !== 'win') {
    handleSubscribe(input)
    return
  }
  if (!winPrice.value || winPrice.value <= 0) {
    showFailToast($t('node.winPriceMissing'))
    return
  }
  handleSubscribe(mulDecimal(input, String(winPrice.value)))
}

onMounted(async () => {
  await Promise.all([
    getSubscribeTiers(),
    getOrderList(),
    person.refreshProfile()
  ])
})
</script>

<style lang="scss" scoped>
@use '@/style/variables.scss' as *;

.node-page {
  --accent: #0052ff;
  --accent-bright: #0052ff;
  --accent-deep: #0648df;
  --accent-dim: rgba(0, 82, 255, 0.1);
  min-height: 100vh;
  /* 纯白。原本是 ink-deep → surface-2 的纵向渐变（深色版的做法）。
     Base 的页面底色是干净的白，全站零渐变。 */
  background: var(--ink);
}

.content {
  padding: 80px 20px 40px;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 20px;

  .mode-tabs {
    width: 100%;
    height: 40px;
    box-sizing: border-box;
    margin: 0 auto 18px;
    padding: 3px;
  border: 1px solid var(--hair);
  border-radius: var(--r-pill);
  /* 浅灰槽。原本是"深色渐变 + inset 深阴影"做的内凹槽 ——
     那套拟物凹陷在白底上不成立（深阴影会变成一道脏灰），
     Base 也没有任何 inset 拟���效果。改成纯浅灰底 + 全圆外框。 */
  background: var(--surface-2);
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .funding-switch {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
    padding: 10px 12px;
    border: 1px solid var(--hair);
    border-radius: var(--r-md);
    background: var(--surface-1);
  }

  .funding-label {
    flex: 0 0 auto;
    color: var(--text-3);
    font-size: var(--fs-sm);
  }

  .funding-options {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    flex: 1 1 auto;
    min-width: 0;
    max-width: 360px;
    padding: 3px;
    border-radius: var(--r-pill);
    background: var(--surface-2);
  }

  .funding-option {
    min-width: 0;
    height: 32px;
    padding: 0 10px;
    border: 0;
    border-radius: var(--r-pill);
    background: transparent;
    color: var(--text-3);
    font-size: var(--fs-sm);
    white-space: nowrap;
    cursor: pointer;
    transition: background-color var(--t-fast) var(--ease), color var(--t-fast) var(--ease);

    &.active {
      background: var(--accent);
      color: var(--surface-1);
      font-weight: 600;
    }

    &:disabled {
      cursor: wait;
      opacity: 0.65;
    }
  }

  .mode-option {
    position: relative;
    display: flex;
    min-width: 0;
    align-items: center;
    justify-content: center;
    height: 100%;
    box-sizing: border-box;

    padding: 0 4px;
    background: transparent;
    border: 0;
    border-radius: var(--r-pill);
    color: var(--text-3);
    cursor: pointer;
    transition:
      background-color var(--t-fast) var(--ease),
      color var(--t-fast) var(--ease);

    input {
      position: absolute;
      width: 1px;
      height: 1px;
      opacity: 0;
      pointer-events: none;
    }

    strong {
      min-width: 0;
      font-size: 13px;
      font-weight: 400;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .radio-dot {
      display: none;
    }

    /* 选中态：白��� + 蓝字 + 蓝色描边。
       ���动的三个原因：
       1. 原本是"凹槽 + 抬起滑块"（180° 灰渐变 + 内高光 + 45% 黑投影）——
          这套拟物隐喻依赖深色底才成立，白底上四个标签几乎无差别，
          实测选中的「铸造」和未选中对比度差异微乎其微。
       2. 那句 `inset 0 1px 0 var(--gloss)` 现在是**无效 CSS**：
          我把 --gloss 改成了 none，而 `inset 0 1px 0 none` 不合法，
          浏览器会把整条 box-shadow 声明一起丢弃。这是改令牌值时
          容易漏掉的连带破坏 —— 令牌被当作"值的片段"拼进复合属性时，
          换值可能直接让整条属性失效。
       3. 用蓝**字**而不是蓝**底**，是为了保住页面主次：
          下方的认购 CTA 是全页唯一的实心蓝块，若标签也是实心蓝，
          两块蓝会互相争夺注意力。蓝字足以标明"已选"，
          又不与唯一主按钮抢位。
       #0000FF 对白 8.6:1，远超 AA。 */
    &.active {
      background: var(--surface-1);
      border: 1.5px solid var(--accent);
      color: var(--accent);

      strong {
        font-weight: 600;
        color: var(--accent);
      }
    }

    &.disabled {
      cursor: wait;
      opacity: 0.65;
    }
  }

  /* 可用余额。这是实时数据，所以数值用蓝 —— 符合用色铁律第 1 条。
     原本是 $brand-primary-light（初版配色遗留的 SCSS 变量）。 */
  .balance-card {
    display: block;
    margin-top: 10px;
    padding: 14px;
    background: var(--surface-1);
    border: 1px solid var(--hair);
    border-radius: var(--r-md);

    .balance-summary {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 14px;

      > span {
        color: var(--text-3);
        font-size: var(--fs-micro);
        letter-spacing: var(--ls-caps);
        text-transform: uppercase;
      }
    }

    strong {
      font-family: var(--aix-font-display);
      color: var(--accent-bright);
      font-size: var(--fs-lead);
      font-weight: 600;
      font-variant-numeric: tabular-nums;
      letter-spacing: var(--ls-tight);

      small {
        margin-left: 3px;
        font-size: var(--fs-micro);
        font-weight: 500;
        color: var(--text-3);
      }
    }
  }

  /* 模式说明文字。
     原本是 color: #fff + font-size: 10px —— 层级是反的：用了全站最亮的
     颜色（纯白，等同主标题）却给了最小的字号。次要说明文字应该是
     "字号小、颜色也退"，而不是"小而刺眼"。10px 正文也偏小，提到 --fs-sm。 */
  .mode-tip {
    margin: 8px 2px 0;
    color: var(--text-3);
    font-size: var(--fs-sm);
    line-height: 1.55;
  }

  /* 花费提示是需要用户核对的数字，比普通说明重要一档 */
  .win-cost-tip {
    color: var(--text-2);
    font-variant-numeric: tabular-nums;
  }
}

.custom-amount {
  margin-bottom: 0;

  .custom-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 8px;
  }

  .custom-hint {
    display: block;
    font-size: var(--fs-sm);
    color: var(--text-3);
    margin: 0;
  }

  .all-btn {
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--accent);
    font-size: var(--fs-sm);
    cursor: pointer;
  }

  .custom-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .custom-input {
    flex: 1 1 auto;
    min-width: 0;
    box-sizing: border-box;
    height: 48px;
    padding: 0 14px;
    border: 1px solid var(--hair-2);
    border-radius: var(--r-md);
    background: var(--ink);
    color: var(--text);
    font-family: var(--aix-font-display);
    font-size: var(--fs-body);
    font-variant-numeric: tabular-nums;
    outline: none;
    transition:
      border-color var(--t-fast) var(--ease),
      box-shadow var(--t-fast) var(--ease);

    &::placeholder {
      font-family: var(--aix-font-sans);
      color: var(--text-3);
    }

    &:focus {
      border-color: var(--accent);
      box-shadow: 0 0 0 3px var(--accent-dim);
    }
  }

  .custom-btn {
    flex: 0 0 auto;
    width: auto;
    min-width: 104px;
    height: 48px;
  }
}

.node-tiers {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 18px;
}

.tier-card {
  padding: 14px;
  border: 1px solid var(--hair);
  border-radius: var(--r-md);
  background: var(--surface-1);
  cursor: pointer;

  &.active {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px var(--accent-dim);
  }

  .tier-header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 12px;
  }

  .tier-price {
    color: var(--text);
    font-family: var(--aix-font-display);
    font-size: var(--fs-lead);
    font-variant-numeric: tabular-nums;
  }

  .tier-unit {
    color: var(--text-3);
    font-size: var(--fs-micro);
  }
}

.subscribe-btn {
  width: 100%;
  min-height: 40px;
  padding: 0 14px;
  border: 1px solid var(--accent);
  border-radius: var(--r-pill);
  background: var(--accent);
  color: var(--surface-1);
  font-size: var(--fs-sm);
  font-weight: 600;
  cursor: pointer;

  &:disabled {
    opacity: 0.5;
    cursor: wait;
  }
}

/* .section-title-wrap 样式已统一到 polish.less */
.record-section {
  margin-top: 20px;
}

.table-card {
  margin-top: 10px;
  /* min-height: 300px 去掉。
     记录只有 3 条时，表格照样撑到 300px，最后一行下面空一大片 ——
     和 wallet.vue 的 records-panel 是同一个毛病（那边是写死 height）。
     空状态的高度由 .empty-state 自己负责，不需要整张卡陪着撑高。 */
  overflow: hidden;
  border: 1px solid var(--hair);
  border-radius: var(--r-lg);
  background: var(--surface-1);
  padding: 0 0 4px;

  .table-header {
    display: flex;
    align-items: center;
    background: var(--ink-deep);
    padding: 10px 0;
    border-bottom: 1px solid var(--hair);

    span {
      flex: 1;
      text-align: center;
      font-size: var(--fs-micro);
      letter-spacing: var(--ls-caps);
      text-transform: uppercase;
      color: var(--text-3);
    }
  }

  .order-list {
    .table-row {
      display: flex;
      align-items: center;
      padding: 12px 8px;
      border-bottom: 1px solid var(--hair);
      transition: background-color var(--t-fast) var(--ease);

      &:last-child {
        border-bottom: none;
      }

      &:hover {
        background: var(--surface-2);
      }

      span {
        flex: 1;
        min-width: 0;
        text-align: center;
        font-size: var(--fs-sm);
        color: var(--text-2);

        .win-deduct {
          display: block;
          margin-top: 2px;
          font-size: 11px;
          color: var(--text-3);
        }
      }

      /* 金额列：给它更多空间并禁止换行。
         原本四列均分 flex:1，"1000.0000 USDT" 在 414px 宽下必然折成两行
         （截图里三行金额全是两行），把整张表撑得很松散。
         这一列信息量最大，理应分到更宽的份额。 */
      span:first-child {
        flex: 1.5;
        white-space: nowrap;
        font-family: var(--aix-font-display);
        font-variant-numeric: tabular-nums;
        letter-spacing: var(--ls-tight);
        color: var(--text);
      }
    }
  }

  /* 空状态。高度从写死的 250px 改为 min-height 160px：
     卡片的 min-height 已去掉，这里就成了空列表时唯一的高���来源，
     不需要占那么高。 */
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 160px;

    p {
      margin: 0;
      font-size: var(--fs-sm);
      color: var(--text-3);
    }
  }

  /* .pagination-wrapper 已删除：模板里没有分页器，是死样式。 */
}

/* 页面唯一主按钮 */
.primary-cta {
  width: 100%;
  min-height: 48px;
  margin-top: 4px;
  font-size: var(--fs-body);
  font-variant-numeric: tabular-nums;

  /* 禁用态只保留 cursor，不再写 opacity: .45。
     那条 opacity 会把整颗按钮（连文字一起）压到 45% 透明，
     正好抵消 polish.less 里给 .aix-cta:disabled 定的可读配色 —— 
     实测这颗按钮的文字只剩 2.1:1。

     而这颗按钮的标签是**操作指引**（未填金额时显示"请输入认购金额"），
     用户必须读懂才知道下一步做什么，不能压到看不清。
     "不可点"由 polish.less 的灰底 + cursor + 无 hover 反馈传达。 */
  &:disabled {
    cursor: not-allowed;
  }
}

/* .action-section 与 @keyframes gradient-move 已删除 —— 两块都��死代码：
   模板里从来没有 .action-section 这个元素，也没有任何规则引用
   gradient-move。它��还各自藏着问题（#04121A 调色板外色、
   transition: all 0.3s、白色外发光 box-shadow），
   而其中那句"全页唯一的主操作"注释指向的 DOM 早就不存在了 ——
   过期注释比没有注释更误导人。 */
</style>

