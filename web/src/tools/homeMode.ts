export type HomeMode = 'static' | 'cinematic'

const STORAGE_KEY = 'aix-home-mode'
const FIRST_SCENE = '/assets/timeline-04-consensus.webp'
const PROBE_MS = 4000

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

/** 仅用于“要不要先探测场景图”。不再据此直接退阶。 */
export function isAndroidInAppWebView(ua = navigator.userAgent || ''): boolean {
  if (!/Android/i.test(ua)) return false
  if (/; wv\)/i.test(ua)) return true
  if (/\bWebView\b/i.test(ua)) return true
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

function hasNonEmptyPixels(img: HTMLImageElement): boolean {
  try {
    const canvas = document.createElement('canvas')
    canvas.width = 32
    canvas.height = 32
    const ctx = canvas.getContext('2d')
    if (!ctx) return false
    ctx.drawImage(img, 0, 0, 32, 32)
    const data = ctx.getImageData(0, 0, 32, 32).data
    let visible = 0
    for (let i = 0; i < data.length; i += 4) {
      if (data[i + 3] > 8 && data[i] + data[i + 1] + data[i + 2] > 12) visible += 1
    }
    return visible > 20
  } catch {
    return false
  }
}

/** 钱包 WebView 先解码一张真实场景图；失败再静态，成功才上电影式。 */
export function probeCinematicReady(): Promise<boolean> {
  return new Promise((resolve) => {
    const img = new Image()
    const finish = (ok: boolean) => {
      clearTimeout(timer)
      img.onload = null
      img.onerror = null
      resolve(ok)
    }
    const timer = window.setTimeout(() => finish(false), PROBE_MS)
    img.onload = () => {
      const decode = 'decode' in img ? img.decode() : Promise.resolve()
      void decode.then(() => finish(hasNonEmptyPixels(img))).catch(() => finish(false))
    }
    img.onerror = () => finish(false)
    img.src = FIRST_SCENE
  })
}

function detectHardStatic(): boolean {
  const ua = navigator.userAgent || ''
  if (prefersReducedMotion()) return true
  if (!canUseCanvas2d()) return true
  const major = chromeMajor(ua)
  if (major !== null && major > 0 && major < 80) return true
  const memory = (navigator as Navigator & { deviceMemory?: number }).deviceMemory
  return typeof memory === 'number' && memory > 0 && memory <= 2 && /Android/i.test(ua)
}

function buildStamp(): string {
  try {
    return document.querySelector('meta[name="aix-build"]')?.getAttribute('content') || ''
  } catch {
    return ''
  }
}

function logHomeMode(mode: HomeMode, source: string) {
  console.info('[AIX home]', mode, `(${source})`, navigator.userAgent, buildStamp())
}

export function persistHomeMode(mode: HomeMode) {
  writeStoredMode(mode)
  logHomeMode(mode, 'persisted')
}

export function shouldProbeCinematic(mode: HomeMode): boolean {
  return mode === 'cinematic' && isAndroidInAppWebView()
}

/**
 * 默认电影式。安卓钱包不再一律静态。
 * 硬条件不足、或上次场景探测失败（localStorage）才静态。
 * `#/?home=static|cinematic|auto` 可覆盖。
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
    if (stored === 'static' || stored === 'cinematic') {
      if (stored === 'cinematic' && detectHardStatic()) {
        logHomeMode('static', 'hard-static')
        return 'static'
      }
      logHomeMode(stored, 'stored')
      return stored
    }
  }

  const mode = detectHardStatic() ? 'static' : 'cinematic'
  logHomeMode(mode, 'detected')
  return mode
}
