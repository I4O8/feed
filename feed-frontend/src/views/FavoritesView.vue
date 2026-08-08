<template>
  <div class="page favorites-page">
    <header class="topbar">
      <button type="button" class="back" @click="router.back()">←</button>
      <span class="topbar-title">我的收藏</span>
      <span class="topbar-spacer"></span>
    </header>

    <div class="grid">
      <VideoCard
        v-for="video in list"
        :key="video.id"
        :video="video"
        @open="router.push(`/video/${video.id}`)"
      />
    </div>

    <!-- 加载 / 空态 / 到底 / 重试 -->
    <div v-if="loading" class="state muted">加载中...</div>
    <p v-else-if="!list.length && !loading" class="state muted">还没有收藏视频</p>
    <p v-else-if="!hasMore && list.length" class="state muted">没有更多了</p>
    <p v-if="error" class="state error" @click="loadMore">加载失败，点击重试</p>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import VideoCard from '../components/VideoCard.vue'
import { favList } from '../api/favorite'
import { usePagination } from '../composables/usePagination'

const router = useRouter()

// usePagination 传 { cursor, limit }；favList 要 { offset, count }、返回 { videos, nextOffset }
// offset 语义 = 收藏记录 ID 游标，nextOffset 为 0 = 没有更多
const { list, loading, hasMore, error, loadMore } = usePagination(async (p) => {
  const data = await favList({ offset: p.cursor || 0, count: p.limit })
  return { videos: data.videos, nextCursor: data.nextOffset }
})

onMounted(loadMore)
</script>

<style scoped>
.favorites-page {
  padding: 0 var(--space-page) 32px;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
  margin: 0 calc(-1 * var(--space-page));
  padding: 0 var(--space-page);
  border-bottom: 1px solid var(--color-border);
}

.back {
  font-size: 20px;
  width: 32px;
  text-align: left;
  color: var(--color-text);
}

.topbar-title {
  font-size: var(--font-md);
  font-weight: 600;
}

.topbar-spacer {
  width: 32px;
}

.grid {
  margin-top: 16px;
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 14px;
}

.state {
  text-align: center;
  padding: 24px 0;
  font-size: var(--font-sm);
}

.state.error {
  color: var(--color-danger);
  cursor: pointer;
}
</style>
