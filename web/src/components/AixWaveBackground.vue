<template>
  <div class="ambient" aria-hidden="true">
    <span class="soft-plane"></span>
  </div>
</template>

<style scoped>
.ambient {
  position: absolute;
  inset: 0;
  z-index: -1;
  overflow: hidden;
  pointer-events: none;
  background: #f7f8fa;
}
.soft-plane {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 44%;
  height: 104px;
  background: #eef1f5;
  clip-path: polygon(22% 0,100% 0,100% 100%,0 100%);
  transform-origin: right bottom;
  animation: plane-deploy .9s .3s cubic-bezier(.16,1,.3,1) both;
}
@keyframes plane-deploy {
  from { opacity: 0; transform: translateX(18%) scaleX(.72); }
}
@media (max-width: 759px) {
  .soft-plane { width: 58%; height: 88px; }
}
@media (prefers-reduced-motion: reduce) {
  .soft-plane { animation: none; }
}
</style>
