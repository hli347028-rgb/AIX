import { defineStore } from "pinia";
import { ETH } from "@/tools/contract";
import request from "@/tools/request";
import lang from '@/i18n/index'
import fetchSign from './fetchSign'
import { getAixBalance, getAixProfile } from '@/api/aix'
import { showToast, showDialog, setToastDefaultOptions } from "vant";

setToastDefaultOptions({
  zIndex: 6001
})

let timeSwitch: any = null//定时获取用户信息
let initializationPromise: Promise<void> | null = null
let listenedProvider: any = null
let accountsChangedHandler: ((accounts: string[]) => void) | null = null
let chainChangedHandler: (() => void) | null = null
let suppressWalletEvents = false
let pendingWalletReinitialize = false
let walletEventVersion = 0

const INVITE_CANCELLED = 'INVITE_CANCELLED'
const WALLET_DISCONNECTED = 'WALLET_DISCONNECTED'

function clearAuthStorage() {
  localStorage.removeItem('token')
  localStorage.removeItem('account')
  localStorage.removeItem('sign')
}

function clearUserTimer() {
  clearInterval(timeSwitch)
  timeSwitch = null
}

function clearAnnouncementReadsForAccount(account: string) {
  const accountSuffix = `:${account.toLowerCase()}`
  const clearStorage = (storage: Storage) => {
    for (let index = storage.length - 1; index >= 0; index -= 1) {
      const key = storage.key(index)
      if (key?.startsWith('aix-announcement-read:') && key.endsWith(accountSuffix)) {
        storage.removeItem(key)
      }
    }
  }
  clearStorage(localStorage)
  clearStorage(sessionStorage)
}

function escapeHtml(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

function readInviteCode() {
  const current = new URL(window.location.href)
  const queryCode = current.searchParams.get('code')
    || [...current.searchParams.entries()].find(([key]) => key.toLowerCase() === 'invitecode')?.[1]
    || ''
  let decodedUrl = window.location.href
  try {
    decodedUrl = decodeURIComponent(decodedUrl)
  } catch {
    // URL 中的其他无效转义不应阻断钱包登录。
  }
  const legacyCode = decodedUrl.match(/-invitetdh-(.*?)-invitetdh-/i)?.[1] || ''
  return (queryCode || legacyCode).trim()
}

function removeInviteFromAddressBar() {
  const current = new URL(window.location.href)
  for (const key of [...current.searchParams.keys()]) {
    if (key.toLowerCase() === 'code' || key.toLowerCase() === 'invitecode') {
      current.searchParams.delete(key)
    }
  }
  current.pathname = current.pathname.replace(/-invitetdh-.*?-invitetdh-/gi, '')
  current.hash = current.hash.replace(/-invitetdh-.*?-invitetdh-/gi, '')
  window.history.replaceState(window.history.state, '', current.toString())
}

export default defineStore('person', {
  state: () => ({
    loadAccount: false,
    isLogin: false,
    userinfo: {
      status: 'ok',
      level: '0',
      locationNum: '0',
      total: '0',
      max: '0',
      min: '0',
      inviteUserAddress: '0x36fEa8A26AaD9Be34B29383D46FEaB42332389e6',
      buy: '0.00',
      amountGetSub: '0.00',
      amountGet: '0.0000',
      unexitedAmount: '0.00',
      outNum: '0',
      location: '0.00',
      recommend: '0.00',
      recommendNum: 0,
      recommendTeamNum: 0,
      recommendTwo: '0.00',
      team: '0.00',
      overflowReward: '0.00',
      overflow_reward: '0.00',
      communityLevel: '0',
      community_level: '0',
      is_zero_account: false,
      isZeroAccount: false,
      is_community_subsidy: false,
      isCommunitySubsidy: false,
      community_subsidy_rate: 0,
      communitySubsidyRate: 0,
      zero_account_reward_total: '0',
      zeroAccountRewardTotal: '0',
      community_subsidy_total: '0',
      communitySubsidyTotal: '0',
      points: '0',
      points_all: '0',
      all: '0.00',
      usdt: '0.00',
      reward: '0.00',
      aix: '0',
      win: '0',
      withdrawRate: 0.06,
      withdrawMin: 10,
      raw: '0.00',
      withdrawRateTwo: 0,
      withdrawMinTwo: 0,
      notice: '',
      goods: [],
      one: '', // 国家
      two: '', // 省份
      three: '', // 城市
      four: '', // 区域
      five: '', // 详情
      six: '', // 收件人手机
      seven: '' // 收件人
    },
    profile: {
      address: '',
      username: '',
      usdt_recharge: '0',
      usdt_reward: '0',
      aix_balance: '0',
      win_balance: '0',
      win_recharge_balance: '0',
      usdt_withdrawable: '0',
      is_zero_account: false,
      is_community_subsidy: false,
      community_subsidy_rate: 0,
      aix_price: '0',
      win_price: 0,
      min_win_recharge: '10',
      win_a_recharge_balance: '0',
      win_a_price: 0,
      min_win_a_recharge: '10',
      min_usdt_recharge: '10',
      aix_to_win_rate: 0,
      exchange_fee_rate: 0.05,
      static_usdt_total: '0',
      pending_amount: '0',
      unexited_amount: '0',
      total_nodes: 0,
      mgmt_level: 0,
      small_area_perf: '0',
      team_perf: '0',
      next_release_at: 0,
      server_time: 0,
      aix_contract: '',
      points: '0',
      points_all: '0',
      overflow_reward: '0',
      pending_mgmt_reward: '0',
    } as Record<string, any>,
    urlCode: '',
    sign: '',
    address: '',
    inviteUserAddress: '',
    isOpened: false,
    authError: '',
    authStage: 'idle' as 'idle' | 'connecting' | 'authenticating' | 'verifying' | 'error',
    isInitializing: false
  }),
  actions: {
    // 所有入口共享同一个初始化 Promise，避免 401、钱包事件和页面挂载重复登录。
    init(): Promise<void> {
      if (initializationPromise) return initializationPromise
      this.isInitializing = true
      this.isLogin = false
      this.authError = ''
      this.loadAccount = false
      initializationPromise = this.initializeAuth().catch((error: any) => {
        const message = error?.response?.data?.message || error?.message || String(error || '')
        this.authError = this.mapAuthError(message, error)
        this.authStage = 'error'
        this.loadAccount = false
        this.isLogin = false
        throw error
      }).finally(() => {
        this.isInitializing = false
        initializationPromise = null
        if (pendingWalletReinitialize) {
          pendingWalletReinitialize = false
          window.setTimeout(() => {
            void this.retryAuth().catch(() => undefined)
          }, 0)
        }
      })
      return initializationPromise
    },
    async initializeAuth() {
      this.urlCode = readInviteCode()
      this.authStage = 'connecting'
      suppressWalletEvents = true
      let account = ''
      try {
        account = await ETH.getAccount()
      } finally {
        suppressWalletEvents = false
      }
      this.address = account
      this.bindWalletEvents()
      const startWalletVersion = walletEventVersion
      const storedAccount = localStorage.getItem('account')
      const storedToken = localStorage.getItem('token')
      const sameAccount = Boolean(
        storedAccount && storedToken && storedAccount.toLowerCase() === account.toLowerCase()
      )

      if (sameAccount) {
        this.authStage = 'verifying'
        try {
          await this.loginSuccess()
          if (startWalletVersion !== walletEventVersion) {
            this.clearAuthentication(true, true)
            throw new Error(WALLET_DISCONNECTED)
          }
          return
        } catch (error: any) {
          const message = error?.response?.data?.message || error?.message || ''
          const reason = error?.response?.data?.reason || ''
          if (reason === 'ACCOUNT_FROZEN' || /账户已被冻结|ACCOUNT_FROZEN/i.test(message)) {
            throw error
          }
          if (!/登录过期|未登录|unauthorized|token|请先登录/i.test(message)) {
            throw error
          }
          this.clearAuthentication(false)
        }
      } else {
        this.clearAuthentication(false)
      }

      this.authStage = 'authenticating'
      let sign = await this.getFreshSign()
      let token = ''
      try {
        token = await this.requestLogin(sign, this.urlCode)
      } catch (error: any) {
        if (/签名挑战|challenge/i.test(error?.message || '')) {
          sign = await this.getFreshSign()
          token = await this.requestLogin(sign, this.urlCode)
        } else if (this.isInviteError(error?.message)) {
          const code = await this.inputInvitationCode()
          token = await this.requestLogin(sign, code)
        } else {
          throw error
        }
      }
      if (startWalletVersion !== walletEventVersion) {
        throw new Error(lang('common.walletDisconnected'))
      }
      this.authStage = 'verifying'
      await this.loginSuccess(token)
      if (startWalletVersion !== walletEventVersion) {
        this.clearAuthentication(true, true)
        throw new Error(WALLET_DISCONNECTED)
      }
    },
    async requestLogin(sign: string, code: string): Promise<string> {
      const res: any = await request.post('app_server/eth_authorize', {
        address: ETH.account,
        code,
        sign,
        noMsg: true,
      })
      if (
        res.status === '用户已锁定'
        || res.status === '请输入推荐码'
        || res.status === '无效的推荐码'
        || res.status === '该用户未激活'
      ) {
        throw new Error(res.status)
      }
      if (res.status !== 'ok' || !res.token) {
        throw new Error(res.message || res.status || lang('common.operationFailed'))
      }
      return res.token
    },
    async getFreshSign() {
      const sign = await fetchSign()
      this.sign = sign
      return sign
    },
    isInviteError(message?: string) {
      return message === '请输入推荐码' || message === '无效的推荐码' || message === '该用户未激活'
    },
    mapAuthError(message?: string, error?: any) {
      if (message === INVITE_CANCELLED) return lang('common.inviteCancelled')
      if (message === WALLET_DISCONNECTED) return lang('common.walletDisconnected')
      if (error?.code === 4001 || /user rejected|用户拒绝|拒绝签名/i.test(String(message || ''))) {
        return lang('common.authRejected')
      }
      if (/账户已被冻结|ACCOUNT_FROZEN/i.test(String(message || '')) || message === '用户已锁定') {
        return lang('common.userLocked')
      }
      if (message === '该用户未激活') return lang('common.inviterNotActivated')
      if (message === '无效的推荐码') return lang('common.inviteCodeMustBeRegisteredWallet')
      if (message === '请输入推荐码') return lang('common.enterInviteCode')
      return message || lang('common.operationFailed')
    },
    async inputInvitationCode(): Promise<string> {
      const genesis = (import.meta as any).env?.VITE_GENESIS_ADDRESS || ''
      const defaultCode = this.urlCode || genesis
      return new Promise((resolve, reject) => {
        let settled = false
        showDialog({
          className: 'invite-code-dialog',
          title: lang('common.inviteCode'),
          message: `
            <input
              id="inviteInput"
              type="text"
              value="${escapeHtml(defaultCode)}"
              placeholder="${escapeHtml(lang('common.registeredWalletAddress'))}"
            />
          `,
          confirmButtonText: lang('common.confirm'),
          cancelButtonText: lang('common.close'),
          showCancelButton: true,
          allowHtml: true,
          zIndex: 6000,
          beforeClose(action: any) {
            if (action === 'confirm') {
              const input = document.getElementById('inviteInput') as HTMLInputElement | null
              const value = input?.value.trim() || ''
              if (!value) {
                showToast({ message: lang('common.enterInviteCode'), type: 'fail' })
                return false
              }
              settled = true
              resolve(value)
            } else if (!settled) {
              settled = true
              reject(new Error(INVITE_CANCELLED))
            }
            return true
          }
        } as any).catch(() => undefined)
      })
    },
    bindWalletEvents() {
      const provider = ETH.getRawProvider()
      if (!provider?.on || listenedProvider === provider) return
      if (listenedProvider?.removeListener) {
        if (accountsChangedHandler) listenedProvider.removeListener('accountsChanged', accountsChangedHandler)
        if (chainChangedHandler) listenedProvider.removeListener('chainChanged', chainChangedHandler)
      }
      accountsChangedHandler = (accounts: string[]) => {
        const nextAccount = accounts?.[0] || ''
        if (!nextAccount) {
          walletEventVersion += 1
          pendingWalletReinitialize = false
          this.clearAuthentication(true, true)
          ETH.resetConnection()
          this.authError = lang('common.walletDisconnected')
          this.authStage = 'error'
          return
        }
        if (nextAccount.toLowerCase() === String(this.address || ETH.account).toLowerCase()) return
        walletEventVersion += 1
        clearAnnouncementReadsForAccount(nextAccount)
        if (this.isInitializing || suppressWalletEvents) {
          pendingWalletReinitialize = true
          return
        }
        this.clearAuthentication(true, true)
        ETH.resetConnection()
        void this.init().catch(() => undefined)
      }
      chainChangedHandler = () => {
        if (suppressWalletEvents) return
        walletEventVersion += 1
        if (this.isInitializing) {
          pendingWalletReinitialize = true
          return
        }
        this.clearAuthentication(false)
        ETH.resetConnection()
        void this.init().catch(() => undefined)
      }
      provider.on('accountsChanged', accountsChangedHandler)
      provider.on('chainChanged', chainChangedHandler)
      listenedProvider = provider
    },
    retryAuth() {
      this.authError = ''
      this.authStage = 'idle'
      return this.init()
    },
    /* 获取用户信息 */
    async getUser() {
      const getData = async () => {
        let res: any = await request.get('app_server/user_info')
        this.userinfo = { ...this.userinfo, ...res }
      }
      clearUserTimer()
      await getData()
      timeSwitch = setInterval(() => {
        void getData().catch((error) => console.error('[getUser:poll]', error))
      }, 30000)
    },
    async refreshProfile() {
      try {
        const res: any = await getAixProfile()
        this.profile = { ...this.profile, ...res }
        const overflow = res?.overflow_reward ?? res?.overflowReward ?? res?.pending_mgmt_reward
        const mgmtTotal = res?.mgmt_reward_total ?? res?.mgmtRewardTotal
        const directTotal = res?.direct_reward_total ?? res?.directRewardTotal
        const updates: Record<string, any> = {}
        if (overflow != null && overflow !== '') {
          updates.overflowReward = String(overflow)
          updates.overflow_reward = String(overflow)
        }
        if (mgmtTotal != null && mgmtTotal !== '') {
          updates.team = String(mgmtTotal)
          updates.mgmt_reward_total = String(mgmtTotal)
        }
        if (directTotal != null && directTotal !== '') {
          updates.recommend = String(directTotal)
          updates.direct_reward_total = String(directTotal)
        }
        if (Object.keys(updates).length > 0) {
          this.userinfo = { ...this.userinfo, ...updates }
        }
        return this.profile
      } catch (profileError) {
        try {
          const balance: any = await getAixBalance()
          this.profile = {
            ...this.profile,
            address: balance.address || this.address,
            usdt_recharge: balance.balance ?? this.userinfo.usdt ?? this.profile.usdt_recharge,
            usdt_reward: balance.released_balance ?? (this.userinfo as any).reward ?? this.profile.usdt_reward,
            aix_balance: balance.claimed_amount ?? balance.claimable_amount ?? this.profile.aix_balance,
            static_usdt_total: balance.static_usdt_total ?? this.profile.static_usdt_total,
            pending_amount: balance.pending_amount ?? this.profile.pending_amount,
            unexited_amount: balance.unexited_amount ?? this.profile.unexited_amount,
            total_nodes: balance.total_nodes ?? this.profile.total_nodes,
            next_release_at: balance.next_release_at ?? this.profile.next_release_at,
            server_time: balance.server_time ?? this.profile.server_time
          }
          return this.profile
        } catch (balanceError) {
          console.error('[refreshProfile]', profileError, balanceError)
          throw balanceError
        }
      }
    },
    /* 登录成功 */
    async loginSuccess(token?: string) {
      if (token) {
        localStorage.setItem('token', token)
        localStorage.setItem('account', ETH.account)
      }
      await this.getUser()
      try {
        await this.refreshProfile()
      } catch (error) {
        // AIX profile 是增强数据，失败时保留 getUser 的基础数据并继续登录。
        console.error('[loginSuccess:refreshProfile]', error)
      }
      this.isLogin = true
      this.loadAccount = true
      this.authError = ''
      this.authStage = 'verifying'
      removeInviteFromAddressBar()
      this.urlCode = ''
    },
    clearAuthentication(clearAddress = false, clearUserData = false) {
      const currentAddress = this.address
      const initializing = this.isInitializing
      clearAuthStorage()
      clearUserTimer()
      if (clearUserData) {
        // 切换账户时必须清除上一账户的余额和团队数据，避免短暂串号展示。
        this.$reset()
        this.isInitializing = initializing
        if (!clearAddress) this.address = currentAddress
      }
      this.isLogin = false
      this.loadAccount = false
      this.sign = ''
      if (clearAddress) this.address = ''
    },
    /* 自动退出系统 */
    outLogin(reinitialize = true) {
      this.clearAuthentication(false, true)
      if (reinitialize && !this.isInitializing) {
        void this.init().catch(() => undefined)
      }
    },
    handleUnauthorized() {
      if (this.isInitializing) return
      this.outLogin(true)
    }
  }
})
