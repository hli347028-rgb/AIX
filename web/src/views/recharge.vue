<template>
<div class='withdraw-page'>
  <Header />
  <nav class="home-return-bar" :aria-label="$t('common.pageNavigation')">
    <RouterLink to="/" class="home-return-link" :aria-label="$t('common.backHome')">
      <span aria-hidden="true">‹</span>
      {{ $t('common.backHome') }}
    </RouterLink>
  </nav>
  <div class="page-main">
    <!-- 页首。原先是一张带渐变、顶边高光和 34px 投影的"玻璃卡"，
         里面塞三个余额 —— 卡片本身比数字更抢眼。
         现在去掉卡面：主余额直接落在页面上，靠字号建立主次。 -->
    <header class="balance-head">
      <p class="aix-label">{{ $t('recharge.balance') }}</p>
      <div class="balance-primary">
        <span class="balance-value aix-figure">{{ displayAmount(rechargeBalance) }}</span>
        <span class="balance-unit aix-figure-unit">USDT</span>
      </div>

      <div class="balance-secondary">
        <p class="aix-label">{{ $t('recharge.winBalance') }}</p>
        <div class="balance-primary">
          <span class="balance-value aix-figure">{{ displayAmount(winBalance) }}</span>
          <span class="balance-unit aix-figure-unit">WIN</span>
        </div>
      </div>

      <!-- 这是本页唯一的主操作，给它实心填充；其余一切保持安静。 -->
      <button class="aix-btn recharge-btn" type="button" @click="showRecharge">
        <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <path d="M12 4v10" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" />
          <path d="M7.8 10.2 12 14.4l4.2-4.2" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" />
          <path d="M4.6 17.2v1.2a1.6 1.6 0 0 0 1.6 1.6h11.6a1.6 1.6 0 0 0 1.6-1.6v-1.2" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" />
        </svg>
        {{ $t('recharge.recharge') }}
      </button>
    </header>

    <div class="section-title-wrap">
      <div class="title-bar"></div>
      <h3 class="section-title">{{ $t('recharge.rechargeRecord') }}</h3>
    </div>

    <!-- 原先这排标签页有 20px 顶部圆角和一层渐变底色，像一张贴在
         列表上的卡片头；选中态是一根悬在下方 10px 的短蓝条，
         和任何基线都不对齐。现在改为共用一条发丝基线的文字标签。 -->
    <div class="record-tabs" role="tablist">
      <button
        v-for="tab in recordTabs"
        :key="tab.key"
        class="record-tab"
        :class="{ active: recordTab === tab.key }"
        type="button"
        role="tab"
        :aria-selected="recordTab === tab.key"
        @click="switchRecordTab(tab.key)"
      >{{ tab.label }}</button>
    </div>

    <div class="ledger">
      <div class="ledger-head" v-if="currentRecords.length > 0">
        <span>{{ $t('recharge.date') }}</span>
        <span class="num">{{ $t('recharge.amount') }}</span>
        <span class="st">{{ $t('recharge.recordStatus') }}</span>
      </div>
      <div class="ledger-row" v-for="(item, index) in currentRecords" :key="item.id ?? index">
        <span class="when">
          <em>{{ splitDateTime(item.createdAt).date }}</em>
          <i>{{ splitDateTime(item.createdAt).time }}</i>
        </span>
        <span class="num">
          {{ displayAmount(item.amount) }}
          <i>{{ recordAssetUnit }}</i>
        </span>
        <span class="st" :class="statusClass(item.status)">
          {{ rechargeStatusText(item.status) }}
        </span>
      </div>

      <!-- 原先空状态放了一张 50px 位图并压到 0.3 透明度 —— 一个看不清的
           灰色图形不传达任何信息。改为一行文字。 -->
      <p class="empty-state" v-if="currentRecords.length === 0">{{ $t('common.noData') }}</p>

      <Pagination
        v-if="currentRecords.length > 0 && allPageCount > 1"
        v-model="page"
        :page-count="allPageCount"
        mode="simple"
        @change="onPageChange"
      />
    </div>

    <div class="safe-bottom"></div>
  </div>
  <RechargeDialog :getBalance="getBalance" :usdtBalance="usdtBalance" :onChange="handleRechargeChange" ref="rechargeDialogRef" />
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import { Contract, ETH } from "@/tools/contract";
import { ref, computed } from 'vue'
import request from "@/tools/request";
import { closeToast, showLoadingToast } from "vant";
import RechargeDialog from "./subpage/components/rechargeDialog.vue";
import { Pagination } from "vant"
import Header from '@/components/Header.vue'
import { useI18n } from 'vue-i18n'
import { displayDecimal } from '@/tools/decimal'

const USDT = import.meta.env.VITE_USDT ? new Contract(import.meta.env.VITE_USDT, 'ERC20') : null
const BUY_USDT = new Contract(import.meta.env.VITE_BUY_USDT || import.meta.env.VITE_BUY, 'BUY')
const BUY = BUY_USDT // 兼容：USDT 授权检查用 USDT 充值合约

const { t: $t } = useI18n()
const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const profile = $computed(() => person.profile);
const rechargeBalance = $computed(() => String(profile.usdt_recharge || userinfo.usdt || '0'))
const winBalance = $computed(() => String(profile.win_recharge_balance || '0'))
/* winPrice 已移除：它唯一的引用是模板里一行被注释掉的价格展示，
   等于一个永远不会显示的计算属性。 */
const displayAmount = (value) => displayDecimal(value)
const rechargeDialogRef = ref(null)
const recordTab = ref('usdt')

const recordTabs = [
  { key: 'usdt', label: 'USDT' },
  { key: 'win', label: 'WIN' },
]
let usdtRecords = $ref([])
let winRecords = $ref([])
let page = $ref(1)
let allPageCount = $ref(1)
let usdtBalance = $ref("0");
let usdtApproved = $ref(false);

const currentRecords = computed(() => recordTab.value === 'win' ? winRecords : usdtRecords)
const recordAssetUnit = computed(() => recordTab.value === 'win' ? 'WIN' : 'USDT')

const rechargeStatusText = (status) => {
  switch (String(status || '').toLowerCase()) {
    case 'confirmed': return $t('recharge.statusConfirmed')
    case 'rejected': return $t('recharge.statusRejected')
    case 'pending':
    default: return $t('recharge.statusPending')
  }
}

const statusClass = (status) => {
  const value = String(status || 'pending').toLowerCase()
  if (value === 'confirmed') return 'is-confirmed'
  if (value === 'rejected') return 'is-rejected'
  return 'is-pending'
}

const splitDateTime = (value) => {
  const text = String(value || '-')
  const parts = text.split(' ')
  if (parts.length >= 2) {
    return { date: parts[0], time: parts.slice(1).join(' ') }
  }
  return { date: text, time: '' }
}

const switchRecordTab = async (tab) => {
  if (recordTab.value === tab) return
  recordTab.value = tab
  page = 1
  if (tab === 'win') await getWinRecords(1)
  else await getUsdtRecords(1)
}

const getUsdtApproved = async () => {
    if (!USDT) return false
    await ETH.getAccount('eoeo')
    let res = await USDT.call("allowance", [ETH.account, BUY.address]);
    usdtApproved = Number(res) > 0;
    closeToast()
    return usdtApproved
}

const getBalance = async () => {
  await ETH.getAccount('eoeo')
  const res = await ETH.getUSDTBalance()
  usdtBalance = res;
}

const getData = async () => {
    await Promise.allSettled([
      person.refreshProfile?.(),
      ...(import.meta.env.VITE_USDT ? [getBalance(), getUsdtApproved()] : []),
    ])
}

getData()

function showRecharge() {
  rechargeDialogRef.value?.open()
}

const getUsdtRecords = async (pageNum = 1) => {
  try {
    const res = await request.get('app_server/deposit_list', {
      params: { page: pageNum },
    })
    allPageCount = Math.max(1, Math.ceil(Number(res.count || 0) / 10))
    usdtRecords = (res.list || []).map((item, index) => ({
      ...item,
      id: item.id ?? index,
      createdAt: item.createdAt || item.created_at || '-',
      status: item.status || 'pending',
    }))
    page = pageNum
  } catch {
    usdtRecords = []
    allPageCount = 1
  }
}

const getWinRecords = async (pageNum = 1) => {
  try {
    const res = await request.get('app_server/deposit_win_list', {
      params: { page: pageNum },
    })
    allPageCount = Math.max(1, Math.ceil(Number(res.count || 0) / 10))
    winRecords = (res.list || res.recharges || []).map((item, index) => ({
      ...item,
      id: item.id ?? index,
      createdAt: item.createdAt || item.created_at || '-',
      status: item.status || 'pending',
    }))
    page = pageNum
  } catch {
    winRecords = []
    allPageCount = 1
  }
}

const onPageChange = async (pageNum = 1) => {
  if (recordTab.value === 'win') await getWinRecords(pageNum)
  else await getUsdtRecords(pageNum)
}

const handleRechargeChange = async (pageNum = 1) => {
  const loadRecords = recordTab.value === 'win'
    ? getWinRecords(pageNum)
    : getUsdtRecords(pageNum)
  await Promise.allSettled([
    person.refreshProfile?.(),
    getBalance(),
    loadRecords,
  ])
}

getUsdtRecords()

</script>
<style lang="scss" scoped>
/* 本页原先 @use variables.scss，用的是 $brand-primary / $text-muted /
   $bg-card / $border-color 等旧变量 —— 与全站新令牌体系并行存在两套颜色。
   已全部改为 var(--*) 令牌，旧变量依赖移除。 */

/* 原先 .withdraw-page 是 100vh + overflow hidden，只让内层列表滚动。
   这种嵌套滚动在移动端很别扭：列表区被压成一个小窗口，
   且和本站其它页面（整页滚动）行为不一致。改为整页自然滚动。 */
.withdraw-page {
  --accent: #0052ff;
  --accent-bright: #0052ff;
  --accent-deep: #0648df;
  --accent-dim: rgba(0, 82, 255, 0.1);
  min-height: 100vh;
  /* 顶栏是 fixed 定位，这里要留出它的高度，
     与 community / mine 两页保持一致。 */
  padding-top: 64px;
}

.page-main {
  /* 不再设 max-width：polish.less 把 body > #app 限死在 414px，
     这里写 760px 永远不会生效。 */
  padding: 0 20px;
}

/* ------------------------------- 页首 ------------------------------- */
.balance-head {
  padding: 28px 0 0;
}

.balance-primary {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-top: 10px;
}

/* 外观来自 .aix-figure / .aix-figure-unit 原语（见 polish.less 第 10a-2 节）， 
   这里不再重复金属渐变的定义。 */

.balance-secondary {
  margin-top: 28px;
  padding-top: 24px;
  border-top: 1px solid var(--hair);
}

/* 外观全部来自 .aix-btn 原语，这里只留本页需要的外边距。 */
.recharge-btn {
  margin-top: 22px;
}

/* ----------------------------- 标签页 ----------------------------- */
.record-tabs {
  display: flex;
  gap: 26px;
  border-bottom: 1px solid var(--hair);
}

.record-tab {
  position: relative;
  padding: 0 0 11px;
  background: transparent;
  border: 0;
  font-family: var(--aix-font);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.06em;
  color: var(--text-3);
  cursor: pointer;
  transition: color var(--t-fast) var(--ease);

  /* 选中标记压在基线上，与容器底边严格重合 —— 
     原实现悬在下方 10px，视觉上无所依附。 */
  &::after {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    bottom: -1px;
    height: 1px;
    background: var(--accent-bright);
    opacity: 0;
    transition: opacity var(--t-fast) var(--ease);
  }

  &:hover {
    color: var(--text-2);
  }

  &.active {
    color: var(--text);

    &::after {
      opacity: 1;
    }
  }

  &:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }
}

/* ------------------------------ 记录表 ------------------------------ */
.ledger {
  margin-top: 2px;
}

.ledger-head,
.ledger-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) minmax(0, 0.9fr);
  gap: 12px;
  align-items: baseline;
}

.ledger-head {
  padding: 12px 0;
  border-bottom: 1px solid var(--hair);

  span {
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--text-3);
  }
}

.ledger-row {
  padding: 14px 0;
  border-bottom: 1px solid var(--hair);

  &:last-of-type {
    border-bottom: 0;
  }

  .when {
    em {
      display: block;
      font-style: normal;
      font-variant-numeric: tabular-nums;
      font-size: 12px;
      color: var(--text-2);
    }

    i {
      display: block;
      margin-top: 3px;
      font-style: normal;
      font-variant-numeric: tabular-nums;
      font-size: 10px;
      color: var(--text-3);
    }
  }

  .num {
    font-family: var(--aix-font-display);
    font-variant-numeric: tabular-nums;
    font-size: 15px;
    font-weight: 500;
    color: var(--text);

    i {
      display: block;
      margin-top: 3px;
      font-style: normal;
      font-family: var(--aix-font);
      font-size: 10px;
      font-weight: 400;
      letter-spacing: 0.08em;
      color: var(--text-3);
    }
  }

  /* 状态：原先用 #52c41a / #faad14 / #ff4d4f 三个饱和的原始 hex，
     那是 Ant Design 的默认色，和本站的钢青体系不搭，且凭空多出三种颜色。
     现在复用已有的 --up / --down 语义色；"待处理"本就是未定状态，
     用中性灰而不是再引入一个橙色。 */
  .st {
    font-size: 11px;
    font-weight: 500;
    color: var(--text-3);

    &.is-confirmed {
      color: var(--up);
    }

    &.is-rejected {
      color: var(--down);
    }
  }
}

.ledger-head .num,
.ledger-head .st,
.ledger-row .num,
.ledger-row .st {
  text-align: right;
}

.empty-state {
  margin: 0;
  padding: 60px 0;
  text-align: center;
  font-size: 12px;
  color: var(--text-3);
}

.safe-bottom {
  height: 56px;
}
</style>
