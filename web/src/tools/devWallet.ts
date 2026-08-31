/**
 * 仅用于预览 / 本地调试的注入式钱包（EIP-1193）。
 *
 * 当 VITE_DEV_WALLET=true 且当前环境没有真实钱包时，向 window.ethereum 注入一个
 * 由本地私钥驱动的 provider，让登录签名流程可以在没有 MetaMask 的浏览器
 * （v0 预览、CI、无插件的桌面浏览器）里跑通。
 *
 * 能力范围：账号授权、eth_chainId、personal_sign / eth_sign / signTypedData_v4、
 * 切链，以及把其余只读 JSON-RPC 转发给链上节点。**不支持发起链上交易**
 * （没有 gas、没有资产），充值 / 提现这类需要签名交易的按钮会明确报错。
 *
 * 生产构建不要开启该开关。
 */
import { ethers } from 'ethers'

type ChainDef = { chainIdHex: string; rpc: string }

// 开发/预览环境默认开启，只有显式设成 false 才关闭。
// 不依赖 .env.prod —— 该文件被 .gitignore 忽略，同步后可能不存在。
const FLAG = String((import.meta as any).env?.VITE_DEV_WALLET ?? '').toLowerCase()
const ENABLED = FLAG === 'false' ? false : FLAG === 'true' || !!(import.meta as any).env?.DEV
const PK_STORAGE_KEY = 'dev_wallet_pk'

/** 优先用 env 里指定的私钥；否则生成一个并持久化，保证刷新后地址不变。 */
function resolvePrivateKey(): string {
  const fromEnv = (import.meta as any).env?.VITE_DEV_WALLET_PK
  if (fromEnv) return String(fromEnv).trim()

  const cached = localStorage.getItem(PK_STORAGE_KEY)
  if (cached) return cached

  const generated = ethers.Wallet.createRandom().privateKey
  localStorage.setItem(PK_STORAGE_KEY, generated)
  return generated
}

function install() {
  const env = (import.meta as any).env ?? {}
  const toHex = (n: number) => `0x${n.toString(16)}`

  const eoeoId = Number(env.VITE_CHAINID || 86233268)
  const bscId = Number(env.VITE_BSC_CHAINID || 56)

  // 走 vite 代理，避免直连公共 RPC 的 CORS 问题
  const chains = new Map<string, ChainDef>([
    [toHex(eoeoId), { chainIdHex: toHex(eoeoId), rpc: '/dev-rpc/eoeo' }],
    [toHex(bscId), { chainIdHex: toHex(bscId), rpc: '/dev-rpc/bsc' }]
  ])

  const wallet = new ethers.Wallet(resolvePrivateKey())
  let current = toHex(eoeoId)
  let rpcId = 0

  const listeners: Record<string, Function[]> = {}
  const emit = (event: string, payload?: any) => {
    ;(listeners[event] || []).forEach((fn) => {
      try {
        fn(payload)
      } catch {
        /* 忽略监听器自身的异常 */
      }
    })
  }

  /** 把未拦截的方法转发给当前链的节点 */
  const forward = async (method: string, params: any[] = []) => {
    const chain = chains.get(current)
    if (!chain?.rpc) throw new Error(`dev wallet: 链 ${current} 没有可用 RPC`)

    const res = await fetch(chain.rpc, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ jsonrpc: '2.0', id: ++rpcId, method, params })
    })
    const body = await res.json()
    if (body.error) {
      throw Object.assign(new Error(body.error.message), { code: body.error.code })
    }
    return body.result
  }

  const isAddressLike = (v: any) => typeof v === 'string' && /^0x[0-9a-fA-F]{40}$/.test(v)

  /** ethers 传 [data, address]，部分库反过来传 [address, data]，这里都兼容 */
  const signPersonal = async (params: any[]) => {
    const [first, second] = params
    const data = isAddressLike(first) && !isAddressLike(second) ? second : first
    const bytes =
      typeof data === 'string' && ethers.utils.isHexString(data)
        ? ethers.utils.arrayify(data)
        : ethers.utils.toUtf8Bytes(String(data))
    return wallet.signMessage(bytes)
  }

  const provider: any = {
    isMetaMask: true,
    isDevWallet: true,
    get chainId() {
      return current
    },
    get selectedAddress() {
      return wallet.address
    },
    async request({ method, params = [] }: { method: string; params?: any[] }) {
      switch (method) {
        case 'eth_requestAccounts':
        case 'eth_accounts':
          return [wallet.address]

        case 'eth_chainId':
          return current

        case 'net_version':
          return String(parseInt(current, 16))

        case 'personal_sign':
          return signPersonal(params)

        case 'eth_sign':
          return signPersonal([params[1], params[0]])

        case 'eth_signTypedData_v4': {
          const raw = isAddressLike(params[0]) ? params[1] : params[0]
          const payload = typeof raw === 'string' ? JSON.parse(raw) : raw
          const types = { ...(payload.types || {}) }
          delete types.EIP712Domain
          return (wallet as any)._signTypedData(payload.domain, types, payload.message)
        }

        case 'wallet_switchEthereumChain': {
          const target = params?.[0]?.chainId
          if (!chains.has(target)) {
            throw Object.assign(new Error(`Unrecognized chain ID ${target}`), { code: 4902 })
          }
          current = target
          emit('chainChanged', current)
          return null
        }

        case 'wallet_addEthereumChain': {
          const def = params?.[0]
          if (def?.chainId) {
            chains.set(def.chainId, { chainIdHex: def.chainId, rpc: def.rpcUrls?.[0] || '' })
            current = def.chainId
            emit('chainChanged', current)
          }
          return null
        }

        case 'eth_sendTransaction':
          throw new Error(
            'dev wallet 不支持发起链上交易（无 gas、无资产）。仅可用于登录签名与只读查询。'
          )

        default:
          return forward(method, params)
      }
    },
    enable() {
      return provider.request({ method: 'eth_requestAccounts' })
    },
    on(event: string, cb: Function) {
      ;(listeners[event] = listeners[event] || []).push(cb)
      return provider
    },
    removeListener(event: string, cb: Function) {
      listeners[event] = (listeners[event] || []).filter((fn) => fn !== cb)
      return provider
    }
  }

  // 强制覆盖 window.ethereum。
  // 关键：真实钱包插件（MetaMask / OKX / Rabby 等）会先注入自己的 provider，
  // 而它并没有连到 EOEO 链（chainId 86233268），会直接抛
  // "The Provider is not connected to the requested chain."。
  // 预览环境下必须让注入钱包胜出，所以这里覆盖而不是让位。
  // 插件常把 window.ethereum 定义成不可写的 getter，先试 defineProperty，
  // 失败再退回直接赋值。
  const win = window as any
  try {
    Object.defineProperty(win, 'ethereum', {
      configurable: true,
      get: () => provider,
      set: () => {
        /* 忽略插件后续的再次注入，保持注入钱包生效 */
      }
    })
  } catch {
    win.ethereum = provider
  }

  // detect-provider 在多插件共存时会读 providers 数组，这里一并覆盖
  win.ethereum.providers = [provider]
  win.web3 = { currentProvider: provider }

  window.dispatchEvent(new Event('ethereum#initialized'))
  console.log('[v0] dev wallet 已注入，地址:', wallet.address)
}

if (ENABLED && typeof window !== 'undefined') {
  install()
}
