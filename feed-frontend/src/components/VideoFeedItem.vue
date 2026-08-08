<template>
  <div class="feed-item">
    <!-- 视频铺满：absolute 定位占满 item，不撑开布局也不被 info 挤压 -->
    <div class="video-wrap">
      <VideoPlayer
        ref="player"
        :src="getAssetUrl(video.videoUrl)"
        :poster="video.coverUrl ? getAssetUrl(video.coverUrl) : ''"
        :loop="true"
        fit="contain"
        @timeupdate="onTimeupdate"
        @click="onTap"
      />
    </div>

    <!-- 左下信息浮层 -->
    <div class="info">
      <div class="author" @click="goProfile">
        <AppAvatar
          :src="video.publisher?.avatar ? getAssetUrl(video.publisher.avatar) : ''"
          :name="video.publisher?.name"
          :size="36"
        />
        <span class="author-name">{{ video.publisher?.name || '未知用户' }}</span>
      </div>
      <p class="title">{{ video.title }}</p>
      <p v-if="video.describe" class="desc">{{ video.describe }}</p>
    </div>

    <!-- 右侧操作区：点赞 / 收藏可交互（乐观更新），播放量展示 -->
    <div class="side-actions">
      <RouterLink class="detail-btn" :to="`/video/${video.id}`">详情</RouterLink>
      <div class="action" :class="{ active: liked }" @click="toggleLike">
        <span class="icon">♥</span>
        <span class="count">{{ formatCount(video.likeCount) }}</span>
      </div>
      <div class="action" :class="{ active: favored }" @click="toggleFav">
        <span class="icon">★</span>
        <span class="count">{{ formatCount(video.favCount) }}</span>
      </div>
      <div class="action">
        <span class="icon">▶</span>
        <span class="count">{{ formatCount(video.playCount) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import VideoPlayer from './VideoPlayer.vue'
import AppAvatar from './AppAvatar.vue'
import { useUserStore } from '../stores/user'
import { getAssetUrl } from '../api/asset'
import { reportPlay } from '../api/video'
import { like, unlike, likeStatus, fav, unfav, favStatus } from '../api/favorite'
import { formatCount } from '../utils/format'
import { showToast } from '../utils/toast'

const props = defineProps({
  video: { type: Object, required: true },
  active: { type: Boolean, default: false },
})

const router = useRouter()
const userStore = useUserStore()
const player = ref(null)
const reported = ref(false)

// 点赞 / 收藏状态（登录后初始化 + 乐观更新）
const liked = ref(false)
const favored = ref(false)
const busy = ref(false)

// 当前屏自动播放（静音），离屏暂停
watch(
  () => props.active,
  (active) => {
    if (active) player.value?.play()
    else player.value?.pause()
  },
  { immediate: true }
)

// 已登录时初始化点赞 / 收藏状态（失败静默，保持未点亮）
onMounted(async () => {
  if (!userStore.isLoggedIn) return
  try {
    const [l, f] = await Promise.all([
      likeStatus(props.video.id),
      favStatus(props.video.id),
    ])
    liked.value = l.liked
    favored.value = f.favored
  } catch (e) {
    // 状态获取失败静默
  }
})

// 播放到 50% 上报一次播放量（标记在本视频项内，滑开再滑回不重复）
function onTimeupdate({ progress, duration }) {
  if (!props.active || reported.value || duration === 0) return
  if (progress >= 0.5) {
    reported.value = true
    reportPlay(props.video.id).catch(() => {})
    props.video.playCount = (props.video.playCount || 0) + 1
  }
}

// 轻触：静音则恢复声音播放，否则播放/暂停切换
function onTap() {
  const v = player.value
  if (!v) return
  if (v.isMuted()) v.unmuteAndPlay()
  else v.toggle()
}

// 未登录点击互动 → 跳登录页
function goLogin() {
  router.push({ path: '/login', query: { redirect: '/' } })
}

// 点赞 / 取消：乐观更新计数，失败回滚
async function toggleLike() {
  if (!userStore.isLoggedIn) return goLogin()
  if (busy.value) return
  busy.value = true
  const prev = liked.value
  liked.value = !prev
  props.video.likeCount = Math.max(0, (props.video.likeCount || 0) + (liked.value ? 1 : -1))
  try {
    if (liked.value) await like(props.video.id)
    else await unlike(props.video.id)
  } catch (err) {
    liked.value = prev
    props.video.likeCount = Math.max(0, (props.video.likeCount || 0) + (prev ? 1 : -1))
    showToast(err?.message || '操作失败', 'error')
  } finally {
    busy.value = false
  }
}

// 收藏 / 取消：同上
async function toggleFav() {
  if (!userStore.isLoggedIn) return goLogin()
  if (busy.value) return
  busy.value = true
  const prev = favored.value
  favored.value = !prev
  props.video.favCount = Math.max(0, (props.video.favCount || 0) + (favored.value ? 1 : -1))
  try {
    if (favored.value) await fav(props.video.id)
    else await unfav(props.video.id)
  } catch (err) {
    favored.value = prev
    props.video.favCount = Math.max(0, (props.video.favCount || 0) + (prev ? 1 : -1))
    showToast(err?.message || '操作失败', 'error')
  } finally {
    busy.value = false
  }
}

// 跳发布者主页：接口未返回 publisher.id 时（契约限制），有 id 才跳
function goProfile() {
  const pid = props.video.publisher?.id ?? props.video.uploader
  if (pid) router.push(`/profile/${pid}`)
}
</script>

<style scoped>
.feed-item {
  position: relative;
  height: 100%;
  width: 100%;
  background: #000;
  scroll-snap-align: start;
  overflow: hidden;
}

.video-wrap {
  position: absolute;
  inset: 0;
  z-index: 1;
}

/* 信息浮层：底部渐变，保证盖在视频之上 */
.info {
  position: absolute;
  left: 12px;
  right: 76px;
  bottom: 20px;
  color: #fff;
  text-shadow: 0 1px 4px rgba(0, 0, 0, 0.5);
  z-index: 20;
}

.author {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  cursor: pointer;
}

.author-name {
  font-size: var(--font-sm);
  font-weight: 600;
}

.title {
  font-size: var(--font-md);
  font-weight: 600;
  margin-bottom: 4px;
}

.desc {
  font-size: var(--font-xs);
  opacity: 0.85;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 右侧操作区 */
.side-actions {
  position: absolute;
  right: 10px;
  bottom: 80px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  color: #fff;
  z-index: 20;
}

.detail-btn {
  font-size: var(--font-xs);
  padding: 4px 10px;
  border: 1px solid rgba(255, 255, 255, 0.6);
  border-radius: 999px;
  color: #fff;
}

.action {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  font-size: 20px;
  cursor: pointer;
}

.action .count {
  font-size: var(--font-xs);
}

/* 点赞 / 收藏点亮态 */
.action.active {
  color: var(--color-primary);
}
</style>
