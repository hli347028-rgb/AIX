<template>
  <li class="team-node">
    <article class="team-node-card">
      <button
        class="toggle-btn"
        type="button"
        :disabled="!hasChildren"
        :aria-expanded="expanded"
        :aria-label="expanded ? $t('common.collapseSubordinates') : $t('common.expandSubordinates')"
        @click="toggle"
      >
        <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" :class="{ expanded }">
          <path d="m8 10 4 4 4-4" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>

      <div class="node-content">
        <div class="node-topline">
          <strong>{{ node.username || usernamePlaceholder }}</strong>
          <div class="node-tags">
            <span>{{ teamCountLabel }}：{{ formatCount(node.teamCount) }}</span>
            <span>{{ directCountLabel }}：{{ formatCount(node.directCount) }}</span>
          </div>
        </div>
        <div class="node-address">
          <code :title="node.address">{{ formatAddress(node.address) || notSetLabel }}</code>
          <button v-if="node.address" type="button" class="copy-btn" :aria-label="copyLabel" @click="$emit('copy', node.address)">
            <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <rect x="8" y="8" width="11" height="11" rx="2" stroke="currentColor" stroke-width="1.6" />
              <path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2" stroke="currentColor" stroke-width="1.6" />
            </svg>
          </button>
        </div>
        <dl class="node-performance">
          <div><dt>{{ personalPerformanceLabel }}</dt><dd>{{ formatPerformance(node.personalPerformance) }}</dd></div>
          <div><dt>{{ teamPerformanceLabel }}</dt><dd>{{ formatPerformance(node.teamPerformance) }}</dd></div>
        </dl>
      </div>
    </article>

    <div v-if="expanded" class="children-wrap">
      <p v-if="loading" class="node-state">{{ loadingLabel }}</p>
      <p v-else-if="node.childrenLoaded && !node.children?.length" class="node-state">{{ emptyLabel }}</p>
      <ul v-else-if="node.children?.length" class="team-branch">
        <TeamTreeNode
          v-for="child in node.children"
          :key="child.address"
          :node="child"
          :load-children="loadChildren"
          :team-count-label="teamCountLabel"
          :direct-count-label="directCountLabel"
          :personal-performance-label="personalPerformanceLabel"
          :team-performance-label="teamPerformanceLabel"
          :copy-label="copyLabel"
          :username-placeholder="usernamePlaceholder"
          :not-set-label="notSetLabel"
          :loading-label="loadingLabel"
          :empty-label="emptyLabel"
          @copy="$emit('copy', $event)"
        />
      </ul>
    </div>
  </li>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t: $t } = useI18n()
const props = defineProps<{
  node: any
  loadChildren: (node: any) => Promise<void>
  teamCountLabel: string
  directCountLabel: string
  personalPerformanceLabel: string
  teamPerformanceLabel: string
  copyLabel: string
  usernamePlaceholder: string
  notSetLabel: string
  loadingLabel: string
  emptyLabel: string
}>()

defineEmits<{ copy: [address: string] }>()

const expanded = ref(false)
const loading = ref(false)
const hasChildren = computed(() => Number(props.node.directCount || 0) > 0 || props.node.hasChildren === true)

const formatCount = (value: any) => Math.max(0, Number(value || 0)).toLocaleString()
const formatPerformance = (value: any) => Number(value || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const formatAddress = (value: string) => value ? `${value.slice(0, 8)}...${value.slice(-7)}` : ''

const toggle = async () => {
  if (!hasChildren.value) return
  expanded.value = !expanded.value
  if (!expanded.value || props.node.childrenLoaded) return
  loading.value = true
  try {
    await props.loadChildren(props.node)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.team-node { list-style: none; }
.team-node-card { display: flex; gap: 12px; padding: 14px; border: 1px solid var(--hair); border-radius: var(--r-sm); background: rgba(255,255,255,.02); }
.toggle-btn { flex: 0 0 auto; display: grid; place-items: center; width: 34px; height: 34px; padding: 0; border: 1px solid var(--hair-2); border-radius: var(--r-sm); background: transparent; color: var(--text-3); cursor: pointer; }
.toggle-btn:disabled { opacity: .32; cursor: default; }
.toggle-btn svg { width: 18px; transition: transform .2s ease; }
.toggle-btn svg.expanded { transform: rotate(180deg); }
.node-content { min-width: 0; flex: 1; }
.node-topline { display: flex; align-items: flex-start; justify-content: space-between; gap: 10px; }
.node-topline strong { min-width: 0; color: var(--text); font-size: 15px; overflow-wrap: anywhere; }
.node-tags { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 6px; }
.node-tags span { padding: 5px 7px; border: 1px solid var(--hair-2); border-radius: 999px; color: var(--text-2); font-size: 10px; white-space: nowrap; }
.node-address { display: flex; align-items: center; gap: 9px; margin-top: 12px; }
.node-address code { min-width: 0; color: var(--text-3); font-size: 13px; }
.copy-btn { display: grid; flex: 0 0 auto; place-items: center; width: 28px; height: 28px; padding: 0; border: 0; background: transparent; color: var(--accent); cursor: pointer; }
.copy-btn svg { width: 18px; }
.node-performance { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; margin: 12px 0 0; }
.node-performance > div { min-width: 0; padding: 9px 10px; border: 1px solid var(--hair); border-radius: var(--r-sm); background: rgba(255,255,255,.015); }
.node-performance dt { color: var(--text-3); font-size: 10px; }
.node-performance dd { margin: 5px 0 0; color: var(--text); font-size: 13px; font-weight: 600; overflow-wrap: anywhere; }
.children-wrap { position: relative; margin: 8px 0 0 16px; padding-left: 14px; border-left: 1px solid var(--hair-2); }
.team-branch { display: grid; gap: 8px; margin: 0; padding: 0; }
.node-state { margin: 0; padding: 12px; color: var(--text-3); font-size: 12px; }
@media (max-width: 390px) {
  .node-topline { flex-direction: column; }
  .node-tags { justify-content: flex-start; }
  .children-wrap { margin-left: 10px; padding-left: 10px; }
}
</style>
