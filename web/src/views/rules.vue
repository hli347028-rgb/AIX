<template>
  <div class="rules-page">
    <Header />
    <main class="rules-main">
      <header class="page-heading">
        <RouterLink class="back-home" to="/" :aria-label="$t('common.backHome')">
          <span aria-hidden="true">←</span> {{ $t('common.backHome') }}
        </RouterLink>
        <p>{{ $t('index.projectInfoTitle1') }}</p>
        <h1>{{ $t('index.predictionExchangeTitle') }}</h1>
        <span>{{ $t('index.aiAgentBot') }}</span>
      </header>

      <section class="supply-section" :aria-labelledby="'supply-title'">
        <div class="section-label">
          <p id="supply-title">{{ $t('index.tokenSupplyLabel') }}</p>
          <small>AIX / FIXED SUPPLY</small>
        </div>
        <AixCountUp class="supply-number" :text="$t('index.tokenSupplyValue')" />
        <div class="token-name"><small>{{ $t('index.tokenNameLabel') }}</small><strong>AIX</strong></div>
      </section>

      <section class="fact-strip" :aria-label="$t('index.rulesSummaryTitle')">
        <div><small>{{ $t('index.tokenNameLabel') }}</small><strong>AIX</strong></div>
        <div><small>{{ $t('index.tradingPairLabel') }}</small><strong>WIN / AIX</strong></div>
        <div><small>{{ $t('index.aiAgentBot') }}</small><strong class="online"><i></i>ONLINE</strong></div>
      </section>

      <section class="rules-editorial">
        <article>
          <header><span>01</span><h2>{{ $t('index.slippageRules') }}</h2></header>
          <ul>
            <li>{{ $t('index.buyDisabled') }}</li>
            <li>{{ $t('index.sellFee') }}</li>
            <li>{{ $t('index.priceDrop2Rule') }}</li>
            <li>{{ $t('index.priceDrop10Rule') }}</li>
          </ul>
        </article>
        <article>
          <header><span>02</span><h2>{{ $t('index.rulesSummaryTitle') }}</h2></header>
          <ul>
            <li>{{ $t('index.staticIncomeRule') }}</li>
            <li>{{ $t('index.directReferralRule') }}</li>
            <li>{{ $t('index.managementRewardRule') }}</li>
            <li>{{ $t('index.exitRule') }}</li>
          </ul>
        </article>
      </section>

      <section class="strength-wrap"><StrengthSection /></section>
    </main>
  </div>
</template>

<script setup lang="ts">
import Header from '@/components/Header.vue'
import AixCountUp from '@/components/AixCountUp.vue'
import StrengthSection from '@/components/StrengthSection.vue'
</script>

<style lang="scss" scoped>
.rules-page{--paper:#f3f5f7;--ink:#111820;--muted:#64707d;--line:rgba(20,33,48,.16);--blue:#0052ff;--accent:#0052ff;--accent-bright:#0052ff;--accent-deep:#0648df;--accent-dim:rgba(0,82,255,.1);min-height:100vh;background:var(--paper);color:var(--ink)}
.rules-main{max-width:1120px;margin:0 auto;padding:108px 34px 72px}
.page-heading{padding-bottom:46px;border-bottom:1px solid var(--line)}
.back-home{display:inline-flex;align-items:center;gap:8px;margin-bottom:54px;color:var(--muted);font-size:13px;text-decoration:none;transition:color .25s ease}.back-home span{font-size:18px}.back-home:hover,.back-home:focus-visible{color:var(--blue);outline:none}
.page-heading>p{margin:0 0 18px;font:500 9px/1 var(--aix-font-display);letter-spacing:.17em;color:var(--blue)}
.page-heading h1{margin:0;font-family:var(--aix-font-sans);font-size:clamp(34px,5vw,52px);font-weight:500;line-height:1.15;letter-spacing:-.035em}.page-heading>span{display:block;margin-top:12px;color:var(--blue);font-weight:500}
.supply-section{display:grid;grid-template-columns:auto max-content auto;align-items:end;gap:30px;padding:54px 0 42px;border-bottom:1px solid var(--line)}
.section-label{display:flex;flex-direction:column;gap:8px}.section-label p{margin:0;font-size:12px;font-weight:500;color:var(--blue)}.section-label small{font:500 8px/1 var(--aix-font-display);letter-spacing:.12em;color:#87919c}
.supply-number{font-family:var(--aix-font-display);font-size:clamp(52px,9vw,92px);font-weight:500;line-height:.82;letter-spacing:-.07em;color:var(--blue);white-space:nowrap;writing-mode:horizontal-tb}
.token-name{display:flex;flex-direction:column;gap:6px}.token-name small{font-size:11px;color:var(--muted)}.token-name strong{font:600 17px/1 var(--aix-font-display);color:var(--blue)}
.fact-strip{display:grid;grid-template-columns:repeat(3,1fr);border-bottom:1px solid var(--line)}.fact-strip>div{display:flex;justify-content:space-between;align-items:center;padding:20px;border-right:1px solid var(--line)}.fact-strip>div:first-child{padding-left:0}.fact-strip>div:last-child{border:0}.fact-strip small{font-size:10px;color:#7a8591}.fact-strip strong{font:550 12px/1 var(--aix-font-display);letter-spacing:.05em;color:var(--blue)}.fact-strip .online{display:flex;align-items:center;gap:7px}.fact-strip .online i{width:4px;height:4px;border-radius:50%;background:currentColor}
.rules-editorial{display:grid;grid-template-columns:1fr 1fr;gap:80px;padding:82px 0}.rules-editorial article header{display:flex;align-items:baseline;gap:18px;margin-bottom:24px}.rules-editorial article header span{font:500 9px/1 var(--aix-font-display);color:var(--blue)}.rules-editorial h2{margin:0;font-family:var(--aix-font-sans);font-size:21px;font-weight:500;letter-spacing:-.02em}.rules-editorial ul{margin:0;padding:0;list-style:none}.rules-editorial li{padding:17px 0;border-top:1px solid var(--line);color:#566270;line-height:1.55}
.strength-wrap{padding-top:4px;border-top:1px solid var(--line)}
@media(max-width:759px){.rules-main{padding:92px 22px 54px}.back-home{margin-bottom:40px}.page-heading{padding-bottom:36px}.supply-section{grid-template-columns:1fr auto;grid-template-areas:"number number" "label token";gap:18px 24px;padding:42px 0 34px}.section-label{grid-area:label}.supply-number{grid-area:number;font-size:clamp(44px,14vw,64px);white-space:nowrap}.token-name{grid-area:token;flex-direction:row;align-items:center;justify-self:end}.fact-strip{grid-template-columns:1fr}.fact-strip>div,.fact-strip>div:first-child{padding:16px 0;border-right:0;border-bottom:1px solid var(--line)}.rules-editorial{grid-template-columns:1fr;gap:54px;padding:62px 0}}
</style>
