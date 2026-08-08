<template>
  <div class="page my-videos-page">
    <header class="topbar">
      <button type="button" class="back" @click="router.back()">←</button>
      <span class="topbar-title">{{ isMe ? '我的作品' : 'Ta 的作品' }}</span>
      <span class="topbar-spacer"></span>
    </header>

    <div class="grid">
      <VideoCard
        v-for="video in list"
        :key="video.id"
        :video="video"
        :show-delete="isMe"
        @open="router.push(`/video/${video.id}`)"
        @delete="askDelete(video)"
      />
    </div>

    <!-- 加载 / 空态 / 到底 -->
    <div v-if="loading" class="state muted">加载中...</div>
    <p v-else-if="!list.length && !loading" class="state muted">还没有发布视频</p>
    <p v-else-if="!hasMore && list.length" class="state muted">没有更多了</p>
    <p v-if="error" class="state error" @click="loadMore">加载失败，点击重试</p>

    <AppDialog
      v-model="showDialog"
      title="删除视频"
      message="确定要删除该视频吗？删除后不可恢复。"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import VideoCard from '../components/VideoCard.vue'
import AppDialog from '../components/AppDialog.vue'
import { useUserStore } from '../stores/user'
import { getVideoList, deleteVideo } from '../api/video'
import { usePagination } from '../composables/usePagination'
import { showToast } from '../utils/toast'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const userId = computed(() => Number(route.params.userId))
const isMe = computed(
  () => userStore.isLoggedIn && userId.value === Number(userStore.userId)
)

const { list, loading, hasMore, error, loadMore, reset } = usePagination((p) =>
  getVideoList({ ...p, userId: userId.value })
)

const showDialog = ref(false)
const pendingVideo = ref(null)

onMounted(loadMore)

// 切换作者时重置并重新加载
watch(userId, () => {
  reset()
  loadMore()
})

function askDelete(video) {
  pendingVideo.value = video
  showDialog.value = true
}

async function confirmDelete() {
  const video = pendingVideo.value
  if (!video) return
  try {
    await deleteVideo(video.id)
    list.value = list.value.filter((v) => v.id !== video.id)
    showToast('已删除', 'success')
  } catch (err) {
    showToast(err?.message || '删除失败', 'error')
  } finally {
    pendingVideo.value = null
  }
}
</script>

<style scoped>
.my-videos-page {
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
