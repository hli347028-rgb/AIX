<template>
  <HomeStatic v-if="useStaticHome" />
  <component :is="cinematicHome" v-else />
</template>

<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
import { resolveHomeMode } from '@/tools/homeMode'
import HomeStatic from './HomeStatic.vue'

const useStaticHome = resolveHomeMode() === 'static'
const cinematicHome = useStaticHome
  ? null
  : defineAsyncComponent(() => import('./HomeCinematic.vue'))
</script>
