<template>
  <!-- 只有数字段参与滚动，前后缀原样保留。
       aria-label 始终给出最终值，读屏软件不会读到滚动中的中间数字。 -->
  <span ref="rootEl" class="aix-countup" :aria-label="text">
    <span aria-hidden="true">{{ prefix }}</span><span class="countup-num" aria-hidden="true">{{ display }}</span><span aria-hidden="true">{{ suffix }}</span>
  </span>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'

const props = withDefaults(
  defineProps<{
    /** 待展示的完整文本，可以是「5亿枚」「500 Million」「1,234 TH/s」这类带单位的本地化字符串 */
    text: string
    /** 滚动时长（ms） */
    duration?: number
  }>(),
  { duration: 1100 }
)

/**
 * 关键约束：传进来的是**本地化字符串**而不是数字。
 * 「5亿枚」的单位嵌在文本里，「500 Million」的单位在后面，
 * 而某些语种可能根本不以数字开头。
 * 所以这里把字符串拆成 前缀 / 数字 / 后缀 三段，只让中间那段滚动；
 * 如果拆不出数字（比如纯文字），就整段静态输出、完全不做动画 ——
 * 宁可不动，也不能把非数字内容硬塞进计数器里。
 */
const parsed = computed(() => {
  // 匹配第一段数字（允许千分位逗号和小数点）
  const m = props.text.match(/[\d][\d,]*(\.\d+)?/)
  if (!m || m.index === undefined) {
    return { prefix: props.text, num: null as number | null, suffix: '', raw: '' }
  }
  const raw = m[0]
  const num = Number(raw.replace(/,/g, ''))
  if (!Number.isFinite(num)) {
    return { prefix: props.text, num: null, suffix: '', raw: '' }
  }
  return {
    prefix: props.text.slice(0, m.index),
    num,
    suffix: props.text.slice(m.index + raw.length),
    raw,
  }
})

const prefix = computed(() => parsed.value.prefix)
const suffix = computed(() => parsed.value.suffix)

/** 是否使用千分位（跟随原字符串的写法，避免把「500」显示成「500」而把「1,000」写成「1000」） */
const useGrouping = computed(() => parsed.value.raw.includes(','))
/** 小数位数同样跟随原字符串，避免滚动结束后与原文不一致 */
const decimals = computed(() => {
  const dot = parsed.value.raw.split('.')[1]
  return dot ? dot.length : 0
})

const current = ref<number>(0)
const done = ref(false)
/**
 * 是否已启动滚动。
 * 这个标志解决一个真实的失效风险：滚动未开始前 current 是 0，
 * 于是页面上会先显示「0亿枚」—— 一旦 IntersectionObserver 因任何原因
 * 没能触发（元素始终不达阈值、观察器被提前回收、浏览器异常），
 * 用户看到的就是一个**永久错误的数字 0**，而不是"动画没播"。
 * 展示错误数据比没有动画严重得多，所以未启动时一律显示终值。
 */
const started = ref(false)

const format = (v: number) =>
  v.toLocaleString(undefined, {
    minimumFractionDigits: decimals.value,
    maximumFractionDigits: decimals.value,
    useGrouping: useGrouping.value,
  })

const display = computed(() => {
  if (parsed.value.num === null) return ''
  // 动画结束后直接回落到原始写法，确保和文案完全一致
  if (done.value) return parsed.value.raw
  // 还没开始滚动时显示终值，而不是 0 —— 见 started 的说明
  if (!started.value) return parsed.value.raw
  return format(current.value)
})

let rafId = 0
let observer: IntersectionObserver | null = null
const rootEl = ref<HTMLElement | null>(null)

const finish = () => {
  done.value = true
}

const run = () => {
  const target = parsed.value.num
  if (target === null) return

  // 尊重系统「减少动态效果」：直接落到终值，不做滚动
  if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) {
    finish()
    return
  }

  // 置位后 display 才会切到滚动中的中间值；在此之前一直显示终值
  started.value = true
  const start = performance.now()
  const tick = (now: number) => {
    const p = Math.min(1, (now - start) / props.duration)
    // easeOutExpo：开头快、尾部长时间缓停，读起来像"数字落定"而不是匀速跑表
    const eased = p === 1 ? 1 : 1 - Math.pow(2, -10 * p)
    current.value = target * eased
    if (p < 1) {
      rafId = requestAnimationFrame(tick)
    } else {
      finish()
    }
  }
  rafId = requestAnimationFrame(tick)
}

onMounted(() => {
  if (parsed.value.num === null) return

  const el = rootEl.value
  // 只在进入视口后才开始滚动 —— 否则用户滚到这一块时动画早已播完，等于没做
  if (!el || typeof IntersectionObserver === 'undefined') {
    run()
    return
  }
  observer = new IntersectionObserver(
    (entries) => {
      for (const e of entries) {
        if (e.isIntersecting) {
          run()
          observer?.disconnect()
          observer = null
          break
        }
      }
    },
    { threshold: 0.4 }
  )
  observer.observe(el)
})

onBeforeUnmount(() => {
  if (rafId) cancelAnimationFrame(rafId)
  observer?.disconnect()
})
</script>

<style scoped lang="less">
/* 等宽数字：滚动过程中每一位宽度恒定，数字不会左右抖动。
   这是计数动画最容易被忽略的一点 —— 少了它，
   位数变化（9→10）会让整行文字横向跳一下。 */
.countup-num {
  font-variant-numeric: tabular-nums;
}

/* 「5亿枚」等文案保持横向；窄栅格里 CJK 逐字折行会看起来像竖排 */
.aix-countup {
  display: inline-block;
  white-space: nowrap;
  writing-mode: horizontal-tb;
}
</style>
