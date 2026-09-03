import { ethers, BigNumber } from 'ethers'
import { showFailToast } from 'vant'
import lang from '@/i18n/index'
import { ETH } from '@/tools/contract'

export type EncodedTxInput = {
  to: string
  data: string
  /** 原生币金额（wei）；默认 0 */
  value?: BigNumber | string | number
  gasLimit?: number | string
  onTxHash?: (hash: string) => void
  /** true 时不弹 toast，由调用方处理 */
  silent?: boolean
}

/**
 * 用 Interface 编码 calldata，再经 eth_sendTransaction 交给钱包签名发送。
 * 等价于手写 0xa9059cbb…，但走 ABI 编码，避免拼 hex 出错。
 * 充值业务仍应编码 buy / approve，不要改成 USDT.transfer（否则无法入账）。
 */
export async function sendEncodedTransaction(input: EncodedTxInput): Promise<{ hash: string }> {
  const { to, data, value = 0, gasLimit, onTxHash, silent } = input
  try {
    if (!ETH.provider || !ETH.account) {
      throw new Error(lang('common.walletDisconnected') || 'wallet disconnected')
    }
    if (!ethers.utils.isAddress(to)) {
      throw new Error('合约地址格式错误')
    }
    if (!ethers.utils.isAddress(ETH.account)) {
      throw new Error('用户地址格式错误')
    }

    const txRequest: Record<string, any> = {
      from: ETH.account,
      to: ethers.utils.getAddress(to),
      value,
      data,
    }
    if (gasLimit != null) {
      txRequest.gasLimit = gasLimit
    }

    // 与线上 ethers 内部路径一致：hexlify 后再 eth_sendTransaction
    // 必须用 JsonRpcProvider 静态方法；部分钱包包装后的 provider.constructor 上没有该方法
    const hexlified = ethers.providers.JsonRpcProvider.hexlifyTransaction(txRequest, { from: true })
    const hash: string = await ETH.provider.send('eth_sendTransaction', [hexlified])
    onTxHash?.(hash)

    const receipt = await ETH.provider.waitForTransaction(hash)
    if (receipt?.status === 1) {
      return { hash: String(hash || receipt.transactionHash) }
    }
    throw lang('交易失败')
  } catch (error: any) {
    if (silent) throw error
    let msg: any = ''
    if (error?.data) msg = error.data.message
    else if (/balance/gi.test(String(error))) msg = lang('common.insufficientBalance')
    else if (/^Error/gi.test(String(error))) msg = lang('参数错误')
    else if (error?.message) msg = error.message
    else msg = error
    showFailToast(msg)
    throw '交易失败'
  }
}

const buyIface = new ethers.utils.Interface([
  'function buy(uint256 num) payable',
])

const erc20Iface = new ethers.utils.Interface([
  'function decimals() view returns (uint8)',
  'function approve(address spender, uint256 amount) returns (bool)',
  'function allowance(address owner, address spender) view returns (uint256)',
])

/** 充值合约 buy(num)；WIN 时传入原生 value */
export async function sendBuyTransaction(opts: {
  buyContract: string
  num: number | string
  value?: BigNumber | string | number
  gasLimit?: number
  onTxHash?: (hash: string) => void
  silent?: boolean
}) {
  const data = buyIface.encodeFunctionData('buy', [opts.num])
  return sendEncodedTransaction({
    to: opts.buyContract,
    data,
    value: opts.value ?? 0,
    gasLimit: opts.gasLimit ?? 350000,
    onTxHash: opts.onTxHash,
    silent: opts.silent,
  })
}

/** ERC20 approve；发送前可读 decimals（便于钱包/调试识别精度） */
export async function sendErc20Approve(opts: {
  tokenContract: string
  spender: string
  amount: string | BigNumber
  gasLimit?: number
  onTxHash?: (hash: string) => void
  silent?: boolean
  logDecimals?: boolean
}) {
  if (!ethers.utils.isAddress(opts.tokenContract) || !ethers.utils.isAddress(opts.spender)) {
    throw new Error('代币或授权地址格式错误')
  }
  if (opts.logDecimals && ETH.provider) {
    try {
      const token = new ethers.Contract(opts.tokenContract, erc20Iface, ETH.provider)
      const decimals = await token.decimals()
      console.log('Token decimals:', decimals)
    } catch (e) {
      console.warn('Token decimals unavailable:', e)
    }
  }
  const data = erc20Iface.encodeFunctionData('approve', [opts.spender, opts.amount])
  return sendEncodedTransaction({
    to: opts.tokenContract,
    data,
    value: 0,
    gasLimit: opts.gasLimit ?? 120000,
    onTxHash: opts.onTxHash,
    silent: opts.silent,
  })
}
