<template>
  <div class="mine-page">
    <Header />
    <div class="container">
      <!-- 空投面板。原先是「一个卡片 + 5 个各带底色圆角的数据行 + 一个描边胶囊按钮」，
           5 行长得一模一样，可领取金额（唯一可操作的数字）完全没有被突出。
           现在把可领取金额提为主角，其余作为账目行跟在后面。 -->
      <section class="airdrop">
        <p class="aix-label">{{ $t('mine.tokenAirdrop') }}</p>

        <div class="claimable">
          <span class="claimable-num aix-figure">{{ formatNum(airdrop.claimable) }}</span>
          <span class="claimable-unit aix-figure-unit">AIX</span>
        </div>
        <p class="claimable-cap">{{ $t('mine.claimableAmount') }}</p>

        <div class="aix-ledger">
          <div class="aix-ledger-row">
            <span class="k">{{ $t('mine.pendingAmount') }}</span>
            <span class="v">{{ formatNum(airdrop.pending) }}</span>
          </div>
          <div class="aix-ledger-row">
            <span class="k">{{ $t('mine.claimedAmount') }}</span>
            <span class="v">{{ formatNum(airdrop.claimed) }}</span>
          </div>
          <div class="aix-ledger-row">
            <span class="k">{{ $t('mine.totalNodes') }}</span>
            <span class="v">{{ airdrop.nodes }}</span>
          </div>
          <div class="aix-ledger-row">
            <span class="k">{{ $t('mine.releaseCountdown') }}</span>
            <!-- i18n 里已有 mine.none（'无'），比硬编码的 '--' 更合适，
                 且 8 个语种都已翻译。 -->
            <span class="v">{{ airdrop.countdown || $t('mine.none') }}</span>
          </div>
        </div>

        <!-- 无可领取额度时按钮本就点不出结果，原先却仍是可点的样式。
             现在显式禁用，状态一眼可见。 -->
        <button
          class="aix-btn claim-btn"
          type="button"
          :disabled="!canClaim"
          @click="onClaim"
        >
          {{ $t('mine.claimAirdrop') }}
        </button>
      </section>

      <div class="section-title-wrap">
        <div class="title-bar"></div>
        <h3 class="section-title">{{ $t('mine.claimRecord') }}</h3>
      </div>

      <div class="ledger">
        <div class="ledger-head">
          <span>{{ $t('mine.time') }}</span>
          <span class="num">{{ $t('mine.amount') }}</span>
        </div>
        <template v-if="claimRecords.length > 0">
          <div class="ledger-row" v-for="(item, index) in claimRecords" :key="index">
            <span class="time">{{ item.createdAt }}</span>
            <span class="num">{{ formatNum(item.amount) }}</span>
          </div>
        </template>
        <p v-else class="empty-state">{{ $t('common.noData') }}</p>
      </div>

      <div class="safe-bottom"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import Header from '@/components/Header.vue'
import { computed } from 'vue'

// 空投数据目前尚无对应接口，先集中成一个对象，
// 接上接口时只需替换这一处，而不用在模板里逐个改写死的 0。
const airdrop = {
  claimable: 0,
  pending: 0,
  claimed: 0,
  nodes: 0,
  countdown: '',
}

const claimRecords: Array<{ createdAt: string; amount: number }> = []

const canClaim = computed(() => Number(airdrop.claimable) > 0)

const formatNum = (value: any) => Number(value || 0).toFixed(2)

const onClaim = () => {
  if (!canClaim.value) return
  // 领取接口待接入
}
</script>

<style lang="scss" scoped>
/* 原先这里的 .stats-value 有三个颜色修饰类 .green / .accent / .purple，
   但 .green 和 .purple 指向的是同一个变量，且都不是绿色或紫色 ——
   三个类名、两种颜色、零语义。数值统一用前景色，
   只把真正需要强调的"可领取金额"提亮。 */

.mine-page {
  min-height: 100vh;
  padding-top: 64px;
}

.container {
  /* 不再设 max-width：polish.less 把 body > #app 限死在 414px，
     这里写 760px 永远不会生效。 */
  padding: 0 20px;
}

.airdrop {
  padding: 28px 0 0;
}

/* 可领取金额：这一页唯一可操作的数字，给它最大的字号。 */
.claimable {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-top: 10px;
}

/* 外观来自 .aix-figure / .aix-figure-unit 原语（见 polish.less 第 10a-2 节）。 */

.claimable-cap {
  margin: 9px 0 22px;
  font-size: 12px;
  color: var(--text-3);
}

/* 外观（含禁用态）全部来自 .aix-btn 原语，这里只留本页需要的外边距。 */
.claim-btn {
  margin-top: 24px;
}

/* 领取记录表 */
.ledger-head,
.ledger-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
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

  &:last-of-type {
    border-bottom: 0;
  }

  .time {
    font-variant-numeric: tabular-nums;
    font-size: 12px;
    color: var(--text-2);
  }

  .num {
    font-family: var(--aix-font-display);
    font-variant-numeric: tabular-nums;
    font-size: 15px;
    font-weight: 500;
    color: var(--text);
  }
}

.ledger-head .num,
.ledger-row .num {
  text-align: right;
}

.empty-state {
  margin: 0;
  padding: 60px 0;
  text-align: center;
  font-size: 12px;
  color: var(--text-3);
}

.safe-bottom {
  height: 56px;
}
</style>
