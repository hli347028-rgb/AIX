<template>
  <div v-if="mode === 'pending'" class="home-probe" aria-hidden="true" />
  <HomeStatic v-else-if="mode === 'static'" />
  <component :is="cinematicHome" v-else @render-failed="onCinematicFailed" />
</template>

<script setup lang="ts">
import { defineAsyncComponent, onMounted, ref } from 'vue'
import {
  persistHomeMode,
  probeCinematicReady,
  resolveHomeMode,
  shouldProbeCinematic,
  type HomeMode,
} from '@/tools/homeMode'
import HomeStatic from './HomeStatic.vue'

const mode = ref<HomeMode | 'pending'>('pending')
const cinematicHome = defineAsyncComponent(() => import('./HomeCinematic.vue'))

const onCinematicFailed = () => {
  persistHomeMode('static')
  mode.value = 'static'
}

onMounted(async () => {
  const resolved = resolveHomeMode()
  if (resolved === 'static') {
    mode.value = 'static'
    return
  }
  if (shouldProbeCinematic(resolved)) {
    const ok = await probeCinematicReady()
    if (!ok) {
      persistHomeMode('static')
      mode.value = 'static'
      return
    }
  }
  mode.value = 'cinematic'
})
</script>

<style scoped>
.home-probe {
  min-height: 100vh;
  background: #020817;
}
</style>
