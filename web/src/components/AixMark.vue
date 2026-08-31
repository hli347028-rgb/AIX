<!--
  AIX 品牌标记
  ---------------------------------------------------------------------------
  为什么是内联 SVG 而不是位图：
  原来的 aix-logo.png 是一枚 3D 铬质徽标，带倒角高光、镜面反光和星芒装饰。
  那层塑料质感和任何高级方向都冲突，我为它反复调过透明度/亮度/对比度都
  只是在掩盖问题 —— 素材本身才是瓶颈。

  我先用生成模型做了一版扁平位图，但字形是糊的（A 和 I 粘连，AIX 读不出来），
  放在品牌位置上依然显业余。所以改为手写几何 SVG：
    · 任意尺寸像素级锐利，不像位图放大就散
    · 用 currentColor 继承文字颜色，一处改色全局生效
    · 零网络请求、零解码成本
    · 没有高光可言，从根上解决了塑料感

  构成上采纳真实品牌的通行做法 —— 标记与字标分离：
  这个 SVG 只做抽象标记（环 + A 形尖顶），品牌名「AIX」由页面用展示字体
  排真实文字。这样标记负责识别度、文字负责清晰度，
  避免了把三个字母硬塞进小圆环里必然产生的糊成一团。
-->
<template>
  <svg
    class="aix-mark"
    :style="{ width: size + 'px', height: size + 'px' }"
    viewBox="0 0 48 48"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden="true"
  >
    <!-- 外环：细描边 + 低透明度，作为"容器"而非主体 -->
    <circle
      cx="24"
      cy="24"
      r="21.25"
      stroke="currentColor"
      stroke-width="1.5"
      :opacity="ring"
    />
    <!-- 内圈刻度环：极细的第二道环，制造仪器的精密感。
         半径刻意只小 3px，形成紧密的双环而非同心圆装饰。 -->
    <circle
      cx="24"
      cy="24"
      r="17.75"
      stroke="currentColor"
      stroke-width="0.75"
      :opacity="ring * 0.45"
    />
    <!-- A 形尖顶：两条主腿 + 横杆。
         用 stroke 而非 fill，笔画粗细才能保持绝对均匀。
         stroke-linejoin=miter 让顶点是锐角 —— 圆角会立刻失去技术感。 -->
    <path
      d="M15 32.5 L24 15 L33 32.5"
      stroke="currentColor"
      stroke-width="2.75"
      stroke-linejoin="miter"
      stroke-linecap="butt"
    />
    <path
      d="M19.4 26 H28.6"
      stroke="currentColor"
      stroke-width="2.75"
      stroke-linecap="butt"
    />
  </svg>
</template>

<script setup lang="ts">
defineProps({
  size: { type: Number, default: 40 },
  // 环的透明度可调：小尺寸时适当提高，否则细环在 20px 下会消失
  ring: { type: Number, default: 0.4 },
})
</script>

<style scoped>
.aix-mark {
  display: block;
  flex: none;
  /* 不设固定颜色 —— 由父级的 color 决定，便于在蓝/白之间切换 */
}
</style>
