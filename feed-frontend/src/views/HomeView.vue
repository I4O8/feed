<template>
  <div class="home">
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

    <!-- 右上发布悬浮按钮（未登录会被路由守卫带到登录页） -->
    <RouterLink to="/publish" class="fab" title="发布视频">＋</RouterLink>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import VideoFeedItem from '../components/VideoFeedItem.vue'
import { getVideoList } from '../api/video'
import { usePagination } from '../composables/usePagination'

const { list, loading, hasMore, error, loadMore } = usePagination((p) => getVideoList(p))

const scrollEl = ref(null)
const sentinel = ref(null)
const currentIndex = ref(0)
let observer = null

// 滚动计算当前屏 index（每屏高度 = 滚动容器可视高）
function onScroll() {
  const el = scrollEl.value
  if (!el || !el.clientHeight) return
  const idx = Math.round(el.scrollTop / el.clientHeight)
  // 越界保护：空列表时保持 0
  const max = Math.max(list.value.length - 1, 0)
  currentIndex.value = Math.min(Math.max(idx, 0), max)
}

onMounted(() => {
  loadMore()
  // 初始固定在顶部（避免 scroll-snap 把位置吸到末尾）
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
.home {
  height: 100%;
  position: relative;
}

/* 固定到可视区（下方让出 56px 给底部 Tab），宽屏居中限宽。
   scroll-snap 用 proximity：温和吸附，不做强制跳转（mandatory 会在异步加载后吸到末尾） */
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

.fab {
  position: fixed;
  right: max(16px, calc((100vw - 480px) / 2 + 16px));
  bottom: 72px;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: var(--color-primary);
  color: #fff;
  font-size: 26px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 90;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.35);
}
</style>
