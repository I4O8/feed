<template>
  <div class="feed-page">
    <div ref="scrollEl" class="feed-scroll" @scroll="onScroll">
      <VideoFeedItem
        v-for="(video, idx) in list"
        :key="video.id"
        :video="video"
        :active="idx === currentIndex"
      />
      <!-- 底部哨兵：进入视口触发加载下一页 -->
      <div ref="sentinel" class="feed-tail">
        <span v-if="loading" class="tail-text">加载中...</span>
        <span v-else-if="!hasMore" class="tail-text">没有更多了</span>
        <span v-else-if="error" class="tail-text error" @click="loadMore">加载失败，点击重试</span>
      </div>
    </div>

    <!-- 空 feed：关注流为空时引导关注 -->
    <div v-if="!loading && !list.length && !error" class="empty">
      <p class="empty-title">关注你感兴趣的人</p>
      <p class="empty-desc">关注后，这里会展示 Ta 发布的视频</p>
      <RouterLink to="/" class="empty-btn">去首页逛逛</RouterLink>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import VideoFeedItem from '../components/VideoFeedItem.vue'
import { getFeed } from '../api/feed'
import { usePagination } from '../composables/usePagination'

// usePagination 传 { cursor, limit }、期望 { videos, nextCursor }；
// feed 接口要 { offset, count }、返回 { videos, nextOffset }，这里适配
const { list, loading, hasMore, error, loadMore } = usePagination(async (p) => {
  const data = await getFeed({ offset: p.cursor || 0, count: p.limit })
  return { videos: data.videos, nextCursor: data.nextOffset }
})

const scrollEl = ref(null)
const sentinel = ref(null)
const currentIndex = ref(0)
let observer = null

// 滚动计算当前屏 index（每屏高度 = 滚动容器可视高）
function onScroll() {
  const el = scrollEl.value
  if (!el || !el.clientHeight) return
  const idx = Math.round(el.scrollTop / el.clientHeight)
  const max = Math.max(list.value.length - 1, 0)
  currentIndex.value = Math.min(Math.max(idx, 0), max)
}

onMounted(() => {
  loadMore()
  scrollEl.value.scrollTop = 0
  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0]?.isIntersecting) loadMore()
    },
    { root: scrollEl.value }
  )
  if (sentinel.value) observer.observe(sentinel.value)
})

onUnmounted(() => {
  observer?.disconnect()
})
</script>

<style scoped>
.feed-page {
  height: 100%;
  position: relative;
}

/* 固定到可视区（下方让出 56px 给底部 Tab），宽屏居中限宽，吸附式滚动同首页 */
.feed-scroll {
  position: fixed;
  top: 0;
  bottom: 56px;
  left: 50%;
  transform: translateX(-50%);
  width: 100%;
  max-width: 480px;
  overflow-y: auto;
  scroll-snap-type: y proximity;
  overscroll-behavior: contain;
  background: #000;
  z-index: 1;
}

.feed-tail {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tail-text {
  color: rgba(255, 255, 255, 0.6);
  font-size: var(--font-xs);
}

.tail-text.error {
  color: #fff;
}

/* 空态：居中引导 */
.empty {
  position: fixed;
  top: 0;
  bottom: 56px;
  left: 50%;
  transform: translateX(-50%);
  width: 100%;
  max-width: 480px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 0 32px;
  text-align: center;
  z-index: 2;
}

.empty-title {
  font-size: var(--font-lg);
  font-weight: 700;
  color: var(--color-text);
}

.empty-desc {
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
}

.empty-btn {
  margin-top: 12px;
  padding: 10px 28px;
  border-radius: 999px;
  background: var(--color-primary);
  color: #fff;
  font-size: var(--font-sm);
  font-weight: 600;
}
</style>
