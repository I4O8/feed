<template>
  <div class="page detail-page">
    <header class="topbar">
      <button type="button" class="back" @click="router.back()">←</button>
      <span class="topbar-title">视频详情</span>
      <span class="topbar-spacer"></span>
    </header>

    <div v-if="loading" class="loading muted">加载中...</div>

    <template v-else-if="video">
      <!-- 播放器：带系统控制条，用户可暂停/拖进度/全屏 -->
      <div class="player-wrap">
        <VideoPlayer
          ref="player"
          :src="getAssetUrl(video.videoUrl)"
          :poster="video.coverUrl ? getAssetUrl(video.coverUrl) : ''"
          autoplay
          :controls="true"
          fit="contain"
          @timeupdate="onTimeupdate"
        />
      </div>

      <div class="body">
        <h1 class="title">{{ video.title }}</h1>
        <p v-if="video.describe" class="desc">{{ video.describe }}</p>

        <div class="publisher" @click="goProfile">
          <AppAvatar
            :src="video.publisher?.avatar ? getAssetUrl(video.publisher.avatar) : ''"
            :name="video.publisher?.name"
            :size="44"
          />
          <div class="pub-info">
            <span class="pub-name">{{ video.publisher?.name || '未知用户' }}</span>
            <span class="pub-time">{{ formatDate(video.createdAt) }}</span>
          </div>
        </div>

        <!-- 计数行：点赞 / 收藏可交互（乐观更新），播放量展示 -->
        <div class="counts">
          <div class="count-item" @click="toggleLike">
            <span class="icon" :class="{ active: liked }">♥</span>
            <span>{{ formatCount(video.likeCount) }}</span>
          </div>
          <div class="count-item" @click="toggleFav">
            <span class="icon" :class="{ active: favored }">★</span>
            <span>{{ formatCount(video.favCount) }}</span>
          </div>
          <div class="count-item">
            <span class="icon">▶</span>
            <span>{{ formatCount(video.playCount) }}</span>
          </div>
        </div>
      </div>
    </template>

    <p v-else class="muted not-found">视频不存在</p>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import VideoPlayer from '../components/VideoPlayer.vue'
import AppAvatar from '../components/AppAvatar.vue'
import { useUserStore } from '../stores/user'
import { getVideoDetail, reportPlay } from '../api/video'
import { like, unlike, likeStatus, fav, unfav, favStatus } from '../api/favorite'
import { getAssetUrl } from '../api/asset'
import { formatCount } from '../utils/format'
import { showToast } from '../utils/toast'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const videoId = computed(() => Number(route.params.videoId))
const loading = ref(true)
const video = ref(null)
const reported = ref(false)

// 点赞 / 收藏状态
const liked = ref(false)
const favored = ref(false)
const busy = ref(false)

function formatDate(iso) {
  if (!iso) return ''
  return iso.slice(0, 10)
}

// 跳发布者主页：契约未返回 publisher.id 时，有 id 才跳
function goProfile() {
  const pid = video.value?.publisher?.id ?? video.value?.uploader
  if (pid) router.push(`/profile/${pid}`)
}

// 播放到 50% 上报一次
function onTimeupdate({ progress, duration }) {
  if (!video.value || reported.value || duration === 0) return
  if (progress >= 0.5) {
    reported.value = true
    reportPlay(video.value.id).catch(() => {})
    video.value.playCount = (video.value.playCount || 0) + 1
  }
}

// 未登录点击互动 → 跳登录带 redirect（登录后回跳本页）
function goLogin() {
  router.push({ path: '/login', query: { redirect: route.fullPath } })
}

// 点赞 / 取消：乐观更新计数，失败回滚
async function toggleLike() {
  if (!userStore.isLoggedIn) return goLogin()
  if (!video.value || busy.value) return
  busy.value = true
  const prev = liked.value
  liked.value = !prev
  video.value.likeCount = Math.max(0, (video.value.likeCount || 0) + (liked.value ? 1 : -1))
  try {
    if (liked.value) await like(video.value.id)
    else await unlike(video.value.id)
  } catch (err) {
    liked.value = prev
    video.value.likeCount = Math.max(0, (video.value.likeCount || 0) + (prev ? 1 : -1))
    showToast(err?.message || '操作失败', 'error')
  } finally {
    busy.value = false
  }
}

// 收藏 / 取消：同上
async function toggleFav() {
  if (!userStore.isLoggedIn) return goLogin()
  if (!video.value || busy.value) return
  busy.value = true
  const prev = favored.value
  favored.value = !prev
  video.value.favCount = Math.max(0, (video.value.favCount || 0) + (favored.value ? 1 : -1))
  try {
    if (favored.value) await fav(video.value.id)
    else await unfav(video.value.id)
  } catch (err) {
    favored.value = prev
    video.value.favCount = Math.max(0, (video.value.favCount || 0) + (prev ? 1 : -1))
    showToast(err?.message || '操作失败', 'error')
  } finally {
    busy.value = false
  }
}

// 已登录时初始化点赞 / 收藏状态（失败静默）
async function loadStatus() {
  if (!userStore.isLoggedIn) return
  try {
    const [l, f] = await Promise.all([
      likeStatus(videoId.value),
      favStatus(videoId.value),
    ])
    liked.value = l.liked
    favored.value = f.favored
  } catch (e) {
    // 状态获取失败静默
  }
}

onMounted(async () => {
  loadStatus() // 与详情加载并行
  loading.value = true
  try {
    video.value = await getVideoDetail(videoId.value)
  } catch (err) {
    showToast(err?.message || '加载失败', 'error')
    // 40400（视频不存在）返回上一页；无历史则回首页
    if (window.history.length > 1) router.back()
    else router.push('/')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.detail-page {
  padding: 0;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
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

.loading {
  padding: 48px 0;
  text-align: center;
}

.player-wrap {
  height: 60vh;
  width: 100%;
  background: #000;
}

.body {
  padding: 16px var(--space-page);
}

.title {
  font-size: var(--font-lg);
  font-weight: 700;
  line-height: 1.4;
}

.desc {
  margin-top: 8px;
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
  line-height: 1.6;
}

.publisher {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 16px;
  cursor: pointer;
}

.pub-info {
  display: flex;
  flex-direction: column;
}

.pub-name {
  font-size: var(--font-sm);
  font-weight: 600;
}

.pub-time {
  font-size: var(--font-xs);
  color: var(--color-text-secondary);
}

.counts {
  display: flex;
  gap: 24px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border);
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
}

.count-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
}

.count-item .icon {
  font-size: 18px;
}

/* 点赞 / 收藏点亮态 */
.count-item .icon.active {
  color: var(--color-primary);
}

.not-found {
  text-align: center;
  padding: 48px 0;
}
</style>
