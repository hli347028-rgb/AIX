<template>
  <div class="transfer-page">
    <van-nav-bar
      :title="$t('transfer.title')"
      left-arrow
      :border="false"
      fixed
      @click-left="router.back()"
    />

    <main class="page-main">
      <div class="transfer-mode" role="tablist" :aria-label="$t('transfer.type')">
        <button
          type="button"
          role="tab"
          :aria-selected="mode === 'self'"
          :class="{ active: mode === 'self' }"
          @click="changeMode('self')"
        >
          {{ $t('transfer.toRewardWallet') }}
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="mode === 'user'"
          :class="{ active: mode === 'user' }"
          @click="changeMode('user')"
        >
          {{ $t('transfer.toUser') }}
        </button>
      </div>

      <section v-if="mode === 'self'" class="wallet-flow" :aria-label="$t('transfer.direction')">
        <div class="wallet-summary">
          <strong>{{ sourceWalletName }}</strong>
          <span class="wallet-balance">{{ sourceBalance }} USDT</span>
        </div>
        <van-icon class="flow-icon" name="arrow" aria-hidden="true" />
        <div class="wallet-summary wallet-summary-target">
          <strong>{{ targetWalletName }}</strong>
          <!-- 这里原本写成 mode === 'self' ? `${rewardBalance} USDT` : ''，
               但整个 <section> 的 v-if 就是 mode === 'self'，条件恒为真，
               false 分支永远取不到。 -->
          <span class="wallet-balance">{{ rewardBalance }} USDT</span>
        </div>
      </section>
      <div v-else class="reward-balance-row">
        <span>{{ $t('transfer.rewardBalance') }}</span>
        <strong>{{ rewardBalance }} USDT</strong>
      </div>

      <section class="transfer-form">
        <template v-if="mode === 'user'">
          <label class="field-label" for="transfer-recipient">{{ $t('transfer.recipientAddress') }}</label>
          <div class="input-shell">
            <van-icon name="contact-o" aria-hidden="true" />
            <input
              id="transfer-recipient"
              v-model.trim="recipient"
              type="text"
              autocomplete="off"
              spellcheck="false"
              :placeholder="$t('transfer.recipientPlaceholder')"
            />
          </div>
        </template>

        <div class="amount-heading">
          <label class="field-label" for="transfer-amount">{{ $t('transfer.amount') }}</label>
          <button type="button" class="all-btn" @click="fillAll">{{ $t('transfer.all') }}</button>
        </div>
        <div class="input-shell amount-shell">
          <input
            id="transfer-amount"
            v-model="amount"
            type="text"
            inputmode="decimal"
            placeholder="0.00"
            @input="normalizeAmount"
          />
          <span class="currency">USDT</span>
        </div>

        <button
          type="button"
          class="aix-btn transfer-submit"
          :disabled="!canSubmit || loading"
          @click="submitTransfer"
        >
          {{ loading ? $t('transfer.processing') : $t('transfer.confirm') }}
        </button>
      </section>

      <p class="security-note">
        <van-icon name="shield-o" aria-hidden="true" />
        {{ transferHint }}
      </p>

      <section class="record-section">
        <div class="section-title-wrap">
          <div class="title-bar"></div>
          <h3 class="section-title">{{ $t('transfer.records') }}</h3>
          <div v-if="mode === 'user'" class="direction-filter" :aria-label="$t('transfer.recordDirection')">
            <button
              v-for="item in directionOptions"
              :key="item.value"
              type="button"
              :class="{ active: direction === item.value }"
              @click="changeDirection(item.value)"
            >
              {{ item.label }}
            </button>
          </div>
        </div>

        <div class="table-card" :aria-busy="recordLoading">
          <div class="table-header">
            <span>{{ mode === 'self' ? $t('transfer.walletDirection') : $t('transfer.directionAndUser') }}</span>
            <span>{{ $t('transfer.amountColumn') }}</span>
            <span>{{ $t('transfer.time') }}</span>
          </div>

          <div v-if="recordLoading" class="record-loading">
            <!-- 原来硬编码 var(--accent)，那正是设计令牌里被换掉的旧电光蓝。
                 改为读取当前品牌色令牌，不再各处留一份旧色。 -->
            <van-loading type="spinner" color="var(--accent)" />
          </div>
          <div v-else-if="recordError" class="empty-state record-error">
            <p>{{ $t('transfer.fetchRecordsFailed') }}</p>
            <button type="button" class="retry-btn" @click="loadTransferRecords(recordPage)">
              {{ $t('common.retry') }}
            </button>
          </div>
          <template v-else-if="records.length > 0">
            <div class="order-list" v-for="item in records" :key="item.id">
              <div class="table-row">
                <span v-if="mode === 'self'" class="wallet-direction">{{ $t('transfer.rechargeToReward') }}</span>
                <span v-else class="counterparty-cell">
                  <strong>{{ recordDirection(item) === 'in' ? $t('transfer.in') : $t('transfer.out') }} · {{ relationshipText(item) }}</strong>
                  <small>{{ formatAddress(counterpartyAddress(item)) }}</small>
                </span>
                <span class="amount-cell" :class="recordDirection(item) === 'in' ? 'income' : mode === 'user' ? 'outcome' : ''">
                  {{ signedAmount(item) }} USDT
                </span>
                <span class="time-cell">{{ formatUnixTime(item.created_at) }}</span>
              </div>
            </div>
          </template>
          <div v-else class="empty-state">
            <p>{{ $t('transfer.noRecords') }}</p>
          </div>

          <div class="pagination-wrapper" v-if="!recordLoading && recordPageCount > 1">
            <Pagination
              v-model="recordPage"
              :page-count="recordPageCount"
              mode="simple"
              @change="loadTransferRecords"
            />
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Pagination, showFailToast, showSuccessToast, showToast } from 'vant'
import userPerson from '@/pinia/person'
import request from '@/tools/request'

type TransferMode = 'self' | 'user'
type TransferDirection = 'all' | 'in' | 'out'

interface PageResult<T> {
  page: number
  page_size: number
  total: number
  list: T[]
}

interface SelfTransferRecord {
  id: number
  asset: 'USDT'
  amount: string
  from_wallet: 'recharge'
  to_wallet: 'reward'
  created_at: number
}

interface LinealTransferRecord {
  id: number
  direction: Exclude<TransferDirection, 'all'>
  relationship: 'upline' | 'downline'
  counterparty_user_id: number
  counterparty_address: string
  asset: 'USDT'
  amount: string
  from_wallet: 'reward'
  to_wallet: 'reward'
  created_at: number
}

type TransferRecord = SelfTransferRecord | LinealTransferRecord

const RECORD_PAGE_SIZE = 10

const router = useRouter()
const { t: $t, locale } = useI18n()
const person = userPerson()

const mode = ref<TransferMode>('self')
const recipient = ref('')
const amount = ref('')
const loading = ref(false)
const records = ref<TransferRecord[]>([])
const recordPage = ref(1)
const recordPageCount = ref(1)
const recordLoading = ref(false)
const recordError = ref(false)
const direction = ref<TransferDirection>('all')
let recordRequestId = 0

const directionOptions = computed<Array<{ label: string; value: TransferDirection }>>(() => [
  { label: $t('transfer.all'), value: 'all' },
  { label: $t('transfer.in'), value: 'in' },
  { label: $t('transfer.out'), value: 'out' },
])

const rechargeBalance = computed(() => String(person.userinfo?.usdt || person.profile?.usdt_recharge || '0'))
const rewardBalance = computed(() => String((person.userinfo as any)?.reward || person.profile?.usdt_reward || '0'))
const sourceBalance = computed(() => mode.value === 'self' ? rechargeBalance.value : rewardBalance.value)
const sourceWalletName = computed(() => mode.value === 'self' ? $t('transfer.rechargeWallet') : $t('transfer.rewardWallet'))
const targetWalletName = computed(() => mode.value === 'self' ? $t('transfer.myRewardWallet') : $t('transfer.userRewardWallet'))
const transferHint = computed(() => mode.value === 'self'
  ? $t('transfer.selfHint')
  : $t('transfer.userHint')
)

const isPositiveAmount = (value: string) => /^\d+(?:\.\d+)?$/.test(value) && /[1-9]/.test(value)

const compareDecimalStrings = (left: string, right: string) => {
  const normalize = (value: string) => {
    const [integer = '0', fraction = ''] = String(value || '0').split('.')
    return {
      integer: integer.replace(/^0+(?=\d)/, '') || '0',
      fraction: fraction.replace(/0+$/, ''),
    }
  }
  const a = normalize(left)
  const b = normalize(right)
  if (a.integer.length !== b.integer.length) return a.integer.length > b.integer.length ? 1 : -1
  if (a.integer !== b.integer) return a.integer > b.integer ? 1 : -1
  const fractionLength = Math.max(a.fraction.length, b.fraction.length)
  const aFraction = a.fraction.padEnd(fractionLength, '0')
  const bFraction = b.fraction.padEnd(fractionLength, '0')
  return aFraction === bFraction ? 0 : aFraction > bFraction ? 1 : -1
}

const canSubmit = computed(() => {
  const recipientReady = mode.value === 'self' || recipient.value.length > 0
  return recipientReady && isPositiveAmount(amount.value)
})

const changeMode = (nextMode: TransferMode) => {
  if (mode.value === nextMode) return
  mode.value = nextMode
  recipient.value = ''
  amount.value = ''
  direction.value = 'all'
  loadTransferRecords(1)
}

const changeDirection = (nextDirection: TransferDirection) => {
  if (direction.value === nextDirection) return
  direction.value = nextDirection
  loadTransferRecords(1)
}

const formatAddress = (value?: string) => {
  if (!value) return '-'
  return value.length > 14 ? `${value.slice(0, 6)}...${value.slice(-4)}` : value
}

const formatUnixTime = (timestamp: number) => {
  if (!timestamp) return '-'
  const dateLocales: Record<string, string> = {
    zh: 'zh-CN',
    'zh-tw': 'zh-TW',
    en: 'en-US',
    ja: 'ja-JP',
    ko: 'ko-KR',
    vi: 'vi-VN',
  }
  return new Date(timestamp * 1000).toLocaleString(dateLocales[locale.value] || locale.value, { hour12: false })
}

const isLinealRecord = (item: TransferRecord): item is LinealTransferRecord => 'direction' in item

const recordDirection = (item: TransferRecord) => isLinealRecord(item) ? item.direction : undefined

const counterpartyAddress = (item: TransferRecord) => isLinealRecord(item) ? item.counterparty_address : undefined

const relationshipText = (item: TransferRecord) => {
  if (!isLinealRecord(item)) return '-'
  if (item.relationship === 'upline') return $t('transfer.upline')
  if (item.relationship === 'downline') return $t('transfer.downline')
  return '-'
}

const signedAmount = (item: TransferRecord) => {
  if (!isLinealRecord(item)) return item.amount
  return `${item.direction === 'in' ? '+' : '-'}${item.amount}`
}

const loadTransferRecords = async (page = 1) => {
  const parsedPage = Number(page)
  const requestedPage = Number.isInteger(parsedPage) && parsedPage > 0 ? parsedPage : 1
  const requestId = ++recordRequestId
  const requestedMode = mode.value
  const requestedDirection = direction.value
  recordPage.value = requestedPage
  recordLoading.value = true
  recordError.value = false

  try {
    const params: Record<string, string | number> = {
      page: requestedPage,
      page_size: RECORD_PAGE_SIZE,
    }
    const result: PageResult<TransferRecord> = requestedMode === 'self'
      ? await request.get<PageResult<SelfTransferRecord>>('/v1/wallet/transfer-records/self', {
          params,
          silent: true,
        })
      : await request.get<PageResult<LinealTransferRecord>>('/v1/wallet/transfer-records/lineal', {
          params: { ...params, direction: requestedDirection },
          silent: true,
        })
    if (
      requestId !== recordRequestId ||
      requestedMode !== mode.value ||
      requestedDirection !== direction.value
    ) return

    const responsePage = Number(result.page)
    const responsePageSize = Number(result.page_size)
    const responseTotal = Number(result.total)
    records.value = Array.isArray(result.list) ? result.list : []
    recordPage.value = Number.isInteger(responsePage) && responsePage > 0 ? responsePage : requestedPage
    recordPageCount.value = Math.max(1, Math.ceil(
      (Number.isFinite(responseTotal) ? responseTotal : 0) /
      (Number.isInteger(responsePageSize) && responsePageSize > 0 ? responsePageSize : RECORD_PAGE_SIZE),
    ))
  } catch (error: any) {
    if (requestId !== recordRequestId) return
    records.value = []
    recordPageCount.value = 1
    recordError.value = true
    if (error?.response?.status !== 401) {
      showFailToast(error?.response?.data?.message || error?.message || $t('transfer.fetchRecordsFailed'))
    }
  } finally {
    if (requestId === recordRequestId) recordLoading.value = false
  }
}

const normalizeAmount = () => {
  let value = amount.value.replace(/[^\d.]/g, '')
  const parts = value.split('.')
  if (parts.length > 2) value = `${parts[0]}.${parts.slice(1).join('')}`
  if (parts[1]?.length > 8) value = `${parts[0]}.${parts[1].slice(0, 8)}`
  amount.value = value
}

const fillAll = () => {
  if (!isPositiveAmount(sourceBalance.value)) {
    showToast({ message: $t('transfer.insufficientBalance', { wallet: sourceWalletName.value }), position: 'middle' })
    return
  }
  amount.value = sourceBalance.value
}

const submitTransfer = async () => {
  if (loading.value) return
  if (!isPositiveAmount(amount.value)) {
    showFailToast($t('transfer.amountMustBePositive'))
    return
  }
  if (compareDecimalStrings(amount.value, sourceBalance.value) > 0) {
    showFailToast($t('transfer.insufficientBalance', { wallet: sourceWalletName.value }))
    return
  }
  if (mode.value === 'user') {
    if (!/^0x[a-fA-F0-9]{40}$/.test(recipient.value)) {
      showFailToast($t('transfer.invalidRecipient'))
      return
    }
    if (recipient.value.toLowerCase() === String(person.address || '').toLowerCase()) {
      showFailToast($t('transfer.cannotTransferToSelf'))
      return
    }
  }

  loading.value = true
  try {
    if (mode.value === 'self') {
      await request.post('/v1/wallet/recharge-to-reward', {
        amount: amount.value,
      })
    } else {
      await request.post('/v1/wallet/transfer', {
        to_address: recipient.value,
        asset: 'USDT',
        amount: amount.value,
        pay_from: 'reward',
      })
    }
    amount.value = ''
    recipient.value = ''
    await Promise.all([
      person.getUser?.(),
      person.refreshProfile?.(),
    ])
    await loadTransferRecords(1)
    showSuccessToast($t('transfer.success'))
  } catch (error: any) {
    // request 已优先展示后端 message，此处仅兜底非 Axios 错误。
    if (!error?.response) showFailToast(error?.message || $t('transfer.failed'))
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await Promise.all([
    person.getUser?.(),
    person.refreshProfile?.(),
  ])
  await loadTransferRecords(1)
})
</script>

<style lang="scss" scoped>
/* 本页原先 @use variables.scss，用 $brand-primary-light / $text-muted /
   $border-color / $radius-md / $gradient-primary 等旧变量，
   与全站令牌体系并行两套颜色；现已全部改为 var(--*)。

   另外去掉了两处 !important（.wallet-direction / .time-cell）——
   它们是在跟同文件里自己写的 .table-row > span 抢权重，
   把选择器理顺后就不需要了。 */

/* 原先这里铺了 linear-gradient(180deg,#030a11,#071421,#020508)：
   三个硬编码 hex 拼出的页面级渐变，本站其它页面都没有，
   于是从别的页跳进来会看到底色突变。改为统一底色。 */
.transfer-page {
  min-height: 100vh;
  background: var(--ink);
}

.page-main {
  width: 100%;
  /* 不设 max-width：polish.less 把 body > #app 限死在 414px，
     这里的 520px 永远不会生效。 */
  padding: 76px 20px 40px;
  box-sizing: border-box;
}

/* ---------------------------- 模式切换 ---------------------------- */
.transfer-mode {
  width: 100%;
  height: 40px;
  margin-bottom: 24px;
  padding: 3px;
  border: 1px solid var(--hair);
  border-radius: var(--r-sm);
  background: linear-gradient(180deg, var(--ink) 0%, var(--surface-1) 100%);
  box-shadow: inset 0 2px 6px rgba(0, 0, 0, 0.07);
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  box-sizing: border-box;

  button {
    min-width: 0;
    border: 0;
    border-radius: 6px;
    background: transparent;
    color: var(--text-3);
    font-size: 13px;
    cursor: pointer;
    transition: color var(--t-fast) var(--ease);

    &.active {
      background: linear-gradient(180deg, var(--surface-3) 0%, var(--surface-2) 100%);
      color: var(--text);
      font-weight: 600;
      box-shadow: var(--shadow-1);
    }
  }
}

/* ---------------------------- 流向摘要 ----------------------------
   原先左右两个 .wallet-summary 各自带描边、圆角和半透明底色 ——
   两张小卡片夹一个箭头，卡面比"从哪到哪"这件事本身更显眼。
   现在去掉卡面，用一条发丝线框住整体，箭头居中分隔。 */
.wallet-flow {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 28px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  margin-bottom: 24px;
  padding: 14px 0;
  border-top: 1px solid var(--hair);
  border-bottom: 1px solid var(--hair);
}

.wallet-summary {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;

  strong {
    color: var(--text);
    font-size: 14px;
    font-weight: 500;
    overflow-wrap: anywhere;
  }
}

/* 目标钱包原先靠一圈 rgba(255, 255, 255, .3) 的浅蓝描边来区分，
   那是个不在调色板里的颜色。方向已由中间的箭头表达清楚，
   这里只把目标侧的文字右对齐即可。 */
.wallet-summary-target {
  text-align: right;
}

.wallet-balance {
  font-family: var(--aix-font-display);
  font-variant-numeric: tabular-nums;
  font-size: 15px;
  font-weight: 500;
  color: var(--text-2);
  overflow-wrap: anywhere;
}

.flow-icon {
  color: var(--text-3);
  font-size: 18px;
  text-align: center;
}

.reward-balance-row {
  margin-bottom: 24px;
  padding: 14px 0;
  border-top: 1px solid var(--hair);
  border-bottom: 1px solid var(--hair);
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  color: var(--text-3);
  font-size: 13px;

  strong {
    min-width: 0;
    font-family: var(--aix-font-display);
    font-variant-numeric: tabular-nums;
    font-size: 17px;
    font-weight: 500;
    color: var(--text);
    overflow-wrap: anywhere;
    text-align: right;
  }
}

/* ------------------------------ 表单 ------------------------------
   原先整块表单套在一张 rgba(8,19,30,.9) + 描边 + $shadow-md 的卡片里，
   而它本来就是页面上唯一的输入区域，不需要再用卡片把它"圈出来"。 */
.transfer-form {
  padding: 0;
}

.field-label {
  display: block;
  margin-bottom: 9px;
  color: var(--text-3);
  font-size: 12px;
}

.amount-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 20px;

  .field-label {
    margin-bottom: 9px;
  }
}

.transfer-form > .amount-heading:first-child {
  margin-top: 0;
}

.all-btn {
  padding: 0 0 9px;
  border: 0;
  background: transparent;
  color: var(--accent-bright);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
}

/* 输入框保留"凹槽"质感 —— 这是少数应该有内陷感的地方，
   它提示这里可以键入，而不是一块展示区。 */
.input-shell {
  width: 100%;
  height: 46px;
  padding: 0 13px;
  border: 1px solid var(--hair);
  border-radius: var(--r-sm);
  background: linear-gradient(180deg, var(--ink-deep) 0%, var(--surface-1) 100%);
  box-shadow: inset 0 2px 6px rgba(0, 0, 0, 0.07);
  display: flex;
  align-items: center;
  gap: 9px;
  box-sizing: border-box;
  transition: border-color var(--t-fast) var(--ease);

  &:focus-within {
    border-color: var(--accent);
  }

  .van-icon {
    flex: 0 0 auto;
    color: var(--text-3);
    font-size: 18px;
  }

  input {
    min-width: 0;
    flex: 1;
    border: 0;
    outline: 0;
    background: transparent;
    color: var(--text);
    font-size: 14px;

    &::placeholder {
      color: var(--text-3);
    }
  }
}

/* 金额是本页要输入的核心数字，用展示字体放大。 */
.amount-shell input {
  font-family: var(--aix-font-display);
  font-variant-numeric: tabular-nums;
  font-size: 20px;
  font-weight: 500;
  letter-spacing: -0.01em;
}

.currency {
  flex: 0 0 auto;
  color: var(--text-3);
  font-size: 12px;
  letter-spacing: 0.08em;
}

/* 外观（含禁用态）来自 .aix-btn 原语，这里只留本页的外边距。

   刻意不叫 .submit-btn：polish.less 第 9e 节的过渡补丁用
   `.submit-btn { ... !important }` 接管旧页面按钮，而 pledge.vue 和
   exchange.vue 还在依赖它，所以那段补丁暂时不能删。沿用同名类会让本页
   按钮被 !important 抢走、绕过新原语；换个类名即可干净退出补丁范围。 */
.transfer-submit {
  margin-top: 24px;
}

.security-note {
  margin-top: 16px;
  color: var(--text-3);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  gap: 6px;
  font-size: 11px;
  line-height: 1.6;

  .van-icon {
    margin-top: 2px;
    color: var(--text-3);
    font-size: 14px;
  }
}

/* ------------------------------ 记录 ------------------------------ */
.record-section {
  margin-top: 32px;
}

.direction-filter {
  height: 26px;
  margin-left: auto;
  display: flex;
  gap: 4px;
  box-sizing: border-box;

  button {
    min-width: 42px;
    padding: 0 9px;
    border: 1px solid transparent;
    border-radius: var(--r-pill);
    background: transparent;
    color: var(--text-3);
    font-size: 11px;
    cursor: pointer;
    transition: color var(--t-fast) var(--ease), border-color var(--t-fast) var(--ease);

    /* 原先选中态是一块旧品牌蓝的半透明填充，调色板里已经没有那个蓝了。 */
    &.active {
      border-color: var(--hair-2);
      color: var(--text);
    }
  }
}

/* 原先是一张 min-height:300px、带描边圆角和 backdrop-filter: blur(10px)
   的卡片 —— 它背后是纯色底，模糊滤镜没有任何可模糊的东西，
   纯粹在耗 GPU。改为与其它三页一致的发丝线账目表。 */
.table-card {
  padding: 0;

  .table-header,
  .table-row {
    display: grid;
    grid-template-columns: minmax(0, 1.35fr) minmax(82px, 0.85fr) minmax(0, 0.9fr);
    gap: 10px;
    align-items: baseline;
  }

  /* 表头原先是一条 #030a11 的实色带，比表体还深，像一根横杠压在上面。
     改为发丝线 + 小型大写标签。 */
  .table-header {
    padding: 12px 0;
    border-bottom: 1px solid var(--hair);

    span {
      min-width: 0;
      color: var(--text-3);
      font-size: 10px;
      font-weight: 600;
      letter-spacing: 0.1em;
      text-transform: uppercase;
    }
  }

  .order-list {
    .table-row {
      padding: 14px 0;
      border-bottom: 1px solid var(--hair);
      box-sizing: border-box;

      > span {
        min-width: 0;
        color: var(--text-2);
        font-size: 12px;
      }
    }

    &:last-of-type .table-row {
      border-bottom: 0;
    }
  }

  /* 金额与时间列右对齐：原先三列全部居中，数字居中最难扫读 ——
     小数点对不齐，视线要在每行左右找位置。 */
  .table-header span:nth-child(2),
  .table-header span:nth-child(3),
  .amount-cell,
  .time-cell {
    text-align: right;
  }

  .counterparty-cell {
    display: flex;
    flex-direction: column;
    gap: 3px;

    strong {
      color: var(--text);
      font-size: 12px;
      font-weight: 500;
    }

    small {
      max-width: 100%;
      color: var(--text-3);
      font-size: 10px;
      font-variant-numeric: tabular-nums;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .wallet-direction {
    color: var(--text-2);
  }

  .amount-cell {
    font-family: var(--aix-font-display);
    font-variant-numeric: tabular-nums;
    font-size: 14px;
    font-weight: 500;
    color: var(--text);
    overflow-wrap: anywhere;

    /* 原先用 #54d6a0 / #ff8d8d 两个就地写死的颜色，
       而令牌里本来就有 --up / --down 这对涨跌语义色。 */
    &.income {
      color: var(--up);
    }

    &.outcome {
      color: var(--down);
    }
  }

  .time-cell {
    color: var(--text-3);
    font-size: 10px;
    font-variant-numeric: tabular-nums;
    line-height: 1.4;
    overflow-wrap: anywhere;
  }

  /* 原先加载态与空状态都硬撑 250px 高、外层卡片再 min-height:300px，
     于是"暂无记录"是一行字浮在一个 300px 的空盒子中央。 */
  .record-loading,
  .empty-state {
    padding: 60px 0;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .empty-state p {
    margin: 0;
    color: var(--text-3);
    font-size: 12px;
  }

  .record-error {
    flex-direction: column;
    gap: 14px;
  }

  .retry-btn {
    min-width: 76px;
    height: 30px;
    padding: 0 16px;
    border: 1px solid var(--hair-2);
    border-radius: var(--r-sm);
    background: transparent;
    color: var(--text-2);
    font-size: 12px;
    cursor: pointer;
    transition: color var(--t-fast) var(--ease), border-color var(--t-fast) var(--ease);

    &:hover {
      color: var(--text);
      border-color: var(--hair-3);
    }
  }

  .pagination-wrapper {
    padding: 18px 0 5px;
    display: flex;
    justify-content: center;
  }
}

@media (max-width: 350px) {
  .page-main {
    padding-right: 14px;
    padding-left: 14px;
  }

  .direction-filter button {
    min-width: 36px;
    padding: 0 6px;
  }

  .table-card {
    .table-header,
    .table-row {
      grid-template-columns: minmax(0, 1.2fr) minmax(72px, 0.8fr) minmax(0, 0.9fr);
    }
  }
}
</style>
