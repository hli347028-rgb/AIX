<template>
<div class='page'>
  <Header />
  <van-nav-bar
    :title="$t('wallet.title')"
    :left-text="$t('common.backHome')"
    left-arrow
    :border="false"
    fixed
    style="top: 60px; left: 50%; width: 100%; max-width: 414px; transform: translateX(-50%)"
    @click-left="handleBack"
  />
  <div class="page-main" style="margin-top: 60px">
    <!-- 余额区：这一页最重要的改动。
         原本 5 个余额全是 12px 同重小字平铺，主数字档位（--fs-hero）空置 ——
         "最该被看到的数字反而最小"是这一页最根本的问题，也是它读起来
         像一张报表而不是一个资产账户的原因。

         现在改为「一个主数字 + 从属账目」：AIX 余额提为主数值（这是本
         产品的核心资产），其余四项降为下方的账目行。层级一旦建立，
         同样的数据就有了主次。 -->
    <div class="usdt-price aix-panel aix-panel--live">
      <div class="balance-head">
        <div class="balance-label-row">
          <span class="aix-signal-dot" aria-hidden="true"></span>
          <span class="balance-label">{{ $t('wallet.aixBalance') }}</span>
        </div>
        <p class="balance-figure">{{ formatFour(aixBalance) }}</p>
      </div>

      <hr class="aix-trace balance-trace" />

      <div class="price-list">
        <div class="price-item">
          <p>{{ $t('wallet.rechargeBalance') }}</p>
          <p>{{ formatFour(rechargeBalance) }}</p>
        </div>
        <div class="price-item">
          <p>{{ $t('wallet.rewardBalance') }}</p>
          <p>{{ formatFour(rewardBalance) }}</p>
        </div>
        <div class="price-item">
          <p>{{ $t('wallet.winBalance') }}</p>
          <p>{{ formatFour(winRechargeBalance) }}</p>
        </div>
        <div class="price-item">
          <p>{{ $t('wallet.usdtWithdrawable') }}</p>
          <p>{{ formatFour(usdtWithdrawable) }}</p>
        </div>
      </div>
    </div>
    <div class="wallet-actions">
      <button type="button" class="wallet-action" @click="router.push('/transfer')">
        <van-icon name="exchange" />
        <span>{{ $t('transfer.title') }}</span>
      </button>
      <button type="button" class="wallet-action" @click="router.push('/exchange')">
        <van-icon name="replay" />
        <span>{{ $t('wallet.exchange') }}</span>
      </button>
      <button type="button" class="wallet-action" @click="router.push('/withdrawal')">
        <van-icon name="balance-pay" />
        <span>{{ $t('withdraw.title') }}</span>
      </button>
    </div>
    <div class="wallet-section-heading">
      <span class="aix-label">{{ $t('wallet.myIncome') }}</span>
    </div>
    <div class="tab-content">
      <div class="pledge">
        <div class="pledge-info">
          <div class="pledge-item">
            <p>{{ $t('wallet.exitRemaining') }}</p>
            <p>{{ formatFour(userinfo.unexitedAmount || 0) }}</p>
          </div>
          <div class="pledge-item">
            <p>{{ $t('wallet.earnedIncome') }}</p>
            <p>{{ formatFour(userinfo.amountGet || 0) }}</p>
          </div>
          <div class="pledge-item">
            <p>{{ $t('wallet.overflowReward') }}</p>
            <p>{{ formatFour(overflowReward) }}</p>
          </div>
          <div class="pledge-item">
            <p>{{ $t('wallet.aixUsdt') }}</p>
            <p>{{ formatFour(profile.points || userinfo.points || profile.points_all || userinfo.points_all || 0) }}</p>
          </div>
        </div>
      </div>
      <div class="pledge-frame">
        <div class="pledge-frame-item">
          <p>{{ $t('wallet.staticIncome') }}</p>
          <p>{{ formatFour(userinfo.location) }}</p>
        </div>
        <div class="pledge-frame-item">
          <p>{{ $t('wallet.directReferralReward') }}</p>
          <p>{{ formatFour(directReferralReward) }}</p>
        </div>
        <div class="pledge-frame-item">
          <p>{{ $t('wallet.managementReward') }}</p>
          <p>{{ formatFour(managementReward) }}</p>
        </div>
        <div class="pledge-frame-item">
          <p>{{ $t('wallet.totalIncome') }}</p>
          <p>{{ formatFour(userinfo.all) }}</p>
        </div>
      </div>
      <van-tabs v-model:active="active" scrollable :ellipsis="false" @change="onChangeTab">
        <van-tab v-for="value in menuType" :title="value[1]" :name="value[0]" :key="value[0]" />
      </van-tabs>
      <div class="records-panel" :aria-busy="rewardLoading">
        <div v-if="rewardLoading" class="records-loading">
          <van-loading type="spinner" color="var(--accent)" />
        </div>
        <div v-else-if="isTableTab" class="subscribe-table" :class="{ 'is-two-col': isTwoColTab }">
          <div class="table-header">
            <span class="col-amount">{{ active === 'points' ? $t('wallet.aixUsdt') : $t('node.amount') }}</span>
            <span v-if="!isTwoColTab" class="col-status">{{ tableMiddleTitle }}</span>
            <span class="col-time">{{ $t('node.time') }}</span>
          </div>
          <van-empty v-if="rewardList.length === 0" :description="emptyRecordsText" :image="emptyImage" />
          <div v-else class="subscribe-table-body">
            <div
              class="income-list-item"
              v-for="(item, index) in rewardList"
              :key="item.id || `${active}-${page}-${index}`"
            >
              <span class="col-amount">{{ formatFour(item.reward) }}</span>
              <span v-if="active === '1'" class="col-status" :class="item.exited ? 'is-exited' : 'is-active'">
                <span class="status-text">{{ item.exited ? $t('node.statusExited') : $t('node.statusActive') }}</span>
                <template v-if="item.progressAcc != null && item.progressTarget">
                  <span class="progress-text">{{ formatFour(item.progressAcc) }} / {{ formatFour(item.progressTarget) }}</span>
                  <span class="progress-bar">
                    <i :style="{ width: `${progressPercent(item)}%` }" />
                  </span>
                </template>
              </span>
              <span v-else-if="active === '3'" class="col-status">
                <span class="status-text">{{ item.address ? formatShortAddr(item.address) : '—' }}</span>
                <span v-if="item.num" class="progress-text">{{ $t('community.generationNum', { num: item.num }) }}</span>
              </span>
              <span class="col-time">
                <span class="date-text">{{ splitDateTime(item.createdAt).date }}</span>
                <span class="time-text">{{ splitDateTime(item.createdAt).time }}</span>
              </span>
            </div>
            <Pagination
              v-if="allPageCount > 1"
              v-model="page"
              :page-count="allPageCount"
              mode="simple"
              @change="getRewardList"
            />
          </div>
        </div>
        <van-empty v-else-if="rewardList.length === 0" :description="$t('wallet.noIncomeOfType')" :image="emptyImage" />
        <div class="income-list" v-else>
          <div class="income-list-main">
            <div class="income-list-item" v-for="(item, index) in rewardList" :key="item.id || `${active}-${page}-${index}`">
              <div class="income-list-item-info">
                <p class="income-list-item-name">{{ item.name || $t('wallet.income') }}</p>
                <p class="income-list-item-time">
                  <span>{{ item.createdAt }}</span>
                  <span class="income-list-item-money">{{ formatFour(item.reward) }}</span>
                </p>
                <p v-if="item.address" class="income-list-item-note">{{ formatShortAddr(item.address) }}</p>
                <p v-if="active === '5'" class="income-list-item-note">
                  {{ $t('wallet.releasedManagementReward') }}: {{ formatFour(item.released || 0) }} / {{ $t('wallet.pendingManagementReward') }}: {{ formatFour(item.pending || 0) }}
                </p>
              </div>
            </div>
            <Pagination
              v-if="allPageCount > 1"
              v-model="page"
              :page-count="allPageCount"
              mode="simple"
              @change="getRewardList"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
</template>
<script setup>
import Header from '@/components/Header.vue'
import userPerson from "@/pinia/person";
import { useRouter } from 'vue-router'
import { onMounted, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import emptyImage from '../../assets/images/custom-empty-image.png'
import request from "@/tools/request";
import { listPointsRecords } from "@/api/aix";
import { Pagination } from "vant"

const { t: $t, locale } = useI18n()
const router = useRouter()
const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const profile = $computed(() => person.profile);
const address = $computed(() => person.address);

const pickBalance = (vals) => {
  let fallback = 0
  for (const v of vals) {
    if (v == null || v === '') continue
    if (fallback === 0 || fallback === '0' || fallback === '0.00') fallback = v
    const n = Number(v)
    if (Number.isFinite(n) && n !== 0) return v
  }
  return fallback
}
const rechargeBalance = $computed(() => pickBalance([userinfo.usdt, profile.usdt_recharge]))
const rewardBalance = $computed(() => pickBalance([userinfo.reward, profile.usdt_reward]))
const aixBalance = $computed(() => pickBalance([profile.aix_balance, userinfo.aix]))
const winRechargeBalance = $computed(() => pickBalance([profile.win_recharge_balance]))
const usdtWithdrawable = $computed(() => pickBalance([profile.usdt_withdrawable, userinfo.usdtWithdrawable]))
const overflowReward = $computed(() => {
  const u = userinfo || {}
  const p = profile || {}
  return u.overflowReward ?? u.overflow_reward ?? p.overflow_reward ?? p.overflowReward ?? p.pending_mgmt_reward ?? 0
})
const directReferralReward = $computed(() => {
  const u = userinfo || {}
  const p = profile || {}
  return u.recommend ?? u.direct_reward_total ?? p.direct_reward_total ?? p.directRewardTotal ?? 0
})
const managementReward = $computed(() => {
  const u = userinfo || {}
  const p = profile || {}
  return u.team ?? u.mgmt_reward_total ?? p.mgmt_reward_total ?? p.mgmtRewardTotal ?? 0
})

let active = $ref('1')
let page = $ref(1);
let allPageCount = $ref(1);
let rewardList = $ref([]);
let rewardLoading = $ref(false)
let rewardRequestId = 0

const menuType = computed(() => [
  ['1', $t('wallet.subscribeRecords')],
  ['2', $t('wallet.staticIncome')],
  ['3', $t('wallet.directReferralReward')],
  ['5', $t('wallet.managementReward')],
  ['points', $t('wallet.pointsRecords')],
])

const isTableTab = computed(() => ['1', '2', '3', 'points'].includes(String(active)))
const isTwoColTab = computed(() => active === '2' || active === 'points')

const tableMiddleTitle = computed(() => {
  if (active === '3') return $t('community.source')
  return $t('node.status')
})

const emptyRecordsText = computed(() => {
  if (active === '1') return $t('wallet.noSubscribeRecords')
  if (active === 'points') return $t('wallet.noPointsRecords')
  return $t('wallet.noIncomeOfType')
})
const formatShortAddr = (value) => {
  if (!value) return ''
  return `${value.slice(0, 6)}...${value.slice(-4)}`
}

const formatFour = (value) => {
  if (value === null || value === undefined || value === '') return '0.0000'
  const str = String(value).trim()
  if (!str || Number.isNaN(Number(str))) return '0.0000'
  const neg = str.startsWith('-')
  const absStr = neg ? str.slice(1) : str
  const [intPart = '0', decPart = ''] = absStr.split('.')
  return `${neg ? '-' : ''}${intPart}.${(decPart + '0000').slice(0, 4)}`
}

const splitDateTime = (value) => {
  const text = String(value || '').trim()
  if (!text) return { date: '—', time: '' }
  const [date, ...rest] = text.split(/\s+/)
  return { date, time: rest.join(' ') }
}

const progressPercent = (item) => {
  const acc = Number(item?.progressAcc)
  const target = Number(item?.progressTarget)
  if (!target || Number.isNaN(acc) || Number.isNaN(target)) return 0
  return Math.min(100, Math.max(0, (acc / target) * 100))
}

const formatUnixDisplay = (value) => {
  const n = Number(value)
  if (!n) return String(value || '')
  const d = new Date(n < 1e12 ? n * 1000 : n)
  const pad = (x) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const getRewardList = async (pageNum = 1, reqType = active) => {
  const requestId = ++rewardRequestId
  const requestedType = String(reqType)
  rewardLoading = true
  try {
    if (requestedType === '1') {
      const res = await request.get("app_server/order_list", {
        params: { page: pageNum }
      });
      if (requestId !== rewardRequestId || requestedType !== active) return
      const count = Number(res?.count || 0)
      allPageCount = Math.max(1, Math.ceil(count / 10));
      rewardList = (res?.list || []).map((item) => {
        const exited = String(item.status) === '2'
        const acc = item.accumulated ?? item.amountGet ?? '0'
        const target = item.exit_target ?? item.amountMax ?? ''
        return {
          name: $t('community.subscribe'),
          exited,
          createdAt: item.createdAt || item.created_at || '',
          reward: item.amount || '0',
          progressAcc: target ? acc : null,
          progressTarget: target || '',
          address: person.address || address || '',
        }
      })
      page = pageNum
      return
    }
    if (requestedType === 'points') {
      const res = await listPointsRecords()
      if (requestId !== rewardRequestId || requestedType !== active) return
      const payload = Array.isArray(res?.records) ? res : (res?.data || res || {})
      const records = Array.isArray(payload?.records) ? payload.records : []
      const count = Number(payload?.count || records.length || 0)
      allPageCount = Math.max(1, Math.ceil(count / 10))
      const start = (pageNum - 1) * 10
      rewardList = records.slice(start, start + 10).map((item) => {
        const status = String(item.status || '').toLowerCase()
        return {
          id: item.id || item.order_id,
          reward: item.points || '0',
          exited: status === 'exited' || status === '2',
          createdAt: formatUnixDisplay(item.created_at ?? item.created_time),
        }
      })
      if (payload?.points != null || payload?.points_all != null) {
        person.userinfo = {
          ...person.userinfo,
          points: payload.points ?? person.userinfo.points,
          points_all: payload.points_all ?? person.userinfo.points_all,
        }
        person.profile = {
          ...person.profile,
          points: payload.points ?? person.profile.points,
          points_all: payload.points_all ?? person.profile.points_all,
        }
      }
      page = pageNum
      return
    }
    const res = await request.get("app_server/reward_list", {
      params: {
        page: pageNum,
        reqType: requestedType
      }
    });
    if (requestId !== rewardRequestId || requestedType !== active) return
    const count = Number(res?.count || 0)
    allPageCount = Math.max(1, Math.ceil(count / 10));
    rewardList = res?.list || []
    page = pageNum
  } catch {
    if (requestId !== rewardRequestId || requestedType !== active) return
    rewardList = []
    allPageCount = 1
  } finally {
    if (requestId === rewardRequestId) rewardLoading = false
  }
}

const formatAddress = (value) => {
  if (!value) return ''
  const frontSix = value.slice(0, 6);
  const backSix = value.slice(-4);
  const middle = '...';
  return frontSix + middle + backSix;
}

watch(locale, () => {
  getRewardList(page, active)
})

onMounted(async () => {
  await Promise.allSettled([person.getUser?.(), person.refreshProfile()])
  getRewardList(1)
})

const onChangeTab = (name) => {
  active = String(name)
  page = 1
  allPageCount = 1
  rewardList = []
  getRewardList(1, active)
}

const handleBack = () => {
  router.push('/')
}

</script>
<style lang='less' scoped>
  /* 页面底：最初铺的是 a3.png（暖橄榄黄底 + 3D 服务器渲染图），
     后来换成冷色渐晕 + 32px 刻度网格，现在再拍平为纯白。
     网格一并删除 —— 它靠 5% 白线成立，白底上完全不可见，
     且 Base 全站零底纹（同 polish.less 里删掉的 .aix-grid-bg）。
     省掉那张 ~100KB 位图的收益保留。 */
  .page {
    --accent: #0052ff;
    --accent-bright: #0052ff;
    --accent-deep: #0648df;
    --accent-dim: rgba(0, 82, 255, 0.1);
    position: relative;
    isolation: isolate;
    min-height: 100vh;
    box-sizing: border-box;
    padding: 50px 15px 20px 15px;

    /* 纯白页面底。原本是 radial-gradient 深色渐晕，起点还硬编码了
       #16181B（近黑）—— 令牌翻转带不走硬编码色，所以它在整站翻白后
       依然是深的。Base 的页面底色是一片干净的白，零渐变。 */
    background: var(--ink);
    .page-main {
      display: flex;
      flex-direction: column;
      gap: 20px;
    }
    /* 余额卡：原本用 a1.png（一枚写实美元金币）撑场面 —— 那张图是暖金色，
       和整套冷色仪器语言彻底冲突，而且占掉右侧 40% 版面，逼得 5 行余额
       只能挤在剩下的 60% 里（这也是 is-compact 那套补丁存在的原因）。

       改成用渐变生成的机雕太阳纹卡面：纹理落在原先放金币的位置，
       既保留了右侧的视觉重量，又不再和文字抢地方，于是宽度可以放到 100%。 */
    .usdt-price {
      position: relative;
      isolation: isolate;
      overflow: hidden;
      width: 100%;
      /* min-height 去掉：卡片高度由内容决定。
         写死 111px 是为了对齐那张已删除的金币位图，现在没有对齐对象了，
         留着只会在内容变化时产生空档（首页 hero 已经踩过这个坑）。 */
      padding: 20px 20px 18px;
      display: block;
      box-sizing: border-box;
      font-size: 12px;

      /* 机雕纹的锚点从 84%（原金币位置）移到 92%，
         并压低透明度 —— 主数字现在占据左侧，纹理要让开，不能和数字抢。 */

      /* —— 主数值区 —— */
      .balance-head {
        display: flex;
        flex-direction: column;
        gap: 8px;
      }

      .balance-label-row {
        display: flex;
        align-items: center;
        gap: 7px;
      }

      .balance-label {
        font-size: var(--fs-micro);
        font-weight: 600;
        letter-spacing: var(--ls-caps);
        text-transform: uppercase;
        color: var(--text-3);
      }

      /* 全站唯一使用 --fs-hero 的地方 —— 这个档位就是为它保留的。
         用蓝色，因为它是实时数据（符合用色铁律的第 1 条）。 */
      .balance-figure {
        margin: 0;
        font-family: var(--aix-font-display);
        font-size: var(--fs-hero);
        font-weight: 500;
        line-height: 1;
        letter-spacing: var(--ls-hero);
        color: var(--accent-bright);
        font-variant-numeric: tabular-nums;
        /* 原本这里有 `text-shadow: 0 0 24px rgba(47,123,255,.45)`，
           删掉的两个原因：
           1. 那是**旧版青蓝**的硬编码值，配色早已换过两轮，它没跟着变；
           2. 白底上 24px 蓝色辉光不会读作"仪器感"，只会在数字周围糊出
              一块毛边蓝斑 —— 实测截图里它确实呈现为一个模糊蓝框。
           Base 没有任何发光，大数字的存在感靠字号和纯色本身。 */
        /* 长数字不换行，靠缩放兜底 */
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }

      .balance-trace {
        margin: 16px 0 14px;
      }

      /* 这里原本有一层"机雕太阳纹"装饰（同心圆 + 放射线叠出莫尔干涉，
         由 repeating-radial-gradient + repeating-conic-gradient 生成）。
         整块删掉，三个理由：
         1. 它画的是 5%~7% 的**白**线 —— 白卡上完全不可见，已成死装饰；
         2. 若改成深色线让它"可见"，等于在余额卡上铺一片纹理噪点，
            而这张卡唯一的职责是让人一眼读到余额；
         3. Base 全站没有任何生成式纹理，卡片就是白底 + 发丝边。 */

      .price-list {
        width: 100%;
        display: flex;
        flex: 1;
        flex-direction: column;
        justify-content: center;
        gap: 4px;

        /* 标签左、数字右 —— 金融界面的常规做法。
           数字右对齐 + 等宽数字后，小数点会形成一条竖直基准线，
           这是"能一眼比大小"和"一堆数字堆在那"的区别。 */
        /* 账目行。AIX 提为主数值后，这几项是从属信息，
           字号和字重都要明确低于主数字 —— 否则又回到"全部同重"的老问题。

           已删除三块死样式：.price-value-action、.price-item-action、
           .exchange-entry。它们对应的 DOM 在余额区重构时已经不存在了
           （AIX 那一行本来带一个兑换入���按钮，现在 AIX 升为主数值，
           那行连同按钮一起没了）。留着不会报错，只会让下一个人以为
           页面上还有这些元素 —— 我甚至刚才还顺手"改进"了其中一块的配色，
           那是纯粹浪费��� */
        .price-item {
          position: relative;
          width: 100%;
          min-height: 20px;
          display: grid;
          grid-template-columns: minmax(72px, auto) minmax(0, 1fr);
          align-items: baseline;
          gap: 12px;

          p {
            margin: 0;
          }

          > p:first-child {
            color: var(--text-3);
            white-space: nowrap;
          }

          > p:nth-child(2) {
            min-width: 0;
            overflow: hidden;
            /* 从 --text 降到 --text-2、字重 600 降到 500：
               这些是从属数值，不该和主数字同等强度。 */
            color: var(--text-2);
            font-weight: 500;
            font-variant-numeric: tabular-nums;
            letter-spacing: -0.01em;
            text-align: right;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }
      }

      /* is-compact 整套已删除。
         它的存在前提是"卡片高度写死（为对齐金币位图），所以第 5 行出现时
         必须把所有字压小才塞得进去"。位图早已删除、min-height 也已去掉，
         卡片现在按内容自适应，多一行就多一行的高度 —— 不需要再把
         字号从 12px 压到 10px 来腾地方。

         这类"为了迁就某个已消失的约束而存在的补丁"，是最容易在重构后
         被留下来的东西：它不报错，只是悄悄让排版变差。 */
    }
    .wallet-actions {
      width: 100%;
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 8px;

      /* 与顶栏 .wallet 完全同一套配色：浅蓝底、品牌蓝文字和低强度蓝框。 */
      .wallet-action {
        min-width: 0;
        height: 42px;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 6px;
        padding: 0 10px;
        box-sizing: border-box;
        border: 1px solid rgba(0, 82, 255, .24);
        border-radius: 18px;
        color: #0052ff;
        background: #f2f5ff;
        font-size: 11px;
        font-weight: 600;
        line-height: 1;
        cursor: pointer;
        transition: color var(--t-fast) var(--ease),
          border-color var(--t-fast) var(--ease),
          background-color var(--t-fast) var(--ease),
          transform var(--t-fast) var(--ease-spring);

        .van-icon {
          color: currentColor;
          font-size: 15px;
          transition: color var(--t-fast) var(--ease);
        }

        span {
          min-width: 0;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        &:hover,
        &:focus-visible {
          outline: none;
          color: var(--on-accent);
          border-color: #0052ff;
          background: #0052ff;
        }

        &:active {
          color: var(--on-accent);
          border-color: var(--accent-deep);
          background: var(--accent-deep);
          transform: scale(.97);
        }
      }
    }
    /* 分段控件：原本选中态是一整块实色 #087BC1，在深色面板上非常刺眼，
       而且和页面里其他蓝色（按钮、链接、图标）互相打架。
       改成"凹槽 + 抬起的滑块"：容器压暗内凹，选中项用浅一档的表面色抬起，
       靠明度差和描边区��，而不是靠饱和度轰。 */
    .wallet-tab {
      width: 100%;
      min-height: 45px;
      background: var(--surface-2);
      border: 1px solid var(--hair);
      border-radius: 31px;
      padding: 5px;
      box-sizing: border-box;
      display: flex;
      /* 内阴影做出凹槽感 */
      box-shadow: inset 0 2px 6px rgba(0, 0, 0, 0.07);

      li {
        flex: 1 0 0;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 26px;
        color: var(--text-3);
        font-size: 13px;
        cursor: pointer;
        transition:
          color var(--t-fast) var(--ease),
          background-color var(--t-fast) var(--ease);

        /* 选中态：白底 + 蓝字 + 蓝框，与节点页/vant 标签同一套语言。
           原本是"灰渐变 + 投影"的抬起滑块 —— 那套拟物依赖深色底才有
           层次，翻白后选中项和未选中几乎无差别。 */
        &.active {
          background: var(--surface-1);
          border: 1.5px solid var(--accent);
          color: var(--accent);
          font-weight: 600;
        }
      }
    }
    /* 与 .usdt-price 同一套卡面语言（原本铺的是 a2.png 位图底）。
       高度放开为 min-height，避免内容换行时被 111px 截断。 */
    .ispay-price {
      width: 100%;
      min-height: 111px;
      border-radius: var(--r-lg);
      border: 1px solid var(--hair);
      /* 拍平为纯白 + 发丝边。这是那份 168° 渐变配方的第 4 个副本
         （.aix-surface / .entries / .aix-panel / 这里），
         同一配方复制四份，改一处必漏其余。 */
      background: var(--surface-1);
      padding: 20px;
      box-sizing: border-box;
      display: flex;
      flex-direction: column;
      gap: 18px;
      font-size: 18px;
      font-variant-numeric: tabular-nums;
    }
    .tab-menu {
      display: flex;
      height: 55px;
      /* 与 .wallet-tab 一致的凹槽 */
      background: var(--surface-2);
      border: 1px solid var(--hair);
      box-shadow: inset 0 2px 6px rgba(0, 0, 0, 0.07);
      border-radius: 31px;
      padding: 5px;
      box-sizing: border-box;
      .tab-item {
        flex: 1;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      /* 与 .wallet-tab 选中态保持同一套语言 */
      .active-tab {
        background: var(--surface-1);
        border: 1.5px solid var(--accent);
        border-radius: var(--r-pill);
        color: var(--accent);
        font-weight: 600;
      }
    }
    .tab-content {
      display: flex;
      flex-direction: column;
      gap: 20px;
      :deep(.van-tabs__wrap) {
        overflow: visible;
      }
      :deep(.van-tabs__nav) {
        overflow-x: auto;
        overflow-y: hidden;
        -webkit-overflow-scrolling: touch;
      }
      :deep(.van-tab) {
        flex: none;
        padding: 0 14px;
        white-space: nowrap;
      }
      :deep(.van-tab__text) {
        overflow: visible;
        white-space: nowrap;
      }
      :deep(.van-tabs__content) {
        display: none;
      }
    /* height → max-height。
       原本是写死的 height: clamp(260px, 45vh, 420px)，目的是让这块
       滚动区高度恒定、切换 tab 时页面不跳动 —— 意图是对的，但代价是
       记录只有 3 条时，面板照样撑到 403px，最后一行下面空 188px。
       实测确认空白就在面板内部，不是外部间距。

       改为 max-height：内容少时按内容收缩，多到超过上限才滚动。
       换来的是切 tab 时高度会变 —— 但"少数据时大片空白"比"切换时高度
       变化"更伤，而且后者可以靠下面的 min-height 兜住����。 */
    .records-panel {
      min-height: 180px;
      max-height: clamp(260px, 45vh, 420px);
      position: relative;
      overflow-y: auto;
      overscroll-behavior: contain;
      -webkit-overflow-scrolling: touch;
    }
      .records-loading {
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
      }
    /* 空状态占位。
       原本是 min-height: 100% —— 依赖父级有确定高度才能解析；
       现在父级改成�� max-height，百分比失去参照会解析失败，
       空状态就会塌成一小条。改用与面板下限一致的绝对值。 */
    .records-panel > :deep(.van-empty),
    .subscribe-table > :deep(.van-empty) {
      min-height: 180px;
      display: flex;
      flex-direction: column;
      justify-content: center;
      box-sizing: border-box;
    }
    /* min-height: 100% 去掉。
       它会让表格无条件撑满 .records-panel，那样上面把 height 改成
       max-height 就完全失效了（表格照样撑到上限、空白��旧）。
       空状态下的居中由 .van-empty 自己的 min-height 负责（见上方规则），
       不需要整张表格陪着一起撑高。 */
    .subscribe-table {
      border: 1px solid var(--hair-2);
        border-radius: 12px;
        background: var(--surface-2);

        .table-header,
        .income-list-item {
          display: grid;
          grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.35fr) minmax(0, 1fr);
          align-items: center;
          column-gap: 6px;
        }

        &.is-two-col {
          .table-header,
          .income-list-item {
            grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
          }
        }

        .table-header {
          position: sticky;
          top: 0;
          z-index: 1;
          padding: 10px 8px;
          border-radius: var(--r-md) var(--r-md) 0 0;
          /* 原为 --ink-deep（近黑）—— 它是 sticky 表头，需要不透明底来
             遮住滚动内容，深色版里正好用页面底色。翻白后这条黑带成了
             白页正中最抢眼的一块，比它标注的数据还重。
             改用白底：同样不透明（sticky 遮挡有效），并靠下方的发丝边
             与内容分隔。 */
          background: var(--surface-1);
          border-bottom: 1px solid var(--hair);

          span {
            min-width: 0;
            text-align: center;
            /* 表头是标签而非数据，退到第三档，并用全大写宽字距 —— 
               Base 表头的处理方式 */
            color: var(--text-3);
            font-size: var(--fs-micro);
            font-weight: 500;
            letter-spacing: var(--ls-caps);
            line-height: 1.3;
          }
        }

        .subscribe-table-body {
          padding: 0 0 8px;
        }

        .income-list-item {
          min-height: 52px;
          padding: 10px 8px;
          border-bottom: 1px solid var(--hair);
          box-sizing: border-box;
          background: var(--surface-1);
          transition: background-color var(--t-fast) var(--ease);

          /* 斑马纹改用中性白。原本是 rgba(8,123,193,.06) —— 初版青蓝残留，
             它会让整张表格的偶数行泛蓝，这正是截图里"行背景发蓝"的来源。
             按用色铁律，表格底纹是装饰，不能用蓝。 */
          &:nth-child(2n) {
            background: var(--surface-2);
          }

          &:last-of-type {
            border-bottom: none;
          }

          /* 悬停时整行轻微提亮，给长表格一个视线锚点 */
          &:hover {
            background: var(--surface-2);
          }

          > span {
            min-width: 0;
            text-align: center;
            color: var(--text-2);
            font-size: var(--fs-sm);
            line-height: 1.3;
          }
        }

        /* 金额列是每行的主信息，用等宽数字并提到最亮的一档，
           与旁边的状态/时间拉开层级。

           写成 span.col-amount 而不是 .col-amount + !important：
           上面的 `> span` 规则会给所有单元格设 --text-2，两者特异性相同时
           后者胜出，所以曾用 !important 压过去 —— 那是掩盖问题而不是解决。
           加上元素选择器把特异性提到 0-1-1，正常级联就够了。 */
        span.col-amount {
          font-family: var(--aix-font-display);
          font-weight: 500;
          font-size: var(--fs-body);
          color: var(--text);
          font-variant-numeric: tabular-nums;
          letter-spacing: var(--ls-tight);
        }

        .col-status {
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 4px;

          .status-text {
            max-width: 100%;
            overflow: hidden;
            /* 原为 #fff —— 白卡上的白字，翻色后完全不可见 */
            color: var(--text);
            font-size: 12px;
            font-weight: 500;
            text-overflow: ellipsis;
            white-space: nowrap;
          }

          /* 进行中：蓝色。这是"激活状态"，符合用色铁律第 3 条。
             硬编码的 #39b7ff 是初版青蓝残留，换成品牌蓝令牌。 */
          &.is-active .status-text {
            color: var(--accent-bright);
          }

          /* 已出局：改为中性灰，不再用绿色。
             原本是 #52c41a —— 既是调色板外的颜色，更重要的是语义错的：
             在金融界面里绿色专指"涨/正收益"，而"已出局"只是一个中性终态，
             染成绿色会被读成"赚了"。终态本就该退到背景里，用灰最准确。
             （--up/--down 两个语义色留给真正的涨跌，不能挪用。） */
          &.is-exited .status-text {
            color: var(--text-3);
          }

          .progress-text {
            max-width: 100%;
            overflow: hidden;
            /* 原为 #CCC —— 对白底只有 1.6:1，远低于可读线。
               改用 --text-2（7.0:1）。 */
            color: var(--text-2);
            font-size: 11px;
            text-overflow: ellipsis;
            white-space: nowrap;
          }

          .progress-bar {
            width: 100%;
            max-width: 72px;
            height: 4px;
            overflow: hidden;
            border-radius: 4px;
            background: var(--surface-3);

            i {
              display: block;
              height: 100%;
              border-radius: 4px;
              background: linear-gradient(90deg, var(--text-2) 0%, var(--accent-bright) 100%);
            }
          }
        }

        .col-time {
          .date-text,
          .time-text {
            display: block;
            line-height: 1.3;
            white-space: nowrap;
          }

          .date-text {
            font-size: 11px;
          }

          .time-text {
            margin-top: 2px;
            color: var(--text-2);
            font-size: 10px;
          }
        }

        :deep(.van-pagination) {
          padding-top: 8px;
        }
      }
      /* 账目列表。
         这里原本是一个 2×2 的四格网格 —— 和紧随其后的 .pledge-frame
         几乎完全一样：同样的两列、同样的居中、同样的"标签在上数值在下"。
         两个近乎相同的面板上下相连，就是"卡片汤"最典型的样子：读者无法
         判断它们有什么区别，也无从知道该先看哪个。

         解决办法不是把它们做得更不一样的装饰，而是让结构反映内容：
         这四项（出局剩余/已获收益/溢出奖励/积分）是账户状态，适合做成
         紧凑的左标签-右数值账目行；下面四项（静态收益/直推/管理/累计）
         是收益构成，适合并列对比、保留网格。
         一个列表 + 一个网格，层级和分工立刻清楚。

         同时删掉了 .pledge-total（模板里没有对应 DOM 的死样式，
         内含 rgb(21,151,229) 又一处初版蓝残留）。 */
      .pledge {
        border: 1px solid var(--hair);
        border-radius: var(--r-lg);
        background: var(--surface-1);
        overflow: hidden;

        .pledge-info {
          display: flex;
          flex-direction: column;
          padding: 4px 16px;

          .pledge-item {
            min-height: 44px;
            padding: 11px 0;
            box-sizing: border-box;
            display: flex;
            flex-direction: row;
            align-items: baseline;
            justify-content: space-between;
            gap: 12px;
            border-bottom: 1px solid var(--hair);

            &:last-child {
              border-bottom: none;
            }

            p {
              margin: 0;
              &:first-child {
                color: var(--text-3);
                font-size: var(--fs-sm);
                line-height: 1.3;
                white-space: nowrap;
              }
              &:nth-child(2) {
                max-width: 100%;
                overflow: hidden;
                font-family: var(--aix-font-display);
                font-weight: 500;
                font-size: var(--fs-lead);
                text-align: right;
                font-variant-numeric: tabular-nums;
                letter-spacing: var(--ls-tight);
                text-overflow: ellipsis;
                white-space: nowrap;
              }
            }
          }
        }
      }

      /* 收益四项。
         原本铺 boxbg1.png 做底 —— 那正是截图里"静态收益面板右下角浮着一个
         灰色 3D 金字塔"的来源，和已删除的 a1/a2/a3.png 是同一批素材。
         改用和 .usdt-price 同一套渐变卡面 + 发丝线分栏。

         同时删除了三块死样式（.pledge-count / .pledge-earnings /
         .pledge-give），它们各自还引用着 xian.png、btnbg.png 两张位图，
         但对应的 DOM 在这一版模板里根本不存在。 */
      .pledge-frame {
        position: relative;
        box-sizing: border-box;
        display: grid;
        grid-template-columns: 1fr 1fr;
        border: 1px solid var(--hair);
        border-radius: var(--r-lg);
        background: var(--surface-1);
        /* 去掉阴影：Base 的卡片只靠一条发丝边分层，不用阴影。
           原本还拼了 `inset 0 1px 0 var(--gloss)` —— 那是既存的无效
           语法（--gloss 本身即完整阴影值），会让整条声明被丢弃。 */
        overflow: hidden;

        .pledge-frame-item {
          display: flex;
          align-items: center;
          justify-content: center;
          flex-direction: column;
          gap: 7px;
          padding: 18px 10px;
          /* 发丝线分栏，取代四个独立盒子 */
          border-right: 1px solid var(--hair);
          border-bottom: 1px solid var(--hair);

          &:nth-child(2n) { border-right: 0; }
          &:nth-child(n+3) { border-bottom: 0; }

          p {
            margin: 0;
            /* 这两处原本写的是 --fs-xs 和 --fs-lg —— 两个我从未定义过的令牌
               （已定义的只有 hero/display/title/lead/body/sm/micro）。
               未定义的 CSS 变量不会报错、不会警告，只是静默失效回退到继承
               字号，于是四个数字会全部塌成同一个大小 —— 恰好就是我在这一页
               要消灭的"全部同重"。这类错误只有核对令牌清单才能发现。 */
            &:first-child {
              font-size: var(--fs-micro);
              letter-spacing: var(--ls-caps);
              text-transform: uppercase;
              color: var(--text-3);
              text-align: center;
            }
            &:nth-child(2) {
              font-family: var(--aix-font-display);
              font-size: var(--fs-title);
              font-weight: 500;
              line-height: 1;
              color: var(--text);
              font-variant-numeric: tabular-nums;
              letter-spacing: var(--ls-tight);
            }
          }
        }
      }
    }
    .income-box {
      display: flex;
      flex-direction: column;
      .income-main {
        display: flex;
        flex-direction: column;
        justify-content: center;
        gap: 20px;
        align-items: center;
        position: relative;
        padding-bottom: 20px;
        &::after {
          content: "";
          position: absolute;
          z-index: 1;
          bottom: 0;
          left: 0;
          width: 100%;
          height: 0.02564rem;
          background: linear-gradient(90deg, transparent 0%, var(--hair-2) 50%, transparent 100%);
        }
        p {
          &:nth-child(1) {
            font-size: 14px;
            color: var(--text-2);
          }
          &:nth-child(2) {
            font-size: 26px;
            color: var(--text);
          }
        }
      }
      .income-footer {
        display: flex;
        flex-wrap: wrap;
        padding-top: 20px;
        gap: 20px 0;
        .income-footer-item {
          width: 25%;
          flex-grow: 1;
          flex-shrink: 0;
          align-items: center;
          display: flex;
          flex-direction: column;
          justify-content: flex-start;
          align-items: center;
          gap: 5px;
          p {
            &:nth-child(1) {
              font-size: 12px;
              color: var(--text-2);
            }
            &:nth-child(2) {
              font-size: 12px;
              color: var(--text);
              display: flex;
              gap: 4px;
              align-items: center;
            }
          }
        }
      }
    }
    .income-list {
      min-height: 100%;
      overflow: hidden;
      .list-menu-select {
        width: 100%;
        height: 40px;
        background: url('data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADAAAAAwCAQAAAD9CzEMAAAAtUlEQVR42u2VUQqCQBRF3y4FoUAoCooCoUAQnJ21gfl0PzdIbOhS+UbfT/HO53g9B0QYcRzHcX4SBPSICJIJOuV7uGGgzdK3GOinpxEjjVrfYCRqPlHiqtJfkAgieYl6cl0j0QmhSZy/Lk+sn5M4flwdWD83sX+72LF+SWIrBDasX5qoXp5UrLdIrJ+nK9ZbJcrHScn/vWWiQMF64wTrTRP2ek6w3j7BevtERETwa9lxHOfvuAOAC4GPzKVVpAAAAABJRU5ErkJggg==') no-repeat right center;
        background-size: 18px 18px;
      }
      .income-list-header {
        width: 100%;
        height: 40px;
        line-height: 40px;
        overflow-x: auto;
        padding-bottom: 10px;
        &::-webkit-scrollbar {
          height: 0;
        }
        .header-list {
          height: 40px;
          line-height: 40px;
          padding: 0 5px;
          display: flex;
          gap: 10px;
          li {
            display: flex;
            align-items: center;
            white-space: nowrap;
            padding: 0 15px;
            border-radius: 6px;
            &.active {
              background: var(--surface-2);
            }
          }
        }
      }
      .income-list-main {
        display: flex;
        flex-direction: column;
        background: var(--surface-2);
        padding: 10px;
        .income-list-item {
          width: 100%;
          box-sizing:border-box;
          -moz-box-sizing:border-box;
          -webkit-box-sizing:border-box;
          padding: 10px;
          border-bottom: 1px solid var(--surface-1);
          &:nth-child(2n) {
            background: var(--surface-2);
          }
          .income-list-item-info {
            width: 100%;
            display: flex;
            flex-direction: column;
            gap: 4px;
            p {
              width: 100%;
              color: var(--text-2);
            }
            .income-list-item-name {
              color: var(--text);
              font-weight: 600;
            }
            .income-list-item-time {
              font-size: 12px;
              display: flex;
              align-items: center;
              justify-content: space-between;
              gap: 8px;

              > span:first-child {
                color: var(--text-3);
              }
            }
            .income-list-item-money {
              flex-shrink: 0;
              color: var(--text);
              font-size: 15px;
              font-weight: 600;
              font-variant-numeric: tabular-nums;
            }
            .income-list-item-note {
              color: var(--text-3);
              font-size: 12px;
            }
          }
        }
      }
    }
  }
</style>
