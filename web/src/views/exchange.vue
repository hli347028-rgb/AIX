<template>
  <div class="exchange-page">
    <van-nav-bar
      :title="$t('exchange.title')"
      left-arrow
      :border="false"
      fixed
      @click-left="router.back()"
    />

    <main class="page-main">
      <section class="balance-card">
        <div>
          <span>{{ $t('exchange.availableAix') }}</span>
          <strong>{{ displayAixBalance(aixBalance) }}</strong>
        </div>
        <van-icon name="arrow" />
        <div class="target-balance">
          <span>{{ $t('exchange.currentUsdtWithdrawable') }}</span>
          <strong>{{ displayAmount(usdtWithdrawable) }}</strong>
        </div>
      </section>

      <section class="exchange-form">
        <div class="amount-heading">
          <label for="exchange-amount">{{ $t('exchange.exchangeAmount') }}</label>
          <button type="button" @click="fillAll">{{ $t('exchange.all') }}</button>
        </div>
        <div class="input-shell" :class="{ invalid: amountError }">
          <input
            id="exchange-amount"
            v-model="amount"
            inputmode="decimal"
            type="text"
            autocomplete="off"
            placeholder="0.00"
            @input="normalizeAmount"
          />
          <span>AIX</span>
        </div>
        <p v-if="amountError" class="error-text">{{ amountError }}</p>

        <div class="rate-row">
          <span>{{ $t('exchange.currentRate') }}</span>
          <strong v-if="unitRate">1 AIX = {{ displayAmount(unitRate.usdtNet) }} USDT</strong>
          <strong v-else>{{ $t('exchange.priceUnavailable') }}</strong>
        </div>
        <div v-if="hasRate" class="rate-row rate-sub">
          <span>{{ $t('exchange.aixPrice') }}</span>
          <strong>{{ displayPrice(aixPrice) }} USDT</strong>
        </div>
        <div class="rate-row">
          <span>{{ $t('exchange.feeRate') }}</span>
          <strong>{{ feeRateText }}</strong>
        </div>
        <p v-if="preview" class="estimate-box">
          {{ $t('exchange.estimate', {
            gross: displayAmount(preview.usdtGross),
            net: displayAmount(preview.usdtNet),
            fee: displayAmount(preview.fee),
          }) }}
        </p>
        <p class="rate-note"><van-icon name="info-o" /> {{ $t('exchange.rateHint') }}</p>

        <button
          type="button"
          class="submit-btn"
          :disabled="!canSubmit || submitting"
          @click="submitExchange"
        >
          {{ submitting ? $t('exchange.processing') : $t('exchange.confirm') }}
        </button>
      </section>

      <section class="record-section">
        <h2>{{ $t('exchange.records') }}</h2>
        <div class="record-card" :aria-busy="recordLoading">
          <div v-if="recordLoading" class="state-box"><van-loading color="#8A9096" /></div>
          <van-empty v-else-if="records.length === 0" :description="$t('exchange.noRecords')" :image="emptyImage" />
          <div v-else class="record-list">
            <article v-for="item in records" :key="item.id">
              <div class="record-assets">
                <strong>{{ displayAmount(item.from_amount) }} AIX</strong>
                <van-icon name="arrow" />
                <strong class="usdt-value">{{ displayAmount(item.to_amount) }} {{ recordToAsset(item.to_asset) }}</strong>
              </div>
              <div class="record-meta">
                <span>{{ formatTime(item.created_at) }}</span>
                <span>{{ $t('exchange.rateShort') }} 1 AIX = {{ recordUnitRate(item) }} {{ recordToAsset(item.to_asset) }}</span>
              </div>
              <div v-if="item.fee_amount" class="record-fee">
                {{ $t('exchange.feeDeducted', { fee: displayAmount(item.fee_amount) }) }}
              </div>
            </article>
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
import { showFailToast, showSuccessToast } from 'vant'
import userPerson from '@/pinia/person'
import { exchangeAixToWin, getAixWinExchangeRecords } from '@/api/aix'
import type { AixExchangeRecord } from '@/api/aix'
import emptyImage from '@/assets/images/custom-empty-image.png'
import {
  calcAixToUsdtPreview,
  calcUnitAixToUsdtRate,
  compareDecimals,
  displayDecimal,
  displayAixPrice,
  divDecimal,
  formatFeeRate,
  isPositiveDecimal,
} from '@/tools/decimal'

const router = useRouter()
const { t: $t } = useI18n()
const person = userPerson()
const amount = ref('')
const submitting = ref(false)
const recordLoading = ref(false)
const records = ref<AixExchangeRecord[]>([])

const aixBalance = computed(() => String(person.profile?.aix_balance || person.userinfo?.amountGet || '0'))
const usdtWithdrawable = computed(() => String(person.profile?.usdt_withdrawable || '0'))
const aixPrice = computed(() => String(person.profile?.aix_price || '0'))
const exchangeFeeRate = computed(() => Number(person.profile?.exchange_fee_rate ?? 0.05))
const feeRateText = computed(() => formatFeeRate(exchangeFeeRate.value))
const hasRate = computed(() => isPositiveDecimal(aixPrice.value))
const unitRate = computed(() => (
  hasRate.value ? calcUnitAixToUsdtRate(aixPrice.value, exchangeFeeRate.value) : null
))
const preview = computed(() => {
  if (!amount.value || !hasRate.value) return null
  return calcAixToUsdtPreview(amount.value, aixPrice.value, exchangeFeeRate.value)
})
const amountError = computed(() => {
  if (!amount.value) return ''
  if (!isPositiveDecimal(amount.value)) return $t('exchange.positiveAmount')
  if (compareDecimals(amount.value, aixBalance.value) > 0) return $t('exchange.insufficientAix')
  if (preview.value && !isPositiveDecimal(preview.value.usdtNet)) return $t('exchange.netAmountTooSmall')
  return ''
})
const canSubmit = computed(() => Boolean(amount.value) && !amountError.value && hasRate.value)

function normalizeAmount(event: Event) {
  const input = event.target as HTMLInputElement
  let value = input.value.replace(/[^\d.]/g, '')
  const dot = value.indexOf('.')
  if (dot >= 0) value = value.slice(0, dot + 1) + value.slice(dot + 1).replace(/\./g, '').slice(0, 18)
  value = value.replace(/^0+(?=\d)/, '')
  amount.value = value
  input.value = value
}

function fillAll() {
  amount.value = aixBalance.value
}

function displayAmount(value: unknown) {
  return displayDecimal(value)
}

function displayAixBalance(value: unknown) {
  return displayDecimal(value, 4)
}

function displayPrice(value: unknown) {
  return displayAixPrice(value)
}

function recordToAsset(asset?: string) {
  const value = String(asset || 'USDT').toUpperCase()
  return value === 'WIN' ? 'WIN' : 'USDT'
}

function recordUnitRate(item: AixExchangeRecord) {
  if (isPositiveDecimal(item.from_amount) && isPositiveDecimal(item.to_amount)) {
    return displayAmount(divDecimal(item.to_amount, item.from_amount))
  }
  return '-'
}

function formatTime(value: number) {
  const date = new Date(Number(value) * 1000)
  if (Number.isNaN(date.getTime())) return ''
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

async function loadRecords() {
  recordLoading.value = true
  try {
    const result = await getAixWinExchangeRecords()
    records.value = result?.records || []
  } catch {
    records.value = []
  } finally {
    recordLoading.value = false
  }
}

async function submitExchange() {
  if (!canSubmit.value || submitting.value) return
  submitting.value = true
  try {
    const result = await exchangeAixToWin(amount.value)
    person.profile = {
      ...person.profile,
      aix_balance: result.aix_balance,
      usdt_withdrawable: result.usdt_withdrawable,
    }
    amount.value = ''
    showSuccessToast($t('exchange.success', { amount: displayAmount(result.to_amount) }))
    await Promise.allSettled([person.refreshProfile(), loadRecords()])
  } catch (error: any) {
    const code = error?.response?.data?.reason || error?.response?.data?.code
    const messageKey: Record<string, string> = {
      INVALID_AMOUNT: 'exchange.positiveAmount',
      INSUFFICIENT_AIX: 'exchange.insufficientAix',
      AIX_PRICE_NOT_CONFIGURED: 'exchange.priceUnavailable',
      USDT_NET_AMOUNT_TOO_SMALL: 'exchange.netAmountTooSmall',
      WIN_PRICE_NOT_CONFIGURED: 'exchange.priceUnavailable',
      WIN_NET_AMOUNT_TOO_SMALL: 'exchange.netAmountTooSmall',
    }
    showFailToast(messageKey[code] ? $t(messageKey[code]) : (error?.response?.data?.message || $t('exchange.failed')))
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  await Promise.allSettled([person.refreshProfile(), loadRecords()])
})
</script>

<style scoped lang="scss">
@use '@/style/variables.scss' as *;

/* 全项目最后一处 a3.png 引用（暖橄榄黄底 + 3D 服务器渲染图，418KB）。
   与 wallet.vue 统一为冷色渐晕 + 仪器刻度网格，纯渐变生成，不再拉位图。 */
.exchange-page {
  position: relative;
  isolation: isolate;
  min-height: 100vh;
  /* 纯白页面底。原本是近黑径向渐晕（起点硬编码 #16181B）——
     这是该配方在项目里的第 4 个副本（另三处：body、html 外圈、
     wallet.vue），四份都各自硬编码，所以两轮整站改色全都没带走它们。
     Base 的页面底是一片干净的白，零渐变。 */
  background: var(--ink);

  /* 这里原本还有一层 ::before 装饰网格（32px 等距细线、5% 白、向下淡出），
     与 wallet.vue 那层是同一份复制品。删掉的两个原因：
     1. 5% 白线在白底上完全不可见，改成深色又会变成"格子纸"，
        与 Base 干净的白底相冲；
     2. 它不承载任何信息，纯装饰 —— 留着只会稀释真正要读的数据。 */
}
.page-main { padding: 70px 15px 30px; display: flex; flex-direction: column; gap: 20px; }
/* 这一页原本大量写死高饱和蓝（#8A9096 / #34aef7）与带蓝的描边（#1d4059），
   与其他页统一后的钢青体系不一致。改为复用发丝描边与卡面渐变。 */
/* 拍平为 Base 的卡片配方：纯白 + 发丝边 + 12px 圆角。
   原本那句 box-shadow 里有 `0 10px 28px rgba(0, 0, 0, 0.07)` —— 50% 的黑，
   在白底上会糊出一大团脏灰，是深色版遗留里危害最大的一类写法。 */
.balance-card { min-height: 80px; padding: 14px 20px; box-sizing: border-box; display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; gap: 12px; border: 1px solid var(--hair); border-radius: var(--r-lg); background: var(--surface-1); }
.balance-card > .van-icon { color: var(--accent-bright); font-size: 24px; }
.balance-card div { min-width: 0; display: flex; flex-direction: column; gap: 8px; }
.balance-card span { color: var(--text-3); font-size: 12px; }
/* 数值列右对齐 + 等宽数��，小数点成一条基准线 */
.balance-card .target-balance { text-align: right; }
.balance-card div { font-variant-numeric: tabular-nums; }
.exchange-form, .record-card { padding: 20px; border: 1px solid var(--hair); border-radius: 16px; background: linear-gradient(180deg, var(--surface-1) 0%, var(--ink) 100%); }
.amount-heading { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; color: var(--text-2); font-size: 14px; }
.amount-heading button { border: 0; color: var(--accent-bright); background: transparent; font-size: 13px; cursor: pointer; }
/* 输入框做成内凹槽，和分段控件同一套语言 */
.input-shell { height: 58px; padding: 0 16px; display: flex; align-items: center; gap: 10px; border: 1px solid var(--hair); border-radius: 12px; background: var(--ink-deep); box-shadow: inset 0 2px 6px rgba(0, 0, 0, 0.07); }
.input-shell:focus-within { border-color: var(--accent); box-shadow: inset 0 2px 6px rgba(0, 0, 0, 0.07), 0 0 0 3px var(--accent-dim); }
.input-shell.invalid { border-color: #e65f5f; }
.input-shell input { min-width: 0; flex: 1; border: 0; outline: 0; color: var(--text); background: transparent; font-size: 24px; font-variant-numeric: tabular-nums; letter-spacing: -0.01em; }
.input-shell span { color: var(--text-3); font-size: 13px; font-weight: 500; }
.error-text { margin: 7px 2px 0; color: #f17b7b; font-size: 12px; }
.rate-row { margin-top: 18px; display: flex; justify-content: space-between; gap: 12px; font-size: 13px; }
.rate-row + .rate-row { margin-top: 10px; }
.rate-row span { color: var(--text-3); }
/* 等宽数字 —— 汇率/价格几行需要能上下对齐比对 */
.rate-row strong { text-align: right; color: var(--text); font-variant-numeric: tabular-nums; }
.rate-row.rate-sub { margin-top: 8px; }
/* 次级价格（AIX 价格按业务要求固定 15 位小数，这里靠字号和弱化色降权，
   避免一长串尾随零抢走主汇率的注意力） */
.rate-row.rate-sub strong { font-size: 11px; color: var(--text-3); font-weight: 400; }
.estimate-box { margin: 12px 0 0; padding: 10px 12px; border-radius: 10px; background: var(--surface-2); color: var(--accent-bright); font-size: 12px; line-height: 1.6; }
.rate-note { margin: 10px 0 0; color: var(--text-3); font-size: 12px; line-height: 1.5; }
/* 主按钮改为近白实底 + 近黑字，与全站主操作统一。
   字色必须同时从 #fff 改掉 —— 底色转近白后再用白字就是白底白字。 */
.submit-btn { width: 100%; height: 46px; margin-top: 22px; border: 0; border-radius: 24px; color: var(--ink-deep); background: var(--accent); font-size: 15px; font-weight: 600; }
.submit-btn:disabled { opacity: .4; }
.record-section h2 { margin: 0 0 12px; font-size: 17px; font-weight: 600; }
.record-card { padding: 0 16px; }
.state-box { height: 120px; display: flex; align-items: center; justify-content: center; }
.record-list article { padding: 16px 0; border-bottom: 1px solid var(--hair-2); }
.record-list article:last-child { border-bottom: 0; }
.record-assets { display: flex; align-items: center; gap: 10px; }
.record-assets .van-icon { color: var(--text-3); }
.record-assets .usdt-value { color: var(--text); }
.record-meta { margin-top: 8px; display: flex; justify-content: space-between; color: var(--text-3); font-size: 11px; }
.record-fee { margin-top: 4px; color: var(--text-3); font-size: 11px; }
</style>
