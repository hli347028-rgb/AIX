<template>
  <div class="announcements-page">
    <van-nav-bar
      :title="$t('announcement.title')"
      left-arrow
      :border="false"
      fixed
      @click-left="router.back()"
    />

    <main class="page-main">
      <div v-if="loading" class="state-box">
        <van-loading color="#1597e5" />
      </div>

      <div v-else-if="list.length === 0" class="empty-state">
        <p>{{ $t('announcement.empty') }}</p>
      </div>

      <div v-else class="announcement-list">
        <article
          v-for="item in list"
          :key="item.id"
          class="announcement-item"
          :class="{ open: expandedId === item.id }"
          @click="toggle(item.id)"
        >
          <div class="item-head">
            <h3>{{ item.title }}</h3>
            <span class="item-time">{{ item.created_at || formatTime(item.add_time) }}</span>
          </div>
          <div v-show="expandedId === item.id" class="item-body" v-html="item.content"></div>
        </article>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { showToast } from 'vant'
import { errMsg, listAnnouncements, type AnnouncementItem } from '@/api/aix'

const router = useRouter()
const { t: $t } = useI18n()

const loading = ref(true)
const list = ref<AnnouncementItem[]>([])
const expandedId = ref<number | null>(null)

function formatTime(ts?: number) {
  if (!ts) return ''
  const date = new Date(ts < 1e12 ? ts * 1000 : ts)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function toggle(id?: number) {
  if (!id) return
  expandedId.value = expandedId.value === id ? null : id
}

onMounted(async () => {
  try {
    const res = await listAnnouncements({ page: 1, page_size: 50 })
    list.value = res.list || []
    if (list.value.length > 0) {
      expandedId.value = list.value[0].id ?? null
    }
  } catch (e) {
    showToast(errMsg(e, $t('announcement.fetchFailed')))
  } finally {
    loading.value = false
  }
})
</script>

<style scoped lang="scss">
@use '@/style/variables.scss' as *;

.announcements-page {
  min-height: 100vh;
  background: #061018;
  color: #fff;
}

.page-main {
  padding: 64px 16px 32px;
}

.state-box,
.empty-state {
  display: flex;
  justify-content: center;
  padding: 48px 0;
  color: rgba(255, 255, 255, 0.55);
}

.announcement-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.announcement-item {
  padding: 16px;
  border-radius: 14px;
  background: linear-gradient(145deg, rgba(13, 27, 42, 0.95), rgba(8, 19, 30, 0.98));
  border: 1px solid rgba(21, 151, 229, 0.22);
  cursor: pointer;
  transition: border-color 0.2s ease;

  &.open,
  &:hover {
    border-color: rgba(21, 151, 229, 0.55);
  }

  .item-head {
    display: flex;
    flex-direction: column;
    gap: 8px;

    h3 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      color: #fff;
      line-height: 1.4;
    }

    .item-time {
      font-size: 12px;
      color: rgba(255, 255, 255, 0.45);
    }
  }

  .item-body {
    margin-top: 14px;
    padding-top: 14px;
    border-top: 1px solid rgba(255, 255, 255, 0.08);
    font-size: 14px;
    line-height: 1.7;
    color: rgba(255, 255, 255, 0.82);
    word-break: break-word;

    :deep(img) {
      max-width: 100%;
      height: auto;
    }

    :deep(p) {
      margin: 0 0 10px;
    }

    :deep(a) {
      color: $brand-primary;
    }
  }
}
</style>
