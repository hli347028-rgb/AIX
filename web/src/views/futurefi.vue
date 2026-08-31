<template>
  <div class="futurefi-page">
    <header class="article-nav">
      <RouterLink to="/" :aria-label="$t('futurefi.backHome')">← <span>{{ $t('futurefi.backHome') }}</span></RouterLink>
      <span>AIX / FUTUREFI</span>
    </header>

    <main>
      <section class="article-hero">
        <p class="eyebrow">WIN CHAIN / FUTURE FINANCE</p>
        <h1><span>AIX × FutureFi</span><em><span>{{ $t('futurefi.heroTitle') }}</span><span>{{ $t('futurefi.heroDirection') }}</span></em></h1>
        <p class="deck">AI Prediction Exchange · FutureFi DAO · WIN Chain</p>
        <blockquote><strong>Trade Tomorrow, Today.</strong><span>{{ $t('futurefi.slogan') }}</span></blockquote>
      </section>

      <article class="article-body">
        <nav class="chapter-map" :aria-label="$t('futurefi.chapters')">
          <p>CONTENTS / {{ $t('futurefi.contents') }}</p>
          <a v-for="section in sections" :key="section.no" :href="`#section-${section.no}`"><span>{{ section.no }}</span>{{ section.en }}</a>
        </nav>

        <div class="reading-column">
          <p class="opening">{{ $t('futurefi.opening') }}</p>

          <section v-for="section in sections" :key="section.no" :id="`section-${section.no}`" class="chapter">
            <aside><span>{{ section.no }}</span><small>{{ section.en }}</small></aside>
            <div class="chapter-content">
              <h2>{{ section.title }}</h2>
              <template v-for="(block, index) in section.blocks" :key="index">
                <p v-if="block.type === 'p'">{{ block.text }}</p>
                <blockquote v-else-if="block.type === 'quote'">{{ block.text }}</blockquote>
                <ul v-else-if="block.type === 'list'" class="feature-list"><li v-for="(item, itemIndex) in block.items" :key="item"><b>{{ String(itemIndex + 1).padStart(2, '0') }}</b><span>{{ item }}</span></li></ul>
                <figure v-else-if="block.type === 'flow'" class="process-flow">
                  <figcaption><span>SYSTEM FLOW / {{ $t('futurefi.systemFlow') }}</span><small>{{ String(block.items.length).padStart(2, '0') }} STEPS</small></figcaption>
                  <ol>
                    <li v-for="(item, itemIndex) in block.items" :key="item" :class="{ 'is-terminal': itemIndex === block.items.length - 1 }">
                      <i aria-hidden="true"></i><small>{{ String(itemIndex + 1).padStart(2, '0') }}</small><strong>{{ item }}</strong>
                    </li>
                  </ol>
                </figure>
                <div v-else class="formula">{{ block.text }}</div>
              </template>
            </div>
          </section>

          <section class="closing">
            <p>CONCLUSION / {{ $t('futurefi.conclusion') }}</p>
            <h2>Predict the Future.<br>Build the Future.</h2>
            <div><p>{{ $t('futurefi.closing1') }}</p><p>{{ $t('futurefi.closing2') }}</p></div>
            <strong>{{ $t('futurefi.closingSlogan') }}</strong>
          </section>

          <section class="links" aria-labelledby="links-title">
            <header><p>OFFICIAL ACCESS</p><h2 id="links-title">{{ $t('futurefi.officialAccess') }}</h2></header>
            <a v-for="(link, index) in links" :key="link.url" :href="link.url" target="_blank" rel="noopener noreferrer"><b>{{ String(index + 1).padStart(2, '0') }}</b><span>{{ link.name }}</span><small>{{ link.url }}</small><i>↗</i></a>
          </section>
        </div>
      </article>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

type Block = { type: 'p' | 'quote' | 'formula'; text: string } | { type: 'list' | 'flow'; items: string[] }
type Section = { no: string; en: string; title: string; blocks: Block[] }

const { locale, tm } = useI18n()
const zhSections: Section[] = [
  { no:'01',en:'INFRASTRUCTURE',title:'从 WIN Chain 到 FutureFi：云链正在搭建完整数字经济基础设施',blocks:[{type:'p',text:'WIN Chain 的目标，并不是只建设一条公链或发行一个代币，而是围绕 WIN 建立交易、钱包、社交、支付、云引擎、AI、项目发行以及链上金融应用组成的完整数字经济生态。'},{type:'p',text:'WIN Chain 官方生态规划将 WIN 定位为整个生态中的结算、应用、治理、燃料与价值承载核心，并围绕交易所、WIN Chain、WIN Wallet、WIN Chat、WIN Pay、Win-Engine 与 WIN AI Ecosystem 等产品形成统一生态体系。'},{type:'quote',text:'FutureFi，正是在这套基础设施之上进一步延伸出来的新金融应用方向。'}]},
  { no:'02',en:'FUTURE FINANCE',title:'什么是 FutureFi？',blocks:[{type:'formula',text:'FutureFi = Future Finance，未来金融。'},{type:'p',text:'传统金融交易的是资产。DeFi 将资产、流动性与金融规则带到了链上。而 FutureFi 希望进一步解决一个新的问题：如果“未来事件本身”也能够成为一种可以被市场表达、定价和验证的金融对象，会发生什么？'},{type:'list',items:['AI 人工智能 × Prediction Market 预测市场','DeFi 去中心化金融 × Oracle 预言机','WIN Chain 链上基础设施']},{type:'p',text:'它所交易的，不只是某一种传统数字资产，而是市场对于未来事件发生概率的判断。未来发生之前，市场已经存在不同判断；FutureFi 希望做的，就是让这些判断形成市场。'}]},
  { no:'03',en:'CORE PROJECT',title:'AIX：WIN 生态首个 AI Prediction Exchange 主打项目',blocks:[{type:'formula',text:'AIX — AI Prediction Exchange'},{type:'p',text:'它是 WIN 生态面向 FutureFi 预测金融方向打造的核心项目之一，也是云预测平台的重要资产与应用入口。AIX 的意义，不只是“多一个币”。'},{type:'list',items:['AI Prediction Engine — AI 预测引擎','Prediction Exchange — 预测交易平台','Oracle — 预言机','DeFi Liquidity — DeFi 流动性','WIN Chain Settlement — 链上结算']},{type:'p',text:'AIX 承担 AI 能力向预测金融场景延伸的角色，形成完整的预测金融应用体系。'}]},
  { no:'04',en:'CORE PAIR',title:'WIN / AIX：云链首个重点币币交易对',blocks:[{type:'formula',text:'WIN / AIX = 基础生态价值 × 预测金融价值的连接池'},{type:'p',text:'WIN 是 WIN Chain 整个生态的核心资产，而 AIX 则代表 FutureFi 云预测金融赛道。WIN/AIX 不只是普通交易对，而是生态应用资产与核心资产之间的重点流动性连接。'},{type:'flow',items:['WIN','AIX','FutureFi','Prediction Exchange','用户与流动性','WIN Chain']}]},
  { no:'05',en:'AI ENGINE',title:'AIX 云预测平台：让 AI 成为预测金融的智能引擎',blocks:[{type:'formula',text:'AI + 数据 + 概率 + 市场 + 预言机 + 链上结算'},{type:'p',text:'AIX 可以通过 AI Prediction Engine 对历史数据、市场数据、趋势数据以及其他可用信息进行分析，为预测市场提供辅助信息与概率参考。预测结果再通过 Oracle 等机制获取和验证外部世界的数据结果。'},{type:'flow',items:['现实世界事件','数据输入','AI Prediction Engine','预测市场形成概率与价格','用户表达判断','Oracle 获取结果','系统规则执行结算','WIN Chain 记录']}]},
  { no:'06',en:'PREDICTION EXCHANGE',title:'从“交易资产”走向“交易未来判断”',blocks:[{type:'p',text:'传统交易所主要解决：这个资产现在值多少钱？Prediction Exchange 希望进一步探索：这个事件未来发生的可能性是多少？'},{type:'quote',text:'过去，我们交易价格；今天，我们交易资产；FutureFi 希望让市场进一步交易“对未来的判断”。'}]},
  { no:'07',en:'ECOSYSTEM ROLE',title:'AIX 在整个 WIN 生态中的位置',blocks:[{type:'p',text:'如果把 WIN Chain 看成一座数字经济城市，每一个产品承担不同的基础功能。AIX 不是一个孤立项目，而是整个 WIN 数字经济生态向 AI 原生金融、预测市场和 FutureFi 延伸的重要组成部分。'},{type:'flow',items:['WIN Chain','WIN','BITWINEX','WIN Wallet / Chat / Pay','Win-Engine','WIN AI Ecosystem','AIX / FutureFi','LaunchHub / FIVE MEME']}]},
  { no:'08',en:'LAUNCHHUB',title:'LaunchHub × FIVE MEME：让更多项目进入 WIN 生态',blocks:[{type:'p',text:'LaunchHub 的战略意义，是为新的 Web3 项目提供项目启动、代币发行、生态孵化以及流动性连接入口。'},{type:'p',text:'最终，不同项目、资产、用户、AI Agent、开发者与流动性可以逐步进入同一套 WIN 数字经济生态。'}]},
  { no:'09',en:'BASE LAYER',title:'WIN Chain：FutureFi 背后的底层基础设施',blocks:[{type:'p',text:'FutureFi 要真正发展，最终仍然需要高性能区块链基础设施承载。WIN Chain 围绕 WIN 构建交易、支付、钱包、AI、云引擎等应用体系。'},{type:'formula',text:'WIN 是底层价值连接，AIX 是 FutureFi 应用价值延伸。'}]},
  { no:'10',en:'EVOLUTION',title:'从 WIN 到 AIX，从 Web3 到 FutureFi',blocks:[{type:'p',text:'AI 能不能成为链上金融的新生产力？预测市场能不能成为 Web3 的新金融场景？现实世界未来事件能不能通过数据、AI、Oracle 与市场机制形成新的链上价值表达？'},{type:'flow',items:['WIN Chain','WIN AI Ecosystem','AIX — AI Prediction Exchange','FutureFi — Future Finance']},{type:'p',text:'这条路线代表 WIN 生态从数字资产基础设施进一步迈向 AI 原生数字金融基础设施的升级。'}]},
  { no:'11',en:'STRATEGIC MAP',title:'WIN 生态战略版图',blocks:[{type:'flow',items:['WIN Chain 公链基础设施','WIN 生态核心资产','BITWINEX 交易与流动性中心','钱包 / 社交 / 支付 / 云引擎','WIN AI Ecosystem','AIX — AI Prediction Exchange','FutureFi 新赛道','WIN / AIX 核心交易对','LaunchHub / FIVE MEME','WIN 全球数字经济生态']}]}]

const sections = computed<Section[]>(() => locale.value === 'zh' ? zhSections : tm('futurefi.sections') as unknown as Section[])
const linkUrls = ['https://prediction-exchange-omega.vercel.app/','https://winchain.io','https://swap.winchain.win','https://m.ave.ai/token/0x193013574dacbd38bf26ecb654b3fd787b94d216-winchain','https://testnet.wallet.eoeo.info/06bx','https://testflight.apple.com/join/bMaXhH3h','https://testflight.apple.com/join/sKHmJBsa','https://chat.winchat.asia/install.html']
const links = computed(() => {
  const names = tm('futurefi.linkNames') as unknown as string[]
  return linkUrls.map((url, index) => ({ name: names[index], url }))
})
</script>

<style scoped>
.futurefi-page{--paper:#f5f7fa;--surface:#fff;--ink:#111827;--muted:#667085;--line:#d9dee7;--blue:#075cff;position:relative;left:50%;width:100vw;min-height:100vh;overflow:hidden;transform:translateX(-50%);background:var(--paper);color:var(--ink);word-break:normal;overflow-wrap:break-word;line-break:strict}.article-nav{display:flex;align-items:center;justify-content:space-between;box-sizing:border-box;max-width:1120px;height:68px;margin:auto;padding:0 28px;border-bottom:1px solid var(--line)}.article-nav a{display:flex;gap:8px;color:var(--ink);font-size:14px;font-weight:620;text-decoration:none}.article-nav>span,.eyebrow,.deck,.chapter-map,.chapter aside,.process-flow figcaption,.links header p,.closing>p{font-family:var(--aix-font-display);letter-spacing:.14em}.article-nav>span,.eyebrow{color:var(--blue);font-size:9px;font-weight:700}.article-hero{box-sizing:border-box;max-width:1120px;margin:auto;padding:100px 64px 72px;border-bottom:1px solid var(--line)}.eyebrow{margin:0 0 28px}.article-hero h1{display:flex;max-width:900px;flex-direction:column;gap:8px;margin:0;font-family:var(--aix-font-sans);font-size:clamp(48px,6vw,72px);font-weight:720;line-height:1.08;letter-spacing:-.05em}.article-hero h1>span{color:var(--ink)}.article-hero h1 em{display:flex;flex-wrap:wrap;gap:0 .28em;max-width:12em;color:var(--blue);font-style:normal}.article-hero h1 em span{white-space:nowrap}.deck{margin:34px 0 0;color:var(--muted);font-size:10px;font-weight:650}.article-hero blockquote{display:flex;align-items:baseline;justify-content:space-between;gap:24px;margin:80px 0 0;padding-top:22px;border-top:1px solid var(--line)}.article-hero blockquote strong{font-family:var(--aix-font-display);font-size:28px;letter-spacing:-.03em}.article-hero blockquote span{color:var(--muted);font-size:14px}.article-body{display:grid;grid-template-columns:130px minmax(0,760px);gap:64px;box-sizing:border-box;max-width:1120px;margin:auto;padding:96px 64px 120px}.chapter-map{position:sticky;top:24px;align-self:start;display:flex;flex-direction:column;border-top:2px solid var(--blue)}.chapter-map p{margin:0;padding:14px 0;color:var(--muted);font-size:8px}.chapter-map a{display:flex;gap:9px;padding:9px 0;border-bottom:1px solid var(--line);color:#77808f;font-size:8px;line-height:1.25;text-decoration:none}.chapter-map a span{color:var(--blue);font-weight:750}.reading-column{min-width:0}.opening{max-width:630px;margin:0 0 96px;padding-left:24px;border-left:2px solid var(--blue);color:#344054;font-size:17px;font-weight:470;line-height:1.95;letter-spacing:.005em}.chapter{display:grid;grid-template-columns:72px minmax(0,1fr);gap:34px;padding:72px 0;border-top:1px solid var(--line);scroll-margin-top:24px}.chapter aside{display:flex;flex-direction:column;gap:10px}.chapter aside span{color:var(--blue);font-size:22px;font-weight:750}.chapter aside small{color:#7b8494;font-size:7px;font-weight:700;line-height:1.45}.chapter h2{max-width:660px;margin:0 0 30px;font-family:var(--aix-font-sans);font-size:30px;font-weight:680;line-height:1.35;letter-spacing:-.025em;text-wrap:pretty}.chapter-content>p{max-width:650px;margin:0 0 22px;color:#4b5565;font-size:15px;font-weight:430;line-height:1.9}.chapter blockquote{max-width:620px;margin:32px 0;padding:22px 0 22px 24px;border-left:2px solid var(--blue);color:#1f2937;font-size:17px;font-weight:580;line-height:1.75}.formula{max-width:620px;margin:28px 0;padding:15px 18px;border:1px solid rgba(7,92,255,.2);background:rgba(7,92,255,.035);color:var(--blue);font-family:var(--aix-font-display);font-size:14px;font-weight:680;line-height:1.65}.feature-list{max-width:620px;margin:30px 0;padding:0;border-top:1px solid var(--line);list-style:none}.feature-list li{display:flex;align-items:baseline;gap:20px;padding:17px 0;border-bottom:1px solid var(--line);font-size:15px;line-height:1.6}.feature-list b{flex:0 0 24px;color:var(--blue);font-family:var(--aix-font-display);font-size:9px}.process-flow{max-width:650px;margin:38px 0 44px}.process-flow figcaption{display:flex;justify-content:space-between;padding-bottom:14px;border-bottom:1px solid var(--line);color:var(--blue);font-size:8px;font-weight:700}.process-flow figcaption small{color:#8992a1;font:inherit}.process-flow ol{position:relative;display:flex;margin:0;padding:34px 0 10px;list-style:none}.process-flow ol::before{content:'';position:absolute;top:41px;right:0;left:0;height:1px;background:var(--blue)}.process-flow li{position:relative;z-index:1;display:flex;min-width:0;flex:1;flex-direction:column;padding-right:12px}.process-flow li i{box-sizing:border-box;width:15px;height:15px;margin-bottom:18px;border:4px solid var(--paper);border-radius:50%;background:var(--blue);box-shadow:0 0 0 1px var(--blue)}.process-flow li small{order:-1;height:18px;color:var(--blue);font-family:var(--aix-font-display);font-size:8px;font-weight:750}.process-flow li strong{max-width:10em;color:#293241;font-size:12px;font-weight:650;line-height:1.5;text-wrap:pretty}.process-flow li.is-terminal{align-items:flex-end;padding-right:0;text-align:right}.process-flow li.is-terminal i{border-radius:2px}.process-flow li.is-terminal strong{color:var(--blue)}.closing{margin-top:80px;padding:58px 54px;background:#101828;color:#fff}.closing>p{margin:0 0 26px;color:#80a9ff;font-size:8px;font-weight:700}.closing h2{margin:0 0 44px;font-family:var(--aix-font-display);font-size:48px;line-height:1.05;letter-spacing:-.045em}.closing>div{display:grid;grid-template-columns:1fr 1fr;gap:34px;padding-top:28px;border-top:1px solid #344054}.closing>div p{margin:0;color:#cbd2dc;font-size:14px;line-height:1.85}.closing>strong{display:block;margin-top:44px;color:#80a9ff;font-size:22px}.links{margin-top:96px}.links header{margin-bottom:24px}.links header p{margin:0 0 10px;color:var(--blue);font-size:8px}.links h2{margin:0;font-size:28px}.links a{display:grid;grid-template-columns:32px minmax(180px,1fr) 1.3fr 20px;gap:16px;align-items:center;padding:18px 0;border-top:1px solid var(--line);color:var(--ink);text-decoration:none}.links a:last-child{border-bottom:1px solid var(--line)}.links b{color:var(--blue);font-size:9px}.links span{font-size:14px;font-weight:650}.links small{overflow:hidden;color:var(--muted);font-size:11px;text-overflow:ellipsis;white-space:nowrap}.links i{font-style:normal}
@media(max-width:920px){.article-hero{padding-inline:48px}.article-body{grid-template-columns:1fr;gap:56px;padding-inline:48px}.chapter-map{position:relative;top:auto;overflow-x:auto;flex-direction:row;border-top-width:1px}.chapter-map p{display:none}.chapter-map a{flex:0 0 auto;width:92px}.opening{margin-bottom:76px}}
@media(max-width:759px){.article-nav{height:60px;padding:0 40px}.article-hero{padding:72px 48px 58px}.article-hero h1{gap:10px;max-width:600px;font-size:40px;line-height:1.14;letter-spacing:-.035em}.article-hero h1 em{display:flex;max-width:none;text-align:left;word-spacing:0}.article-hero blockquote{margin-top:58px}.article-hero blockquote strong{font-size:25px}.article-body{gap:48px;padding:72px 48px 104px}.chapter-map{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));overflow:visible}.chapter-map a{box-sizing:border-box;width:auto;min-width:0;padding:12px 8px}.chapter-map a:nth-child(n+10){display:none}.opening{max-width:600px;margin-bottom:72px;padding-left:24px;font-size:16px;line-height:1.9}.chapter{grid-template-columns:72px minmax(0,1fr);gap:34px;padding:68px 0}.chapter aside span{font-size:19px}.chapter h2{max-width:22em;margin-bottom:30px;font-size:27px;line-height:1.38;letter-spacing:-.018em;text-wrap:pretty}.chapter-content>p{max-width:650px;font-size:16px;line-height:1.9}.feature-list,.formula,.chapter blockquote{max-width:650px}.process-flow{max-width:650px;margin-block:38px}.process-flow ol{display:flex;flex-direction:column;padding:24px 0 0}.process-flow ol::before{top:28px;bottom:16px;left:7px;width:1px;height:auto}.process-flow li,.process-flow li.is-terminal{box-sizing:border-box;align-items:flex-start;min-height:62px;padding:0 0 22px 38px;text-align:left}.process-flow li i{position:absolute;top:1px;left:0;margin:0}.process-flow li small{order:0;height:auto;margin-bottom:6px}.process-flow li strong,.process-flow li.is-terminal strong{max-width:none;font-size:15px;text-align:left}.closing{margin-top:64px;padding:52px 44px}.closing h2{font-size:40px}.closing>div{grid-template-columns:1fr}.links{margin-top:80px}.links a{grid-template-columns:30px minmax(0,1fr) 20px}.links small{grid-column:2/-1;grid-row:2}}
@media(max-width:479px){.article-nav,.article-hero,.article-body{padding-inline:22px}.article-hero h1{font-size:34px}.article-hero h1 em{gap:0 .2em;white-space:normal}.article-body{padding-top:56px}.chapter{grid-template-columns:1fr;gap:18px}.chapter aside{flex-direction:row;align-items:center}.chapter aside small{max-width:none}.chapter h2{font-size:24px}.closing{margin-inline:-22px;padding-inline:22px}.closing h2{font-size:33px}}

/* Conclusion and official access */
.closing{position:relative;isolation:isolate;overflow:hidden;background:#111827;color:#f5f7fa;border:1px solid rgba(102,112,133,.4)}
.closing::after{position:absolute;z-index:-1;right:-18%;bottom:-42%;width:58%;aspect-ratio:1;content:"";border:1px solid rgba(7,92,255,.28);border-radius:50%}
.closing>p{color:#075cff}
.closing h2{color:#f5f7fa!important;text-shadow:none}
.closing>div{border-top-color:rgba(245,247,250,.2)}
.closing>div p{color:#f5f7fa}
.closing>strong{color:#075cff}
.links{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;margin-top:96px;padding-top:0;border-top:0}
.links header{display:flex;grid-column:1/-1;align-items:flex-end;justify-content:space-between;gap:24px;margin-bottom:18px;padding-bottom:24px;border-bottom:1px solid #d9dee7}
.links header p{margin:0;color:#075cff;font-size:10px;font-weight:700;letter-spacing:.16em}
.links header h2{margin:0;color:#111827;font-family:var(--aix-font-sans);font-size:clamp(28px,4vw,42px);line-height:1.15;letter-spacing:-.035em}
.links a{display:grid;grid-template-columns:34px minmax(0,1fr) 22px;grid-template-rows:auto auto;gap:8px 12px;box-sizing:border-box;min-height:132px;padding:22px;background:#fff;border:1px solid #d9dee7;color:#111827;text-decoration:none;transition:border-color .2s ease,transform .2s ease,box-shadow .2s ease}
.links a:hover{transform:translateY(-3px);border-color:#075cff;box-shadow:0 14px 30px rgba(17,24,39,.08)}
.links a b{grid-row:1/3;color:#075cff;font-family:var(--aix-font-display);font-size:11px;letter-spacing:.1em}
.links a span{align-self:start;font-size:16px;font-weight:700;line-height:1.45}
.links a small{align-self:end;overflow:hidden;color:#667085;font-size:11px;line-height:1.4;text-overflow:ellipsis;white-space:nowrap}
.links a i{grid-column:3;grid-row:1/3;align-self:start;color:#075cff;font-size:18px;font-style:normal;text-align:right}
@media(max-width:759px){.closing{padding:48px 38px}.closing h2{font-size:38px}.closing>div{grid-template-columns:1fr}.links{grid-template-columns:1fr;margin-top:82px}.links header{align-items:flex-start;flex-direction:column-reverse;gap:10px;margin-bottom:12px}.links a{grid-template-columns:32px minmax(0,1fr) 22px;min-height:116px;padding:20px}.links small{grid-column:2;grid-row:2}}
@media(max-width:479px){.closing{padding-inline:22px}.closing h2{font-size:32px}.links a{min-height:108px;padding:18px}.links header h2{font-size:30px}}

/* Mobile layout: keep every section in normal flow and give long labels room. */
@media(max-width:759px){
  .futurefi-page{left:0;width:100%;transform:none;overflow-x:clip}
  .article-nav{padding-inline:24px}
  .article-nav>span{font-size:8px;letter-spacing:.1em}
  .article-hero{padding:64px 24px 48px}
  .article-hero h1{font-size:clamp(32px,9vw,42px);line-height:1.12}
  .article-hero h1 em{display:block;max-width:none}
  .article-hero h1 em span{display:block;width:max-content;max-width:100%;white-space:normal;text-align:left;font-family:var(--aix-font-sans);letter-spacing:-.035em;word-spacing:normal}
  .deck{line-height:1.7;letter-spacing:.1em}
  .article-hero blockquote{align-items:flex-start;flex-direction:column;gap:8px;margin-top:44px}
  .article-hero blockquote strong{font-size:23px;line-height:1.25}
  .article-body{display:block;padding:48px 24px 80px}
  .chapter-map{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:0;overflow:visible;border-top:1px solid var(--blue);border-bottom:1px solid var(--line)}
  .chapter-map p{display:none}
  .chapter-map a,.chapter-map a:nth-child(n+10){display:flex;align-items:flex-start;gap:10px;box-sizing:border-box;width:auto;min-width:0;padding:14px 8px;border-bottom:1px solid var(--line);font-size:9px;line-height:1.35;letter-spacing:.12em;white-space:normal;overflow-wrap:anywhere}
  .chapter-map a:nth-last-child(-n+2){border-bottom:0}
  .chapter-map a span{flex:0 0 auto;font-size:10px}
  .opening{box-sizing:border-box;margin:56px 0 36px;padding-left:18px;font-size:15px;line-height:1.85}
  .chapter{display:block;padding:50px 0}
  .chapter aside{display:flex;align-items:baseline;flex-direction:row;gap:12px;margin-bottom:20px}
  .chapter aside span{font-size:18px;line-height:1}
  .chapter aside small{max-width:none;font-size:9px;line-height:1.4;letter-spacing:.12em}
  .chapter h2{margin-bottom:24px;font-size:clamp(23px,6.5vw,28px);line-height:1.38;overflow-wrap:anywhere}
  .chapter-content>p{font-size:15px;line-height:1.85}
  .chapter-content>p+p{margin-top:18px}
  .formula{padding:15px 16px;font-size:13px;line-height:1.65;overflow-wrap:anywhere}
  .feature-list li{grid-template-columns:30px minmax(0,1fr);gap:12px;padding:14px 0;font-size:14px;line-height:1.55}
  .chapter blockquote{margin-block:24px;padding-left:16px;font-size:17px;line-height:1.65}
  .process-flow{margin-block:30px}
  .process-flow figcaption{align-items:flex-start;gap:6px;flex-direction:column;padding-bottom:12px;line-height:1.5}
  .process-flow ol{display:flex!important;flex-direction:column!important;padding-top:20px!important}
  .process-flow li,.process-flow li.is-terminal{position:relative;display:flex!important;align-items:flex-start!important;flex-direction:column!important;box-sizing:border-box;width:100%!important;min-height:0!important;padding:0 0 22px 34px!important;text-align:left!important}
  .process-flow li i{position:absolute!important;top:3px!important;left:0!important}
  .process-flow li small{height:auto!important;margin:0 0 4px!important;line-height:1.2!important}
  .process-flow li strong,.process-flow li.is-terminal strong{max-width:100%!important;font-size:14px!important;line-height:1.5!important;text-align:left!important;overflow-wrap:anywhere}
  .closing{margin:48px 0 0;padding:38px 24px}
  .closing h2{font-size:clamp(30px,9vw,38px);line-height:1.12;overflow-wrap:normal}
  .closing>div{display:block;padding-top:28px}
  .closing>div p{font-size:15px;line-height:1.8}
  .closing>div p+p{margin-top:18px}
  .closing>strong{display:block;font-size:20px;line-height:1.5}
  .links{display:block;margin-top:64px}
  .links header{display:flex;margin-bottom:12px}
  .links a{margin-top:10px}
}
@media(max-width:359px){
  .article-nav a span{display:none}
  .chapter-map{grid-template-columns:1fr}
  .chapter-map a:nth-last-child(-n+2){border-bottom:1px solid var(--line)}
  .chapter-map a:last-child{border-bottom:0}
}
</style>
