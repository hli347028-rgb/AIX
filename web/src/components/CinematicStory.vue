<template>
  <section ref="storyRef" class="story" aria-label="AIX 未来交易场景">
    <div class="stage">
      <div class="stage-bg" aria-hidden="true"></div>
      <div class="light-sweep" aria-hidden="true"></div>

      <div class="terminal" aria-hidden="true">
        <img src="/assets/aix-terminal-cinematic.png" alt="" />
        <div class="screen">
          <div class="screen-noise"></div>
          <div class="screen-content screen-content--boot">
            <small>AIX PREDICTION NETWORK</small>
            <strong>FUTURE<br />IS LIQUID</strong>
            <i></i>
          </div>
          <div class="screen-content screen-content--market">
            <header><span>GLOBAL SIGNALS</span><b>LIVE</b></header>
            <div v-for="item in signals" :key="item.symbol" class="signal-row">
              <b>{{ item.symbol }}</b><span><i :style="{ '--w': item.width }"></i></span><em>{{ item.price }}</em>
            </div>
          </div>
          <div class="screen-content screen-content--oracle">
            <span class="oracle-eye"></span>
            <small>ORACLE CONSENSUS</small>
            <strong>97.42%</strong>
            <p>未来信号已完成定价</p>
          </div>
        </div>
      </div>

      <div class="boot-caption">
        <small>AIX PREDICTION EXCHANGE</small>
        <span>正在接入未来信号</span>
      </div>

      <div class="film-rail" aria-hidden="true">
        <div class="film-track">
          <span v-for="frame in 10" :key="frame"><i>{{ String(frame).padStart(2, '0') }}</i></span>
        </div>
      </div>
      <div class="progress" aria-hidden="true"><i></i><span>SCROLL TO EXPLORE</span></div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const storyRef = ref<HTMLElement | null>(null)
const signals = [
  { symbol: 'BTC', price: '78,260', width: '74%' },
  { symbol: 'AIX', price: '2.0340', width: '88%' },
  { symbol: 'ETH', price: '2,455.8', width: '61%' },
  { symbol: 'XAU', price: '4,529.90', width: '48%' },
]
let frame = 0
const update = () => {
  cancelAnimationFrame(frame)
  frame = requestAnimationFrame(() => {
    if (!storyRef.value) return
    const rect = storyRef.value.getBoundingClientRect()
    const distance = Math.max(1, rect.height - innerHeight)
    const progress = Math.min(1, Math.max(0, -rect.top / distance))
    storyRef.value.style.setProperty('--story', progress.toFixed(4))
  })
}
onMounted(() => {
  addEventListener('scroll', update, { passive: true })
  addEventListener('resize', update, { passive: true })
  update()
})
onBeforeUnmount(() => {
  cancelAnimationFrame(frame)
  removeEventListener('scroll', update)
  removeEventListener('resize', update)
})
</script>

<style scoped>
.story{--story:0;position:relative;height:260vh;margin-top:-1px;background:transparent;color:#f5f9ff}.story::before,.story::after{content:'';position:absolute;z-index:10;left:0;right:0;height:24vh;pointer-events:none}.story::before{top:0;background:linear-gradient(180deg,#020817,transparent)}.story::after{bottom:0;background:linear-gradient(180deg,transparent,#020817)}.stage{position:sticky;top:0;height:100vh;min-height:620px;overflow:hidden;isolation:isolate;background:transparent;perspective:1400px;opacity:calc(1 - max(0,(var(--story) - .82) * 5.56));transform:scale(calc(1 + max(0,(var(--story) - .78)) * .18));transform-origin:center;will-change:opacity,transform}.stage-bg{position:absolute;z-index:-3;inset:-7%;background:linear-gradient(90deg,rgba(2,8,23,.28),rgba(2,8,23,.06) 52%,rgba(2,8,23,.54)),url('/assets/aix-terminal-cinematic.png') center/cover no-repeat;filter:brightness(.72) saturate(1.08);transform:translate3d(calc((var(--story) - .5) * -4vw),calc(var(--story) * -3vh),-80px) scale(1.13);will-change:transform}.stage::after{content:'';position:absolute;z-index:8;inset:0;pointer-events:none;background:linear-gradient(180deg,rgba(2,8,23,.18),transparent 18%,transparent 78%,rgba(2,8,23,.82)),radial-gradient(ellipse at center,transparent 40%,rgba(2,8,23,.62) 100%)}.light-sweep{position:absolute;z-index:-1;inset:-30%;background:linear-gradient(110deg,transparent 38%,rgba(69,166,255,.14) 48%,transparent 58%);transform:translateX(calc((var(--story) - .5) * 40%)) rotate(-4deg)}.terminal{position:absolute;z-index:2;inset:0;transform:translate3d(calc((var(--story) - .45) * -2.5vw),calc((var(--story) - .5) * 1.5vh),120px);will-change:transform}.terminal>img{display:none}.screen{position:absolute;right:14%;top:31%;width:25%;height:29%;overflow:hidden;border-radius:3%;background:rgba(1,8,18,.84);transform:perspective(800px) rotateY(-2deg) scale(calc(.92 + var(--story) * .13));box-shadow:0 0 0 1px rgba(75,161,235,.09),inset 0 0 32px rgba(61,162,255,.14);mix-blend-mode:screen}.screen::after{content:'';position:absolute;inset:0;background:repeating-linear-gradient(0deg,rgba(255,255,255,.022) 0 1px,transparent 1px 4px);mix-blend-mode:screen}.screen-noise{position:absolute;inset:0;opacity:.15;background-image:url("data:image/svg+xml,%3Csvg viewBox='0 0 120 120' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence baseFrequency='.9' numOctaves='3'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E")}.screen-content{position:absolute;inset:0;display:flex;flex-direction:column;justify-content:center;padding:8%;font-family:var(--aix-font-display);transition:opacity .15s linear;will-change:opacity}.screen-content--boot{align-items:center;text-align:center;opacity:clamp(0,calc(1 - var(--story) * 4),1)}.screen-content--boot small,.screen-content--oracle small{font-size:clamp(5px,.58vw,9px);letter-spacing:.2em;color:#67bdff}.screen-content--boot strong{margin-top:5%;font-size:clamp(14px,2.1vw,34px);line-height:.9;letter-spacing:.03em}.screen-content--boot i{width:38%;height:2px;margin-top:7%;background:#178cff;box-shadow:0 0 15px #178cff}.screen-content--market{gap:7%;opacity:clamp(0,calc((var(--story) - .18) * 5),calc((.72 - var(--story)) * 6))}.screen-content--market header,.signal-row{display:flex;align-items:center;gap:5%;font-size:clamp(5px,.62vw,10px)}.screen-content--market header{justify-content:space-between;color:#6fbfff;letter-spacing:.14em}.screen-content--market header b{color:#50dca2}.signal-row>b{width:11%;color:#d7eaff}.signal-row>span{height:2px;flex:1;background:rgba(111,191,255,.12)}.signal-row>span i{display:block;width:var(--w);height:100%;background:#36a8ff;box-shadow:0 0 8px #36a8ff}.signal-row em{width:20%;color:#9ab2ce;font-style:normal;text-align:right}.screen-content--oracle{align-items:center;text-align:center;opacity:clamp(0,calc((var(--story) - .68) * 5),1)}.screen-content--oracle strong{font-size:clamp(18px,3.4vw,54px);color:#f5f9ff}.screen-content--oracle p{font-size:clamp(6px,.7vw,11px);color:#8fa7c2}.oracle-eye{width:14%;aspect-ratio:1;margin-bottom:5%;border:1px solid #47b1ff;border-radius:50%;box-shadow:0 0 22px rgba(71,177,255,.5),inset 0 0 12px rgba(71,177,255,.4)}.story-copy{position:absolute;z-index:5;top:50%;width:min(34vw,470px);transform:translateY(-50%);will-change:opacity,transform}.story-copy p{margin:0 0 20px;font:600 10px/1 var(--aix-font-display);letter-spacing:.22em;color:#66bdff}.story-copy h2{margin:0;color:#f5f9ff;font-family:"PingFang SC","Microsoft YaHei",var(--aix-font-sans);font-size:clamp(42px,5.3vw,78px);font-weight:700;line-height:1.02;letter-spacing:-.055em;text-wrap:balance;text-shadow:0 16px 50px rgba(0,4,18,.72)}.story-copy span{display:block;max-width:390px;margin-top:25px;color:#a7b6ca;font-size:15px;line-height:1.8}.story-copy--one{left:6%;opacity:clamp(0,calc(1 - var(--story) * 4),1);transform:translate3d(calc(var(--story) * -80px),-50%,60px)}.story-copy--two{right:6%;opacity:clamp(0,calc((var(--story) - .24) * 5),calc((.7 - var(--story)) * 6));transform:translate3d(calc((.48 - var(--story)) * 100px),-50%,80px);text-align:right}.story-copy--two span{margin-left:auto}.story-copy--three{left:7%;opacity:clamp(0,calc((var(--story) - .72) * 5),1);transform:translate3d(calc((.82 - var(--story)) * -80px),-50%,80px)}.film-rail{position:absolute;z-index:6;left:0;right:0;bottom:2.5vh;height:72px;overflow:hidden;opacity:.72;mask-image:linear-gradient(90deg,transparent,#000 8%,#000 92%,transparent)}.film-track{display:flex;width:max-content;gap:8px;transform:translateX(calc(var(--story) * -42vw));will-change:transform}.film-track span{position:relative;width:150px;height:66px;border:1px solid rgba(116,185,255,.2);background:rgba(5,17,40,.58);backdrop-filter:blur(8px)}.film-track span::before,.film-track span::after{content:'';position:absolute;left:6px;right:6px;height:5px;background:repeating-linear-gradient(90deg,rgba(104,181,255,.5) 0 8px,transparent 8px 16px)}.film-track span::before{top:5px}.film-track span::after{bottom:5px}.film-track i{position:absolute;right:10px;bottom:14px;color:#5baeff;font:600 8px/1 var(--aix-font-display);font-style:normal}.progress{position:absolute;z-index:7;right:3%;top:50%;display:flex;align-items:center;gap:12px;writing-mode:vertical-rl}.progress i{width:2px;height:130px;background:linear-gradient(to bottom,#35a7ff calc(var(--story) * 100%),rgba(127,181,245,.17) 0)}.progress span{font:600 8px/1 var(--aix-font-display);letter-spacing:.16em;color:#7791af}@media(max-width:759px){.story{height:340vh}.stage{min-height:600px}.terminal{inset:0;transform:translate3d(calc((var(--story) - .4) * -2vw),calc(var(--story) * 1vh),100px)}.screen{right:5%;top:35%;width:27%;height:21%;background:rgba(1,8,18,.68)}.story-copy{top:auto;left:24px!important;right:48px!important;bottom:17vh;width:auto;text-align:left!important}.story-copy h2{font-size:clamp(37px,12vw,62px)}.story-copy span{margin-left:0!important;font-size:14px}.film-rail{height:58px}.film-track span{width:112px;height:52px}.progress{right:12px}.stage-bg{background-position:58% center}}@media(prefers-reduced-motion:reduce){.story{height:auto}.stage{position:relative}.terminal{transform:translate3d(-50%,-50%,0) scale(.85)}.story-copy--one{opacity:1;transform:translateY(-50%)}.story-copy--two,.story-copy--three,.film-rail,.progress{display:none}}
.boot-caption{position:absolute;z-index:9;left:6vw;top:16vh;display:flex;flex-direction:column;gap:10px;opacity:calc(1 - min(1,var(--story) * 4));transform:translate3d(0,calc(var(--story) * -24px),80px)}
.boot-caption small{color:#52b5ff;font:700 11px/1 var(--aix-font-display);letter-spacing:.22em}
.boot-caption span{color:#d9e9ff;font-size:14px;letter-spacing:.08em}
@media(max-width:759px){.story{height:250vh}.boot-caption{left:24px;top:14vh}}
</style>
