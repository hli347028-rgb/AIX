<template>
    <!-- :right-text="lang('规则')" -->
<div class='page'>
  <van-nav-bar
    :title="lang('算力统计')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
    @click-right="handleRule"
  />
  <div class="page-main">
    <div class="current-price">
      <span>{{ lang('当前代币价格') }}</span>
      <span>1.000000 USDT</span>
    </div>
    <div class="page-title">{{ lang('算力统计') }}</div>
    <div class="total-power">
      <span>{{ lang('我的总算力') }}</span>
      <span>0.00 MB(0.00%)</span>
    </div>
    <div class="double-column">
      <div class="column-left">
        <p>{{ lang('我的原发算力') }}</p>
        <p>0.00 MB</p>
      </div>
      <div class="column-right">
        <p>{{ lang('我的分享算力') }}</p>
        <p>0.00 MB</p>
      </div>
    </div>
    <div class="current-price">
      <span>{{ lang('USD Brc20') }}</span>
      <p>{{ userinfo.rawNew }}</p>
    </div>
    <div class="double-column-2">
      <div class="column-left">
        <p>{{ lang('我的溢出算力') }}</p>
        <p>0.00 MB</p>
      </div>
      <div class="column-right">
        <p>{{ lang('全网原发算力') }}</p>
        <p>0.00 MB</p>
      </div>
    </div>
    <div class="double-column-3">
      <div class="column-left">
        <p>{{ lang('全网分享算力') }}</p>
        <p>0.00 MB</p>
      </div>
      <div class="column-right">
        <p>{{ lang('全网总算力') }}</p>
        <p>0.00 MB</p>
      </div>
    </div>
  </div>
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import { useRouter } from 'vue-router'
import lang from '@/i18n/index'

const router = useRouter()
const person = userPerson();
const userinfo = $computed(() => person.userinfo);

const handleBack = () => {
  router.back()
}

const handleRule = () => {

}

</script>
<style lang='less' scoped>
  .page {
    min-height: 100vh;
    box-sizing: border-box;
    padding: 50px 15px 20px 15px;
    background: url('../../assets/images/topbg4.png') no-repeat;
    background-size: 100% auto;
    .current-price {
      background: hsla(0, 0%, 100%, .1);
      border: 1px solid var(--hair);
      border-radius: 8px;
      padding: 20px 15px;
      display: flex;
      margin-bottom: 20px;
      justify-content: space-between;
      span {
        &:nth-child(2) {
          /* 原为高饱和蓝 rgb(21,151,229)：价格数值不需要用颜色强调，
             近白 + 等宽数字已经足够醒目，且与全站无彩色方向一致。 */
          color: var(--text);
          font-weight: 500;
          font-variant-numeric: tabular-nums;
        }
      }
    }
    .page-title {
      padding: 20px 0;
      font-weight: bold;
    }
    .total-power {
      width: 100%;
      height: 89px;
      background: url('../../assets/images/boxbg.png') no-repeat;
      background-size: 100% 89px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 0 20px;
      box-sizing: border-box;
      margin-bottom: 20px;
      span {
        &:nth-child(2) {
          font-size: 16px;
          font-weight: bold;
        }
      }
    }
    .double-column {
      display: flex;
      justify-content: space-between;
      gap: 10px;
      margin-bottom: 20px;
      &>div {
        flex: 1;
        border-radius: 8px;
        padding: 15px;
        display: flex;
        flex-direction: column;
        gap: 15px;
        p {
          &:nth-child(2) {
            font-size: 16px;
            font-weight: 500;
          }
        }
      }
      /* 三对统计卡原本都是「浅色块 + 蓝色块」。蓝色归中性后两块会完全同色，
         所以配对关系改由「实底 vs 描边」承担：重点项实底，参照项描边。
         这比两块彩色更克制，主次也一眼可辨。三对统一用这一套写法。 */
      .column-left {
        background: var(--accent);
        color: var(--ink-deep);
      }
      .column-right {
        background: var(--surface-1);
        border: 1px solid var(--hair-2);
        color: var(--text);
      }
    }
    .double-column-2 {
      display: flex;
      justify-content: space-between;
      gap: 10px;
      margin-bottom: 20px;
      &>div {
        border-radius: 8px;
        padding: 15px;
        display: flex;
        flex-direction: column;
        gap: 15px;
        p {
          &:nth-child(2) {
            font-size: 16px;
            font-weight: 500;
          }
        }
      }
      .column-left {
        width: 45%;
        background: var(--accent);
        /* 底色转近白，字色必须同时从 #FFF 改为近黑 */
        color: var(--ink-deep);
      }
      .column-right {
        flex: 1;
        background: var(--surface-1);
        border: 1px solid var(--hair-2);
        color: var(--text);
      }
    }
    .double-column-3 {
      display: flex;
      justify-content: space-between;
      gap: 10px;
      margin-bottom: 20px;
      &>div {
        border-radius: 8px;
        padding: 15px;
        display: flex;
        flex-direction: column;
        gap: 15px;
        p {
          &:nth-child(2) {
            font-size: 16px;
            font-weight: 500;
          }
        }
      }
      /* 这一对的重点项在右侧（全网总算力是汇总值），保留其实底不动，
         只把左侧参照项的亮白描边降为发丝线 —— 描边用 --accent（近白）
         会和实底一样抢眼，失去主次。 */
      .column-left {
        width: 50%;
        background: var(--surface-1);
        border: 1px solid var(--hair-2);
        color: var(--text);
      }
      .column-right {
        flex: 1;
        background: var(--accent);
        color: var(--ink-deep);
      }
    }
  }
</style>
