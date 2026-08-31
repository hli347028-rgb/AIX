<template>
  <div class="partners-wall" :aria-label="$t('index.partners')">
    <div v-for="(row, rowIndex) in partnerRows" :key="rowIndex" class="wall-row" :data-direction="rowIndex === 1 ? 'reverse' : 'normal'">
      <div class="wall-track">
        <div v-for="(partner, index) in [...row, ...row]" :key="`${rowIndex}-${index}-${partner.name}`" class="wall-item" :class="{ 'brand-item': partner.brand }">
          <span v-if="partner.brand" class="brand-logo" :data-brand="partner.brand">
            <img :src="partner.image" alt="" />
          </span>
          <span v-if="partner.brand" class="brand-name">{{ partner.name }}</span>
          <img v-else :src="partner.image" :alt="`${partner.name} Logo`" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
type Brand = 'bitwinex' | 'winchat' | 'winfive' | 'winchain' | 'winwallet' | 'yunbao'
type Partner = { name: string; image: string; brand?: Brand }

const stock = (name: string, index: number): Partner => ({ name, image: `/static/partners/partner-${index}.png` })
const brand = (name: string, image: string, key: Brand): Partner => ({ name, image, brand: key })

const partnerRows: Partner[][] = [
  [
    stock('金色财经', 1), stock('Chain Catcher', 2),
    brand('BITWINEX', '/static/partners/bitwinex.jpeg', 'bitwinex'),
    stock('Binance', 4), stock('MetaMask', 5), stock('OKX', 6), stock('TokenPocket', 7),
  ],
  [
    stock('Curve', 8), stock('DappRadar', 9), stock('Cardano', 10), stock('Uniswap', 3), stock('PANews', 12),
    brand('WIN CHAT', '/static/partners/win-chat.png', 'winchat'),
    brand('云宝网', '/static/partners/yunbao.png', 'yunbao'),
  ],
  [
    stock('Aave', 17), stock('NEAR', 18), stock('Arbitrum', 19),
    brand('WIN.FIVE', '/static/partners/win-five.png', 'winfive'),
    brand('WIN CHAIN', '/static/partners/win-chain.jpeg', 'winchain'),
    brand('WIN WALLET', '/static/partners/win-wallet-new.jpg', 'winwallet'),
  ],
]
</script>

<style lang="scss" scoped>
.partners-wall {
  margin-top: 18px;
  transform-origin: top;
  animation: wall-unfold .9s .24s cubic-bezier(.16,1,.3,1) both;
  margin-bottom: 52px;
  padding: 14px 0;
  overflow: hidden;
  background: #0a0b0d;
  border: 1px solid #0a0b0d;
  mask-image: linear-gradient(90deg, transparent 0, #000 8%, #000 92%, transparent 100%);
  -webkit-mask-image: linear-gradient(90deg, transparent 0, #000 8%, #000 92%, transparent 100%);

  .wall-row {
    overflow: hidden;
    margin-bottom: 1px;
    border-bottom: 1px solid rgba(255,255,255,.1);

    &:last-child {
      margin-bottom: 0;
      border-bottom: 0;
    }

    .wall-track {
      display: flex;
      align-items: center;
      gap: 26px;
      width: max-content;
      animation: scroll-normal 26s linear infinite;
    }

    &[data-direction="reverse"] .wall-track {
      animation: scroll-reverse 32s linear infinite;
    }

    .wall-item {
      flex-shrink: 0;
      display: flex;
      align-items: center;
      justify-content: center;
      min-width: 96px;
      min-height: 52px;
      padding: 8px 12px;

      > img {
        max-height: 30px;
        width: auto;
        object-fit: contain;
        opacity: .82;
        mix-blend-mode: screen;
        transition: opacity var(--t-base) var(--ease), transform var(--t-base) var(--ease);
      }

      &:hover > img,
      &:hover .brand-logo {
        opacity: 1;
        transform: translateY(-2px);
      }
    }

    .brand-item {
      gap: 9px;
      min-width: 148px;
    }

    .brand-logo {
      position: relative;
      width: 30px;
      height: 30px;
      flex: 0 0 30px;
      overflow: hidden;
      opacity: .9;
      transition: opacity var(--t-base) var(--ease), transform var(--t-base) var(--ease);
    }

    .brand-logo img {
      position: absolute;
      max-width: none;
      object-fit: cover;
      mix-blend-mode: screen;
      filter: contrast(1.08) saturate(1.06);
    }

    .brand-logo[data-brand='bitwinex'] img {
      width: 61px;
      height: 61px;
      left: -15px;
      top: -13px;
    }

    .brand-logo[data-brand='winchat'] img {
      width: 50px;
      height: auto;
      left: -10px;
      top: -3px;
    }

    .brand-logo[data-brand='winfive'] img {
      width: 30px;
      height: 30px;
      inset: 0;
    }

    .brand-logo[data-brand='winchain'] img {
      width: 60px;
      height: 60px;
      left: -15px;
      top: -15px;
    }

    .brand-logo[data-brand='winwallet'] {
      width: 34px;
      height: 34px;
      flex-basis: 34px;
      border-radius: 7px;
    }

    .brand-logo[data-brand='winwallet'] img {
      width: 46px;
      height: 46px;
      left: -6px;
      top: -6px;
    }

    .brand-logo[data-brand='yunbao'] {
      border-radius: 50%;
      background: #fff;
    }

    .brand-logo[data-brand='yunbao'] img {
      width: 42px;
      height: 42px;
      left: -6px;
      top: -6px;
    }

    .brand-name {
      color: rgba(255,255,255,.86);
      font-family: Arial, Helvetica, sans-serif;
      font-size: 15px;
      font-weight: 700;
      line-height: 1;
      letter-spacing: .025em;
      white-space: nowrap;
    }
  }
}

.partners-wall:hover .wall-track { animation-play-state: paused; }

@keyframes wall-unfold {
  from { opacity: 0; transform: scaleY(.72) translateY(18px); clip-path: inset(0 0 100%); }
  to { opacity: 1; transform: scaleY(1) translateY(0); clip-path: inset(0); }
}

@keyframes scroll-normal {
  from { transform: translateX(0); }
  to { transform: translateX(-50%); }
}

@keyframes scroll-reverse {
  from { transform: translateX(-50%); }
  to { transform: translateX(0); }
}

@media (prefers-reduced-motion: reduce) {
  .partners-wall { animation: none; }
  .partners-wall .wall-row .wall-track { animation: none; transform: none; }
  .partners-wall .wall-row:nth-child(n + 2) { display: none; }
}
</style>
