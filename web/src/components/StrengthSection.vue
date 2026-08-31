<template>
  <div class="strength-section">
    <h2 class="section-title">{{ $t('advantage.title') }}</h2>

    <!-- 静态清单。原本是自动轮播的三张卡，见 <style> 中的说明。 -->
    <ul class="strength-list">
      <li v-for="card in cards" :key="card.tag" class="strength-item">
        <svg
          class="item-icon"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.25"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path :d="card.iconPath" />
        </svg>
        <div class="item-body">
          <h3 class="item-title">{{ $t(card.tag) }}</h3>
          <p class="item-desc">{{ $t(card.desc) }}</p>
        </div>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
interface StrengthCard {
  tag: string
  /** 线性图标路径（24x24 viewBox） */
  iconPath: string
  desc: string
}

/* 去掉了 illustration 字段：三张深色库存图（保险箱转盘/金币堆/电路板）
   已不再使用。同时删除了首尾克隆、自动播放定时器、ResizeObserver 和
   translateX 计算 —— 静态清单不需要这些。 */
const cards: StrengthCard[] = [
  {
    tag: 'advantage.revenueTag',
    // 上升折线
    iconPath: 'M3 17l5-5 4 3 6-7M17 8h4v4',
    desc: 'advantage.revenueDesc'
  },
  {
    tag: 'advantage.transparentTag',
    // 链接环
    iconPath: 'M10 13a4 4 0 006 0l2-2a4 4 0 00-6-6l-1 1M14 11a4 4 0 00-6 0l-2 2a4 4 0 006 6l1-1',
    desc: 'advantage.transparentDesc'
  },
  {
    tag: 'advantage.securityTag',
    // 盾牌 + 勾
    iconPath: 'M12 3l7 3v6c0 4-3 7-7 9-4-2-7-5-7-9V6l7-3zM9 12l2 2 4-4',
    desc: 'advantage.securityDesc'
  }
]
</script>

<style lang="scss" scoped>
/* 已改为消费 polish.less 的设计令牌，不再依赖 variables.scss 的旧色值。 */

/* 这一块原本是「自动轮播的三张卡 + 深色库存图 + 悬浮胶囊标签」。
   三个问题叠在一起，是全页最主要的廉价感来源：

     1. 库存图 —— 保险箱转盘、金币堆、电路板。这类图几乎是"廉价加密项目"
        的视觉标志，而且它们自带的棕/蓝色偏与整套中性灰阶冲突（之前还得
        专门写一个构建脚本去给它们统一色调，这本身就是信号：素材不对）。
     2. 自动轮播 —— 只有三条短文案，却让用户每 3 秒被动等一次，
        还永远只能看到一张半。手法比内容重。
     3. 悬浮胶囊标签 —— 骑在卡沿上的小圆角标签是促销贴纸的语言。

   改为静态纵向清单：三条并列可见，一眼读完，不抢视线也不需要等待。
   这也和页面其他部分（账目行、区块标题）的发丝线语言统一起来。 */
.strength-section {
  /* 与首页其余区块统一到 40px 的纵向节奏 */
  margin-bottom: 40px;
}

  /* 区块标题。与首页"合作机构"保持同一套写法（一致性是对的），
     但档位从 --fs-micro(10px) 提到 --fs-title(19px)。

     原来的问题是层级倒挂：区块标题用了全站最弱的一档 —— 10px 全大写
     微型标签 —— 结果比它所统辖的条目标题（item-title，15px）还轻。
     标题比内容轻，读者就无法靠字号判断结构。
     微型大写标签适合做 kicker（眉标），不适合做区块主标题。

     字距同时从 0.32em 收到 --ls-tight：宽字距是小号大写标签的做法，
     19px 的中文标题用宽字距会散。 */
  .section-title {
  margin: 0 0 14px;
  font-family: var(--aix-font-display);
  font-size: var(--fs-title);
  font-weight: 500;
  letter-spacing: var(--ls-tight);
  color: var(--text);
  }

.strength-list {
  margin: 0;
  padding: 0;
  list-style: none;
}

/* 发丝线分隔而非卡片：三条内容是并列关系，用一条线区隔就够，
   给每条套一个圆角容器只会把简单的列表变成三个盒子。 */
.strength-item {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 20px 0;
  border-bottom: 1px solid var(--hair);
}

.strength-item:last-child {
  border-bottom: 0;
}

/* 图标压到最小可辨，且不给强调色 —— 它是标记，不是主角 */
.item-icon {
  flex: 0 0 auto;
  width: 18px;
  height: 18px;
  margin-top: 3px;
  color: var(--text-3);
}

.item-body {
  min-width: 0;
}

/* 标题用细字重的显示字体。原本这里是 10px 全大写粗体的胶囊标签文字，
   现在恢复成正常的小标题 —— 内容本来就是完整短句，不该被塞进标签里。 */
.item-title {
  margin: 0 0 5px;
  font-family: var(--aix-font-display);
  font-size: 15px;
  font-weight: 500;
  letter-spacing: -0.005em;
  color: var(--accent);
}

.item-desc {
  margin: 0;
  font-size: var(--fs-sm);
  line-height: 1.7;
  color: var(--text-2);
  text-wrap: pretty;
}

.strength-section { padding-top: 12px; }
.strength-item {
  position: relative;
  transition: padding-left .35s cubic-bezier(.2,.8,.2,1), background .35s ease;
}
.strength-item::after {
  content: '';
  position: absolute;
  left: 0;
  bottom: -1px;
  width: 0;
  height: 1px;
  background: var(--accent);
  transition: width .45s cubic-bezier(.2,.8,.2,1);
}
.strength-item:hover { padding-left: 14px; }
.strength-item:hover::after { width: 100%; }
.item-icon { transition: color .3s ease, transform .35s cubic-bezier(.2,.8,.2,1); }
.strength-item:hover .item-icon { color: var(--accent); transform: translateY(-2px); }
@media (prefers-reduced-motion: reduce) {
  .strength-item, .strength-item::after, .item-icon { transition: none; }
}
</style>
