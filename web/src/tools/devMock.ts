/**
 * 预览用离线 Mock（纯附加，不改动任何业务代码）
 *
 * 由 VITE_DEV_MOCK 控制，仅在开发预览下启用。原理：在 request.ts / api/aix.ts
 * 创建各自的 axios 实例之前，替换 axios.defaults.adapter。两个实例都通过
 * axios.create() 继承 defaults，因此一处拦截即可覆盖全部请求。
 *
 * 关键点：这里连 /v1/auth/login 一起 mock 并返回 token，
 * 所以 person.ts 里原有的「签名 -> 登录 -> loginSuccess」流程会自己走完，
 * 无需邀请码，也无需修改 person.ts / request.ts / backendAdapter.ts。
 *
 * 关闭方式：把 .env.prod 里的 VITE_DEV_MOCK 改成 false（或删掉该行）。
 */
import axios from 'axios'

// 开发/预览环境默认开启，只有显式设成 false 才关闭。
// 不依赖 .env.prod —— 该文件被 .gitignore 忽略，同步后可能不存在。
const FLAG = String((import.meta as any).env?.VITE_DEV_MOCK ?? '').toLowerCase()
const ENABLED = FLAG === 'false' ? false : FLAG === 'true' || !!(import.meta as any).env?.DEV

const DAY = 86400
const nowSec = () => Math.floor(Date.now() / 1000)

function addr(): string {
  return (
    localStorage.getItem('account') ||
    localStorage.getItem('devWalletAddress') ||
    '0xA6C8AE22b3C7Ea0a87699763017FdbfEa283CAfE'
  )
}

/** 以当天为基准往前推 n 天的 YYYY-MM-DD */
function dateBack(n: number): string {
  const d = new Date(Date.now() - n * DAY * 1000)
  const pad = (v: number) => String(v).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/* ---------------------------------- 数据 ---------------------------------- */

const ORDERS = [
  { id: 1001, amount: '1000.0000', exit_multiplier: 3, exit_amount: '3000.0000', released_amount: '420.0000', status: 'active', pay_from: 'recharge', product_name: '节点认购', created_at: nowSec() - 26 * DAY },
  { id: 1002, amount: '500.0000', exit_multiplier: 3, exit_amount: '1500.0000', released_amount: '150.0000', status: 'active', pay_from: 'win', product_name: '节点认购', created_at: nowSec() - 14 * DAY },
  { id: 1003, amount: '3000.0000', exit_multiplier: 3, exit_amount: '9000.0000', released_amount: '9000.0000', status: 'completed', pay_from: 'recharge', product_name: '节点认购', created_at: nowSec() - 90 * DAY },
]

// 近 12 天静态释放
const RELEASES = Array.from({ length: 12 }).map((_, i) => ({
  id: 5000 + i,
  order_id: 1001,
  settlement_date: dateBack(i),
  amount: '10.0000',
  money: '30.0000',
  exit_multiplier: 3,
  referral_distributed: i % 2 === 0 ? '4.5000' : '0',
  created_at: nowSec() - i * DAY,
}))

const REFERRALS = Array.from({ length: 8 }).map((_, i) => ({
  id: 6000 + i,
  settlement_date: dateBack(i),
  generation: (i % 3) + 1,
  reward_amount: (3 - (i % 3)).toFixed(4),
  from_address: '0x36fEa8A26AaD9Be34B29383D46FEaB42332389e6',
  created_at: nowSec() - i * DAY,
}))

const ECOS = Array.from({ length: 6 }).map((_, i) => ({
  id: 7000 + i,
  settlement_date: dateBack(i),
  community_level: `V${(i % 3) + 1}`,
  community_stake: '12000.0000',
  base_amount: '12000.0000',
  base_rate: '0.02',
  base_reward: '24.0000',
  equal_reward: '6.0000',
  total_reward: '30.0000',
  created_at: nowSec() - i * DAY,
}))

const INVITEES = [
  { address: '0x36fEa8A26AaD9Be34B29383D46FEaB42332389e6', team_stake: '8000.0000', exit_amount: '9000.0000', community_level: 'V3', direct_count: 4, created_at: nowSec() - 40 * DAY },
  { address: '0x7B1f9C3D5E2a4B6c8D0e1F2a3B4c5D6e7F8a9B0c', team_stake: '4200.0000', exit_amount: '4500.0000', community_level: 'V2', direct_count: 2, created_at: nowSec() - 30 * DAY },
  { address: '0x1A2b3C4d5E6f7A8b9C0d1E2f3A4b5C6d7E8f9A0b', team_stake: '1500.0000', exit_amount: '1500.0000', community_level: 'V1', direct_count: 1, created_at: nowSec() - 20 * DAY },
  { address: '0x9F8e7D6c5B4a3928170615243342516607788990', team_stake: '600.0000', exit_amount: '600.0000', community_level: '0', direct_count: 0, created_at: nowSec() - 10 * DAY },
]

const PROFILE = () => ({
  address: addr(),
  usdt_recharge: '1860.5000',
  usdt_reward: '742.3000',
  usdt_withdrawable: '742.3000',
  aix_balance: '3250.0000',
  win_balance: '820.0000',
  win_recharge_balance: '300.0000',
  is_zero_account: false,
  is_community_subsidy: true,
  aix_price: '0.85',
  win_price: 0.12,
  min_win_recharge: '10',
  min_usdt_recharge: '10',
  aix_to_win_rate: 7.0833,
  exchange_fee_rate: 0.05,
  static_usdt_total: '570.0000',
  pending_amount: '4500.0000',
  unexited_amount: '3930.0000',
  total_nodes: 2,
  mgmt_level: 3,
  small_area_perf: '4200.0000',
  team_perf: '14300.0000',
  next_release_at: nowSec() + 3600 * 5,
  server_time: nowSec(),
  aix_contract: '0x314D550572a0fA001B465a9EBc1dd04D834a0688',
  sdt_contract: '0x314D550572a0fA001B465a9EBc1dd04D834a0688',
  win_contract: '0x94db6bb040107ef9a2F1e9DB9d84dD8D6D98997e',
  usdt_contract: '0x926632975149221891f1b9B56Efd125Dfe90ba2f',
  points: '1280',
  points_all: '3400',
  overflow_reward: '96.0000',
  pending_mgmt_reward: '48.0000',
  mgmt_reward_total: '312.0000',
  withdrawRate: 0.06,
  withdrawMin: 10,
})

const ANNOUNCEMENTS = [
  { id: 3, title: 'AIX 生态节点认购规则更新', content: '<p>自本月起，节点认购出局倍数统一调整为 3 倍，静态释放按日结算。</p>', add_time: nowSec() - 2 * DAY, created_at: dateBack(2) },
  { id: 2, title: '关于 WIN 充值到账延迟的说明', content: '<p>近期链上拥堵，WIN 充值确认可能延迟 10-30 分钟，请耐心等待。</p>', add_time: nowSec() - 6 * DAY, created_at: dateBack(6) },
  { id: 1, title: 'AIX 积分提现功能上线', content: '<p>积分可按 1:1 提现为 SDT 代币，单笔最低 10。</p>', add_time: nowSec() - 12 * DAY, created_at: dateBack(12) },
]

/* --------------------------------- 路由表 --------------------------------- */

type Handler = (cfg: any) => any
const routes: Record<string, Handler> = {
  '/v1/auth/challenge': (cfg) => ({
    address: cfg?.params?.address || addr(),
    message: `AIX login\naddress: ${cfg?.params?.address || addr()}\nnonce: devmock-${Date.now()}`,
    expire_at: nowSec() + 600,
  }),
  // 直接放行：返回 token，让原有登录流程自己走完（无需邀请码）
  '/v1/auth/login': () => ({
    token: `devmock.${btoa(addr()).replace(/=+$/, '')}.${Date.now()}`,
    address: addr(),
    is_new_user: false,
  }),
  '/v1/auth/profile': () => ({ address: addr(), invite_code: addr(), inviter_address: '0x36fEa8A26AaD9Be34B29383D46FEaB42332389e6', level: 3, created_at: nowSec() - 120 * DAY }),
  '/v1/auth/invitees': () => ({ invitees: INVITEES, list: INVITEES, count: INVITEES.length }),

  '/v1/wallet/aix-profile': () => PROFILE(),
  '/v1/wallet/balance': () => ({
    address: addr(),
    balance: '1860.5000',
    released_balance: '742.3000',
    claimable_amount: '3250.0000',
    claimed_amount: '3250.0000',
    static_usdt_total: '570.0000',
    pending_amount: '4500.0000',
    unexited_amount: '3930.0000',
    total_nodes: 2,
    next_release_at: nowSec() + 3600 * 5,
    server_time: nowSec(),
  }),
  '/v1/wallet/orders': () => ({ orders: ORDERS, count: ORDERS.length }),
  '/v1/wallet/releases': () => ({ releases: RELEASES, list: RELEASES, count: RELEASES.length }),
  '/v1/wallet/referral-rewards': () => ({ rewards: REFERRALS, list: REFERRALS, count: REFERRALS.length }),
  '/v1/wallet/management-rewards': () => ({ rewards: ECOS, list: ECOS, count: ECOS.length }),
  '/v1/wallet/points-records': () => ({
    points: '1280',
    points_all: '3400',
    count: 2,
    records: [
      { id: 1, order_id: 1001, points: '800', principal: '1000.0000', fund_source: 'recharge', status: 'completed', created_at: nowSec() - 26 * DAY },
      { id: 2, order_id: 1002, points: '480', principal: '500.0000', fund_source: 'win', status: 'completed', created_at: nowSec() - 14 * DAY },
    ],
  }),
  '/v1/wallet/exchange-records': () => ({
    records: [
      { id: 1, from_asset: 'AIX', from_amount: '100.0000', to_asset: 'USDT', to_amount: '80.7500', fee_amount: '4.2500', fee_rate: '0.05', exchange_price: '0.8500', status: 'completed', remark: '', created_at: nowSec() - 3 * DAY },
    ],
  }),
  '/v1/wallet/withdraw-records': () => ({
    records: [
      { id: 1, asset: 'WIN', amount: '200.0000', fee: '12.0000', net_amount: '188.0000', to_address: addr(), status: 'completed', tx_hash: '0xdev1', remark: '', created_at: nowSec() - 5 * DAY, updated_at: nowSec() - 5 * DAY },
      { id: 2, asset: 'SDT', amount: '100.0000', fee: '6.0000', net_amount: '94.0000', to_address: addr(), status: 'pending', tx_hash: '', remark: '', created_at: nowSec() - DAY, updated_at: nowSec() - DAY },
    ],
  }),
  '/v1/wallet/withdrawals': () => ({ records: [], list: [], count: 0 }),
  '/v1/wallet/recharges': () => ({ recharges: [{ id: 1, amount: '1000.0000', asset: 'USDT', status: 'completed', tx_hash: '0xdevusdt', created_at: nowSec() - 26 * DAY }], count: 1 }),
  '/v1/wallet/recharges-win': () => ({ recharges: [{ id: 2, amount: '300.0000', asset: 'WIN', status: 'completed', tx_hash: '0xdevwin', created_at: nowSec() - 14 * DAY }], count: 1 }),
  '/v1/wallet/downline-usdt-recharges': () => ({ recharges: INVITEES.map((v, i) => ({ id: 10 + i, address: v.address, amount: v.team_stake, status: 'completed', created_at: v.created_at })), count: INVITEES.length }),
  '/v1/wallet/transfer-records/lineal': () => ({ records: [], count: 0 }),

  '/v1/announcements': () => ({ list: ANNOUNCEMENTS, count: ANNOUNCEMENTS.length, page: 1 }),
  '/v1/announcement/detail': (cfg) => {
    const id = Number(cfg?.params?.id)
    return ANNOUNCEMENTS.find((a) => a.id === id) || ANNOUNCEMENTS[0]
  },
}

/** 写操作：统一返回成功，避��预览中点按钮报错 */
const writeOk: Record<string, Handler> = {
  '/v1/wallet/subscribe-aix': (cfg) => ({ status: 'ok', order_id: 9001, amount: cfg?.data?.amount, message: '认购成功（预览模拟）' }),
  '/v1/wallet/exchange-aix-to-win': (cfg) => {
    const amount = Number(JSON.parse(cfg?.data || '{}')?.aix_amount || 0)
    return { record_id: 9002, from_asset: 'AIX', from_amount: amount.toFixed(4), to_asset: 'USDT', to_amount: (amount * 0.85 * 0.95).toFixed(4), exchange_price: '0.8500', exchange_fee_rate: 0.05, status: 'completed', aix_balance: '3150.0000', usdt_withdrawable: '823.0500', created_at: nowSec() }
  },
  '/v1/wallet/withdraw-sdt': () => ({ withdraw_id: 9004, asset: 'SDT', amount: '100.0000', to_address: addr(), status: 'pending', tx_hash: '' }),
  '/v1/wallet/withdraw-usdt': () => ({ withdraw_id: 9005, asset: 'USDT', amount: '100.0000', to_address: addr(), status: 'pending', tx_hash: '' }),
  '/v1/wallet/claim': () => ({ status: 'ok', claimed: '3250.0000' }),
  '/v1/wallet/recharge': () => ({ status: 'ok', order_id: 9006 }),
  '/v1/wallet/recharge/confirm': () => ({ status: 'ok' }),
  '/v1/wallet/recharge-win': () => ({ status: 'ok', order_id: 9007 }),
  '/v1/wallet/recharge-win/confirm': () => ({ status: 'ok' }),
  '/v1/wallet/transfer': () => ({ status: 'ok' }),
}

/* --------------------------------- 安装 --------------------------------- */

function pathOf(url: string): string {
  const clean = String(url || '').split('?')[0]
  const idx = clean.indexOf('/v1/')
  return idx >= 0 ? clean.slice(idx) : clean
}

if (ENABLED) {
  const realAdapter = (axios as any).getAdapter(['xhr', 'fetch', 'http'])

  ;(axios.defaults as any).adapter = async (config: any) => {
    const path = pathOf(config?.url || '')
    const handler = routes[path] || writeOk[path]

    if (!handler) return realAdapter(config)

    // 模拟少量网络延迟，让 loading 状态可见
    await new Promise((r) => setTimeout(r, 90))
    const data = handler(config)
    console.log('[v0] mock', config?.method?.toUpperCase(), path)
    return {
      data,
      status: 200,
      statusText: 'OK',
      headers: { 'content-type': 'application/json' },
      config,
      request: {},
    }
  }

  console.log('[v0] dev mock 已启用，/v1/* 请求走本地模拟数据')
}

export {}
