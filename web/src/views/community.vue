<template>
  <div class="community-page">
    <Header />
    <nav class="home-return-bar" :aria-label="$t('common.pageNavigation')">
      <RouterLink to="/" class="home-return-link" :aria-label="$t('common.backHome')">
        <span aria-hidden="true">‹</span>
        {{ $t('common.backHome') }}
      </RouterLink>
    </nav>
    <div class="container">

      <!-- 页首：等级是这一页唯一的"身份"信息，让它成为主角。
           原先是「16px 灰标签 + 20px 青色值」并排，两者体量接近，
           谁都不突出。现在拉开落差：细排字标签在上，大字号数值在下。 -->
      <header class="page-head">
        <p class="aix-label">{{ $t('community.communityLevel') }}</p>
        <p class="level-value aix-figure">{{ levelLabel }}</p>
      </header>

      <section class="team-dashboard" aria-labelledby="team-dashboard-title">
        <div class="team-dashboard-head">
          <div>
            <p class="aix-label">{{ $t('community.teamDataCenter') }}</p>
            <h2 id="team-dashboard-title">{{ $t('community.myTeam') }}</h2>
          </div>
          <button type="button" class="refresh-btn" :disabled="refreshLocked || teamLoading" @click="refreshTeamPage">{{ $t('community.refresh') }}</button>
        </div>

        <div class="team-summary team-summary-single">
          <div><span>{{ $t('community.teamTotalMembers') }}</span><strong>{{ teamTotal }}</strong></div>
        </div>

        <div class="username-setting">
          <div class="username-setting-head">
            <span>{{ $t('community.usernameSetting') }}</span>
            <button v-if="!editingUsername" type="button" class="username-edit-btn" @click="startUsernameEdit">
              {{ $t('community.editUsername') }}
            </button>
          </div>
          <strong v-if="!editingUsername" class="username-current">{{ currentUsername }}</strong>
          <form v-else class="username-form" @submit.prevent="saveUsername">
            <label class="sr-only" for="team-username">{{ $t('community.username') }}</label>
            <input
              id="team-username"
              v-model="usernameDraft"
              type="text"
              maxlength="24"
              autocomplete="nickname"
              :placeholder="$t('community.username')"
              :disabled="savingUsername"
            />
            <div class="username-actions">
              <button type="button" :disabled="savingUsername" @click="cancelUsernameEdit">{{ $t('community.cancelUsername') }}</button>
              <button type="submit" class="primary" :disabled="savingUsername">{{ $t('community.saveUsername') }}</button>
            </div>
          </form>
        </div>
      </section>

      <section class="metrics-section" aria-labelledby="team-metrics-title">
        <div class="section-title-wrap">
          <div class="title-bar"></div>
          <h3 id="team-metrics-title" class="section-title">{{ $t('community.teamDataOverview') }}</h3>
        </div>

        <div class="metric-group">
          <h4>{{ $t('community.teamScale') }}</h4>
          <div class="aix-metrics">
            <div><span class="k">{{ $t('community.directReferralCount') }}</span><span class="v">{{ formatCount(userinfo.recommendNum) }}</span></div>
            <div><span class="k">{{ $t('community.activeSubscription') }}</span><span class="v">{{ formatNum(teamActiveSubscribe) }}</span></div>
          </div>
        </div>

        <div class="metric-group">
          <h4>{{ $t('community.performanceBreakdown') }}</h4>
          <div class="aix-metrics">
            <div><span class="k">{{ $t('community.teamTotalPerformance') }}</span><span class="v">{{ formatNum(userinfo.total) }}</span></div>
            <div><span class="k">{{ $t('community.regionalPerformance') }}</span><span class="v">{{ formatNum(userinfo.max) }}</span></div>
            <div><span class="k">{{ $t('community.smallAreaPerformance') }}</span><span class="v">{{ formatNum(userinfo.min) }}</span></div>
          </div>
        </div>

        <div class="metric-group">
          <h4>{{ $t('community.incomeAndRewards') }}</h4>
          <div class="aix-metrics">
            <div><span class="k">{{ $t('community.directReferralReward') }}</span><span class="v">{{ formatNum(directReferralReward) }}</span></div>
            <div><span class="k">{{ $t('community.staticIncomeTotal') }}</span><span class="v">{{ formatNum(userinfo.location) }}</span></div>
            <div><span class="k">{{ $t('community.managementReward') }}</span><span class="v">{{ formatNum(managementReward) }}</span></div>
            <div><span class="k">{{ $t('wallet.overflowReward') }}</span><span class="v">{{ formatNum(overflowReward) }}</span></div>
            <div class="metric-total"><span class="k">{{ $t('community.incomeTotal') }}</span><span class="v">{{ formatNum(incomeTotal) }}</span></div>
            <div v-if="showCommunitySubsidy">
              <span class="k">{{ $t('community.communitySubsidy') }}</span>
              <span class="v">{{ formatNum(communitySubsidyReward) }}</span>
            </div>
          </div>
        </div>
      </section>

      <!-- 地址区。原先标题是青色、地址值是普通白色 —— 层级正好倒置：
           标签被强调，真正的数据反而退后。现在标签用次要色细排字，
           值用等宽数字并提亮。 -->
      <div class="addr-block">
        <p class="aix-label">{{ $t('community.superiorAddress') }}</p>
        <p class="addr-value aix-mono">{{ formatAddress(userinfo.inviteUserAddress) || '-' }}</p>
      </div>

      <div class="addr-block">
        <p class="aix-label">{{ $t('community.myInviteLink') }}</p>
        <div class="addr-row">
          <p class="addr-value aix-mono">{{ inviteUrl }}</p>
          <!-- 原先是一个 <i> 元素用 base64 PNG 做背景图：键盘无法聚焦、
               读屏软件也不会识别为控件。改为真正的 button + 矢量图标。 -->
          <button
            class="copy-btn"
            type="button"
            :aria-label="$t('common.copy')"
            @click="copyToClipboard(inviteUrl)"
          >
            <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <rect x="9" y="9" width="11" height="11" rx="2.5" stroke="currentColor" stroke-width="1.4" />
              <path d="M15 5.5A2.5 2.5 0 0 0 12.5 3H6.5A2.5 2.5 0 0 0 4 5.5v6A2.5 2.5 0 0 0 6.5 14" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
            </svg>
          </button>
        </div>
      </div>


      <div class="section-title-wrap">
        <div class="title-bar"></div>
        <h3 class="section-title">{{ $t('community.teamMembers') }}</h3>
      </div>

      <div v-if="teamLoading" class="team-state">{{ $t('common.loading') }}</div>
      <div v-else-if="teamMembers.length === 0" class="team-state">{{ $t('community.noTeamMembers') }}</div>
      <ul v-else class="team-tree">
        <TeamTreeNode
          v-for="member in teamMembers"
          :key="member.address"
          :node="member"
          :load-children="loadChildren"
          :team-count-label="$t('community.teamTotalMembers')"
          :direct-count-label="$t('community.directMembers')"
          :personal-performance-label="$t('community.personalPerformance')"
          :team-performance-label="$t('community.teamPerformance')"
          :copy-label="$t('common.copy')"
          :username-placeholder="$t('community.username')"
          :not-set-label="$t('community.notSet')"
          :loading-label="$t('common.loading')"
          :empty-label="$t('community.noTeamMembers')"
          @copy="copyToClipboard"
        />
      </ul>

      <div class="section-title-wrap">
        <div class="title-bar"></div>
        <h3 class="section-title">{{ $t('community.directInviteData') }}</h3>
      </div>

      <div class="ledger">
        <div class="ledger-head">
          <span>{{ $t('community.walletAddress') }}</span>
          <span class="num">{{ $t('community.usdtAmount') }}</span>
          <span class="time">{{ $t('community.time') }}</span>
        </div>
        <template v-if="downlineRechargeList.length > 0">
          <div class="ledger-row" v-for="(item, index) in downlineRechargeList" :key="item.id || index">
            <span class="aix-mono">{{ formatAddr(item.address) }}</span>
            <span class="num">{{ formatNum(item.amount) }}</span>
            <span class="time">{{ item.createdAt }}</span>
          </div>
          <Pagination
            v-model="downlinePage"
            :page-count="downlinePageCount"
            mode="simple"
            @change="getDownlineRecharges"
          />
        </template>
        <p v-else class="empty-state">{{ $t('common.noData') }}</p>
      </div>


      <div class="safe-bottom"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Header from '@/components/Header.vue'
import TeamTreeNode from '@/components/TeamTreeNode.vue'
import userPerson from '@/pinia/person'
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { showToast } from 'vant'
import copy from 'copy-to-clipboard'
import request from '@/tools/request'
import { Pagination } from 'vant'
import { useI18n } from 'vue-i18n'

const person = userPerson()
const { t: $t } = useI18n()
const userinfo = computed<Record<string, any>>(() => person.userinfo)
let teamMembers = $ref<any[]>([])
let teamLoading = $ref(false)
let refreshLocked = $ref(false)
let editingUsername = $ref(false)
let savingUsername = $ref(false)
let usernameDraft = $ref('')
const REFRESH_COOLDOWN_MS = 2000
let refreshCooldownTimer: ReturnType<typeof setTimeout> | null = null

const firstValue = (...values: any[]) => values.find((value) => value !== undefined && value !== null && value !== '')
const teamTotal = computed(() => formatCount(firstValue(
  userinfo.value?.recommendTeamNum,
  userinfo.value?.teamTotalMembers, userinfo.value?.team_total_members,
  userinfo.value?.teamNum, userinfo.value?.team_num, userinfo.value?.countLow,
  teamMembers.length,
)))
const teamActiveSubscribe = computed(() => firstValue(
  userinfo.value?.teamActiveSubscribe,
  userinfo.value?.team_active_subscribe_principal,
  0,
))
const savedUsername = computed(() => String(firstValue(
  person.profile?.username, person.profile?.userName, person.profile?.nickname,
  userinfo.value?.username, userinfo.value?.userName, userinfo.value?.nickname,
) || '').trim())
const currentUsername = computed(() => savedUsername.value || $t('community.username'))

const startUsernameEdit = () => {
  usernameDraft = savedUsername.value
  editingUsername = true
}
const cancelUsernameEdit = () => {
  usernameDraft = ''
  editingUsername = false
}
const saveUsername = async () => {
  const username = usernameDraft.trim()
  if (!username) {
    showToast($t('community.usernameRequired'))
    return
  }
  if ([...username].length > 24) {
    showToast($t('community.usernameTooLong'))
    return
  }
  savingUsername = true
  try {
    const res: any = await request.patch('/v1/auth/profile', { username })
    const saved = String(res?.username || username).trim()
    if (person.profile) person.profile = { ...person.profile, username: saved }
    if (person.userinfo) person.userinfo = { ...person.userinfo, username: saved }
    await Promise.allSettled([person.refreshProfile?.(), person.getUser?.()])
    await loadTeamMembers()
    editingUsername = false
    usernameDraft = ''
    showToast($t('community.usernameSaved'))
  } finally {
    savingUsername = false
  }
}

const managementReward = computed(() => {
  const u = userinfo.value || {}
  const p = person.profile || {}
  return u.team ?? u.mgmt_reward_total ?? p.mgmt_reward_total ?? p.mgmtRewardTotal ?? 0
})
const directReferralReward = computed(() => {
  const u = userinfo.value || {}
  const p = person.profile || {}
  return u.recommend ?? u.direct_reward_total ?? p.direct_reward_total ?? p.directRewardTotal ?? 0
})
const overflowReward = computed(() => {
  const u = userinfo.value || {}
  const p = person.profile || {}
  return u.overflowReward ?? u.overflow_reward ?? p.overflow_reward ?? p.overflowReward ?? p.pending_mgmt_reward ?? 0
})
const incomeTotal = computed(() => userinfo.value?.all ?? 0)
const communitySubsidyTiers = [5, 10, 15]
const showCommunitySubsidy = computed(() => {
  const u = userinfo.value || {}
  const p = person.profile || {}
  const enabled = !!(u.is_community_subsidy ?? u.isCommunitySubsidy ?? p.is_community_subsidy ?? p.isCommunitySubsidy)
  const raw = u.community_subsidy_rate ?? u.communitySubsidyRate ?? p.community_subsidy_rate ?? p.communitySubsidyRate ?? 0
  const rate = parseInt(String(raw), 10)
  return enabled && communitySubsidyTiers.includes(rate)
})
const communitySubsidyReward = computed(() => {
  const u = userinfo.value || {}
  const p = person.profile || {}
  return u.community_subsidy_total ?? u.communitySubsidyTotal ?? p.community_subsidy_total ?? p.communitySubsidyTotal ?? 0
})

// 使用 ?code= 便于本地登录弹窗预填（与 eth_authorize 邀请码一致）
const inviteUrl = computed(() => {
  const addr = person.address || ''
  if (!addr) return ''
  return `${window.location.origin}/?code=${addr}`
})

const levelLabel = computed(() => {
  const raw = String(
    userinfo.value?.communityLevel ||
      userinfo.value?.community_level ||
      userinfo.value?.level ||
      '0',
  )
    .trim()
    .toUpperCase()
  const n = parseInt(raw.replace(/^[WAV]/g, ''), 10)
  const lv = Number.isFinite(n) && n > 0 ? Math.min(n, 10) : 0
  return `A${lv}`
})

let downlineRechargeList = $ref<any[]>([])
let downlinePage = $ref(1)
let downlinePageCount = $ref(1)

const formatAddress = (value: string) => {
  if (!value) return ''
  const frontSix = value.slice(0, 6)
  const backSix = value.slice(-4)
  const middle = '...'
  return frontSix + middle + backSix
}

const formatAddr = (value: string) => formatAddress(value) || '-'

const formatNum = (value: any) => Number(value || 0).toFixed(2)
const formatCount = (value: any) => Math.max(0, Number(value || 0)).toLocaleString()

const normalizeMember = (item: any) => {
  const directCount = firstValue(item.directCount, item.direct_count, item.recommendNum, item.recommend_num, item.countLow, 0)
  return {
    ...item,
    username: firstValue(item.username, item.userName, item.nickname, item.name),
    address: firstValue(item.address, item.walletAddress, item.wallet_address),
    teamCount: firstValue(item.teamCount, item.team_count, item.team_downline_count, item.teamDownlineCount, item.teamNum, item.team_num, item.communityCount, item.community_count, 0),
    directCount,
    personalPerformance: firstValue(item.personalPerformance, item.personal_performance, item.personal_stake, item.personalStake, item.performance, 0),
    teamPerformance: firstValue(item.teamPerformance, item.team_performance, item.team_stake, item.teamTotalPerformance, item.team_total_performance, item.teamTotal, item.team_total, 0),
    hasChildren: item.hasChildren != null ? !!item.hasChildren : Number(directCount) > 0,
    children: item.children,
    childrenLoaded: Array.isArray(item.children),
  }
}

const fetchDirectMembers = async (address: string) => {
  const res: any = await request.get('app_server/recommend_list', {
    params: { address },
  })
  return (res.recommends || res.list || []).map(normalizeMember)
}

const loadChildren = async (node: any) => {
  if (!node.address || node.childrenLoaded) return
  try {
    node.children = await fetchDirectMembers(node.address)
  } catch {
    node.children = []
  } finally {
    node.childrenLoaded = true
  }
}

const loadTeamMembers = async () => {
  const address = person.address
  if (!address) {
    teamMembers = []
    return
  }
  teamLoading = true
  try {
    teamMembers = await fetchDirectMembers(address)
  } catch {
    teamMembers = []
  } finally {
    teamLoading = false
  }
}

const copyToClipboard = (text: string) => {
  copy(text)
  showToast($t('common.copiedToClipboard'))
}

const getDownlineRecharges = async (pageNum: number = 1) => {
  const res: any = await request.get('app_server/downline_recharges', {
    params: { page: pageNum },
  })
  downlinePageCount = Math.ceil((res.count || 0) / 10) || 1
  downlineRechargeList = res.list || []
}

const refreshTeamPage = async () => {
  if (refreshLocked) return
  refreshLocked = true
  try {
    await Promise.allSettled([person.getUser?.(), person.refreshProfile?.()])
    await Promise.allSettled([getDownlineRecharges(downlinePage), loadTeamMembers()])
  } finally {
    refreshCooldownTimer = setTimeout(() => {
      refreshLocked = false
      refreshCooldownTimer = null
    }, REFRESH_COOLDOWN_MS)
  }
}

onMounted(() => {
  refreshTeamPage()
})
onBeforeUnmount(() => {
  if (refreshCooldownTimer) clearTimeout(refreshCooldownTimer)
})
</script>

<style lang="scss" scoped>
/* 说明：原先这里有一整块 .stats-grid 样式（约 30 行），但模板里
   从来没有任何元素用过这个类 —— 已随本次重写删除。
   .info-card / .performance-list / .table-card 等旧类名同样一并移除，
   数据网格与表格改用 polish.less 里的共享原语。 */

.community-page {
  --accent: #0052ff;
  --accent-bright: #0052ff;
  --accent-deep: #0648df;
  --accent-dim: rgba(0, 82, 255, 0.1);
  min-height: 100vh;
  padding-top: 64px;
}

.container {
  /* 不���设 max-width：polish.less 把 body > #app 限死在 414px，
     这里写 760px 永远不会生效。 */
  padding: 0 20px;
}

.team-dashboard {
  padding: 22px 0 24px;
  border-top: 1px solid var(--hair);
  border-bottom: 1px solid var(--hair);
}

.team-dashboard-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.team-dashboard-head h2 {
  margin: 6px 0 0;
  color: var(--text);
  font-size: 24px;
}

.refresh-btn {
  padding: 8px 12px;
  border: 1px solid var(--hair-2);
  border-radius: var(--r-sm);
  background: transparent;
  color: var(--accent);
  cursor: pointer;
}

.team-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-top: 20px;
  border: 1px solid var(--hair);
}

.team-summary > div {
  min-width: 0;
  padding: 14px 10px;
  border-right: 1px solid var(--hair);
}
.team-summary > div:last-child { border-right: 0; }
.team-summary-single { grid-template-columns: 1fr; }
.team-summary-single > div { border-right: 0; }
.team-summary span { display: block; min-height: 28px; font-size: 10px; color: var(--text-3); }
.team-summary strong { display: block; margin-top: 8px; color: var(--text); font-size: 18px; overflow-wrap: anywhere; }

.refresh-btn:disabled { opacity: .55; cursor: wait; }

.username-setting {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid var(--hair);
}
.username-setting-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--text-3);
  font-size: 11px;
}
.username-edit-btn,
.username-actions button {
  padding: 7px 11px;
  border: 1px solid var(--hair-2);
  border-radius: var(--r-sm);
  background: transparent;
  color: var(--text-2);
  cursor: pointer;
}
.username-edit-btn { color: var(--accent); }
.username-current {
  display: block;
  margin-top: 8px;
  color: var(--text);
  font-size: 16px;
  overflow-wrap: anywhere;
}
.username-form { display: grid; gap: 10px; margin-top: 10px; }
.username-form input {
  width: 100%;
  min-height: 42px;
  padding: 0 12px;
  border: 1px solid var(--hair-2);
  border-radius: var(--r-sm);
  outline: none;
  background: rgba(255,255,255,.025);
  color: var(--text);
  font-size: 14px;
}
.username-form input:focus { border-color: var(--accent); }
.username-actions { display: flex; justify-content: flex-end; gap: 8px; }
.username-actions .primary { border-color: var(--accent); background: var(--accent); color: #fff; }
.username-actions button:disabled { opacity: .55; cursor: wait; }

.team-state { padding: 30px 0; text-align: center; color: var(--text-3); }
.team-tree { display: grid; gap: 9px; margin: 0; padding: 0; }

.metrics-section { padding-bottom: 8px; }
.metric-group { margin-top: 18px; }
.metric-group h4 {
  margin: 0 0 9px;
  color: var(--text-3);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: .12em;
}
.metric-group .aix-metrics {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-top: 0;
}
.metric-group .aix-metrics > :last-child:nth-child(odd) {
  grid-column: 1 / -1;
}
.metric-group .metric-total {
  background: var(--accent-dim);
}
.metric-group .metric-total .k { color: var(--accent); }
.metric-group .metric-total .v { color: var(--accent-bright); }

/* 页首 */
.page-head {
  padding: 26px 0 22px;
}

/* ���观来自 .aix-figure 原语（见 polish.less 第 10a-2 节），
   这里只留本页的外边距。 */
.level-value {
  margin: 10px 0 0;
  color: var(--accent);
  }

/* 地址块。不做成卡片 —— 一条发丝线分隔就够，
   页面因此从"一叠方框"变回"一份文档"。 */
.addr-block {
  padding: 16px 0;
  border-top: 1px solid var(--hair);
}

.addr-value {
  margin: 7px 0 0;
  font-size: 14px;
  line-height: 1.55;
  color: var(--text);
  overflow-wrap: anywhere;
}

.addr-row {
  display: flex;
  align-items: flex-start;
  gap: 14px;

  .addr-value {
    flex: 1 1 auto;
    min-width: 0;
  }
}

.copy-btn {
  flex: 0 0 auto;
  width: 34px;
  height: 34px;
  margin-top: 2px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  background: transparent;
  border: 1px solid var(--hair-2);
  border-radius: var(--r-sm);
  color: var(--text-3);
  cursor: pointer;
  transition: color var(--t-fast) var(--ease), border-color var(--t-fast) var(--ease);

  svg {
    width: 16px;
    height: 16px;
  }

  &:hover {
    color: var(--accent-bright);
    border-color: var(--accent-deep);
  }

  /* 键盘可见的聚焦环 —— 原来的 <i> 元素根本无法聚焦。 */
  &:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }
}

/* 账目表。三列网格，表头与表体共用同一套列宽定义，
   这样列标题永远对得上下面的值。 */
.ledger {
  margin-top: 2px;
}

.ledger-head,
.ledger-row {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px;
  align-items: baseline;
}

.ledger-head {
  padding: 10px 0;
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
  font-size: 13px;
  color: var(--text-2);

  &:last-of-type {
    border-bottom: 0;
  }

  .num {
    font-family: var(--aix-font-display);
    font-variant-numeric: tabular-nums;
    font-size: 15px;
    font-weight: 500;
    color: var(--text);

    /* 代数标记跟在金额后面，作为附注而不是另起一行。 */
    em {
      display: block;
      margin-top: 3px;
      font-size: 10px;
      font-style: normal;
      font-weight: 400;
      letter-spacing: 0.04em;
      color: var(--text-3);
    }

    &.accent {
      color: var(--accent-bright);
    }
  }

  .time {
    font-variant-numeric: tabular-nums;
    font-size: 11px;
    color: var(--text-3);
  }
}

.ledger-head .num,
.ledger-head .time,
.ledger-row .num,
.ledger-row .time {
  text-align: right;
}

/* 首列永远左对齐、末列永远右对齐 —— 两端与容器边缘齐平。
   否则当数值列恰好排在第一位时（代数奖励表的"数量"列），
   右对齐会在左边留下一道很空的沟。 */
.ledger-head > :first-child,
.ledger-row > :first-child {
  text-align: left;
}

.empty-state {
  margin: 0;
  padding: 54px 0;
  text-align: center;
  font-size: 12px;
  color: var(--text-3);
}

.safe-bottom {
  height: 56px;
}
</style>
