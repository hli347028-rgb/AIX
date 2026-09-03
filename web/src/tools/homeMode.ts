export type HomeMode = 'static' | 'cinematic'

const STORAGE_KEY = 'aix-home-mode'

function readQueryParam(name: string): string | null {
  try {
    const fromSearch = new URLSearchParams(window.location.search).get(name)
    if (fromSearch) return fromSearch
    const hash = window.location.hash || ''
    const queryIndex = hash.indexOf('?')
    if (queryIndex < 0) return null
    return new URLSearchParams(hash.slice(queryIndex + 1)).get(name)
  } catch {
    return null
  }
}

function parseForcedMode(): HomeMode | 'auto' | null {
  const raw = (readQueryParam('home') || '').trim().toLowerCase()
  if (raw === 'static' || raw === 'lite') return 'static'
  if (raw === 'cinematic' || raw === 'full') return 'cinematic'
  if (raw === 'auto' || raw === 'reset') return 'auto'
  return null
}

function readStoredMode(): HomeMode | null {
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY)
    if (stored === 'static' || stored === 'cinematic') return stored
  } catch {
    /* private mode / WebView storage blocked */
  }
  return null
}

function writeStoredMode(mode: HomeMode | null) {
  try {
    if (!mode) {
      window.localStorage.removeItem(STORAGE_KEY)
      return
    }
    window.localStorage.setItem(STORAGE_KEY, mode)
  } catch {
    /* ignore */
  }
}

function prefersReducedMotion(): boolean {
  try {
    return Boolean(window.matchMedia?.('(prefers-reduced-motion: reduce)')?.matches)
  } catch {
    return false
  }
}

/** Android 系统 Chrome 不含 `; wv)`；钱包内置页几乎都带这个标记。 */
function isAndroidInAppWebView(ua: string): boolean {
  if (!/Android/i.test(ua)) return false
  if (/; wv\)/i.test(ua)) return true
  if (/\bWebView\b/i.test(ua)) return true
  // 部分 OEM 内置页不带 wv 标记，但仍是 Version/4.0 Chrome WebView。
  if (/Version\/4\.0/i.test(ua) && /Chrome\//i.test(ua)) return true
  return false
}

function chromeMajor(ua: string): number | null {
  const match = ua.match(/Chrome\/(\d+)/i)
  if (!match) return null
  const major = Number(match[1])
  return Number.isFinite(major) ? major : null
}

function canUseCanvas2d(): boolean {
  try {
    const canvas = document.createElement('canvas')
    return Boolean(canvas.getContext && canvas.getContext('2d'))
  } catch {
    return false
  }
}

function detectHomeMode(): HomeMode {
  const ua = navigator.userAgent || ''
  if (prefersReducedMotion()) return 'static'
  if (!canUseCanvas2d()) return 'static'
  if (isAndroidInAppWebView(ua)) return 'static'

  const major = chromeMajor(ua)
  if (major !== null && major > 0 && major < 80) return 'static'

  const memory = (navigator as Navigator & { deviceMemory?: number }).deviceMemory
  if (typeof memory === 'number' && memory > 0 && memory <= 2 && /Android/i.test(ua)) {
    return 'static'
  }

  return 'cinematic'
}

function buildStamp(): string {
  try {
    return document.querySelector('meta[name="aix-build"]')?.getAttribute('content') || ''
  } catch {
    return ''
  }
}

function logHomeMode(mode: HomeMode, source: 'forced' | 'stored' | 'detected') {
  console.info('[AIX home]', mode, `(${source})`, navigator.userAgent, buildStamp())
}

/**
 * 能力退阶：无法安全合成电影式场景时走静态首页。
 * 不要用 1024 PNG 做探测——解码本身就会打满钱包 WebView 的 GPU。
 *
 * 调试：`#/?home=static` 强制静态，`#/?home=cinematic` 强制电影式，`#/?home=auto` 清除覆盖。
 */
export function resolveHomeMode(): HomeMode {
  const forced = parseForcedMode()
  if (forced === 'auto') {
    writeStoredMode(null)
  } else if (forced) {
    writeStoredMode(forced)
    logHomeMode(forced, 'forced')
    return forced
  } else {
    const stored = readStoredMode()
    const ua = navigator.userAgent || ''
    // 钱包 WebView 忽略上次强制的 cinematic，避免黑屏被写进 localStorage 后反复出现。
    if (stored === 'static' || (stored === 'cinematic' && !isAndroidInAppWebView(ua))) {
      logHomeMode(stored, 'stored')
      return stored
    }
  }

  const mode = detectHomeMode()
  logHomeMode(mode, 'detected')
  return mode
}
