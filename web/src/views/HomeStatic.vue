<template>
  <div class="home-static">
    <Header />
    <main class="home-static-main" :aria-label="$t('home.staticAria')">
      <section class="hero">
        <p class="aix-label">AI PREDICTION EXCHANGE</p>
        <h1>{{ $t('home.heroTitle') }}<em>{{ $t('home.heroEmphasis') }}</em></h1>
        <p class="lead">{{ $t('home.heroDescription') }}</p>
        <button class="aix-btn" type="button" @click="router.push('/node')">
          {{ $t('home.startNow') }}
        </button>
        <p class="meta">
          {{ $t('home.oracle') }} ONLINE · {{ $t('home.tradingPair') }} AIX / WIN
        </p>
      </section>

      <section>
        <p class="aix-label">01 · FUTUREFI</p>
        <h2>{{ $t('home.futureTitle') }}{{ $t('home.futureEmphasis') }}</h2>
        <p>{{ $t('home.futureLead') }}</p>
        <p>{{ $t('home.futureStory1') }}</p>
        <p>{{ $t('home.futureStory2') }}</p>
        <ul class="axis" :aria-label="$t('home.protocolAria')">
          <li><b>AI Prediction Exchange</b>{{ $t('home.predictionTrading') }}</li>
          <li><b>FutureFi DAO</b>{{ $t('home.futureProtocol') }}</li>
          <li><b>WIN Chain</b>{{ $t('home.chainInfrastructure') }}</li>
        </ul>
        <p class="flow">{{ $t('home.flow') }}</p>
        <button class="aix-btn ghost" type="button" @click="router.push('/futurefi')">
          {{ $t('home.viewNarrative') }}
        </button>
      </section>

      <section>
        <p class="aix-label">02 · {{ $t('market.liveMarket') }}</p>
        <h2>AIX / WIN</h2>
        <p>{{ $t('home.marketPlain') }}</p>
      </section>

      <section>
        <p class="aix-label">03 · ECOSYSTEM</p>
        <h2>{{ $t('home.ecosystemTitle') }}{{ $t('home.ecosystemEmphasis') }}</h2>
        <p>{{ $t('home.ecosystemDescription') }}</p>
        <dl>
          <div>
            <dt>{{ $t('home.networkStatus') }}</dt>
            <dd>ONLINE</dd>
          </div>
          <div>
            <dt>{{ $t('home.ecosystemPartners') }}</dt>
            <dd>06</dd>
          </div>
          <div>
            <dt>{{ $t('home.settlementNetwork') }}</dt>
            <dd>MULTI-CHAIN</dd>
          </div>
        </dl>
      </section>

      <nav class="quick-links" :aria-label="$t('home.quickLinksAria')">
        <button type="button" @click="router.push('/recharge')">{{ $t('home.assetRecharge') }}</button>
        <button class="featured" type="button" @click="openPrediction">{{ $t('home.predictFuture') }}</button>
        <button type="button" @click="openWallet">{{ $t('home.getWinWallet') }}</button>
      </nav>
    </main>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import Header from '@/components/Header.vue'

const router = useRouter()
const openPrediction = () => window.open('https://prediction-exchange-lovat.vercel.app', '_blank', 'noopener,noreferrer')
const openWallet = () => window.open('https://testnet.wallet.eoeo.info/06bx', '_blank', 'noopener,noreferrer')
</script>

<style scoped>
.home-static {
  min-height: 100vh;
  background: var(--ink, #fff);
  color: var(--text, #171717);
}

.home-static-main {
  box-sizing: border-box;
  padding: 76px 20px 48px;
}

section {
  padding: 28px 0;
  border-bottom: 1px solid rgba(23, 23, 23, 0.08);
}

h1,
h2 {
  margin: 8px 0 12px;
  font-size: 28px;
  font-weight: 650;
  line-height: 1.2;
  letter-spacing: -0.03em;
}

h1 em,
h2 em {
  font-style: normal;
  color: #0052ff;
}

.lead,
section p {
  margin: 0 0 12px;
  color: var(--text-2, #4a5568);
  font-size: 14px;
  line-height: 1.7;
}

.meta {
  margin: 14px 0 0;
  color: var(--text-3, #8892a4);
  font-size: 11px;
  letter-spacing: 0.06em;
}

.axis {
  margin: 8px 0 16px;
  padding: 0;
  list-style: none;
}

.axis li {
  padding: 10px 0;
  border-top: 1px solid rgba(23, 23, 23, 0.08);
  color: var(--text-2, #4a5568);
  font-size: 13px;
}

.axis b {
  display: block;
  margin-bottom: 2px;
  color: var(--text, #171717);
  font-size: 14px;
}

.flow {
  padding-left: 12px;
  border-left: 2px solid #0052ff;
  color: var(--text, #171717) !important;
}

dl {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin: 16px 0 0;
}

dt {
  color: var(--text-3, #8892a4);
  font-size: 10px;
}

dd {
  margin: 6px 0 0;
  font-size: 12px;
  font-weight: 700;
}

.aix-btn.ghost {
  margin-top: 8px;
  border: 1px solid rgba(0, 82, 255, 0.28);
  background: #fff;
  color: #0052ff;
}

.quick-links {
  display: grid;
  gap: 8px;
  padding: 24px 0 8px;
}

.quick-links button {
  width: 100%;
  padding: 14px 16px;
  border: 1px solid rgba(0, 82, 255, 0.2);
  background: #f2f5ff;
  color: #07101e;
  font-size: 14px;
  font-weight: 650;
  text-align: left;
  cursor: pointer;
}

.quick-links button.featured {
  border-color: #0052ff;
  background: #0052ff;
  color: #fff;
}
</style>
