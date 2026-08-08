<template>
  <div class="page profile-page">
    <header class="topbar">
      <span class="topbar-title">个人主页</span>
    </header>

    <!-- 加载态 -->
    <div v-if="loading" class="loading">
      <span class="spinner" aria-hidden="true"></span>
      <span>加载中...</span>
    </div>

    <!-- 资料卡片 -->
    <section v-else-if="profile" class="card">
      <div class="avatar-wrap">
        <AppAvatar
          :src="profile.avatar ? getAssetUrl(profile.avatar) : ''"
          :name="profile.name"
          :size="88"
        />
      </div>
      <h2 class="name">{{ profile.name }}</h2>
      <p v-if="profile.bio" class="bio">{{ profile.bio }}</p>
      <p v-else class="muted bio-empty">这个人很懒，还没有简介</p>
      <p class="muted joined">加入于 {{ formatDate(profile.createdAt) }}</p>

      <!-- 关注 / 粉丝统计（点击进列表页） -->
      <div class="stats">
        <RouterLink :to="`/profile/${userId}/following`" class="stat">
          <span class="stat-num">{{ followCount }}</span>
          <span class="stat-label">关注</span>
        </RouterLink>
        <RouterLink :to="`/profile/${userId}/fans`" class="stat">
          <span class="stat-num">{{ fansCount }}</span>
          <span class="stat-label">粉丝</span>
        </RouterLink>
      </div>

      <!-- 非本人的关注 / 取关按钮 -->
      <div v-if="!isMe" class="follow-area">
        <AppButton
          v-if="userStore.isLoggedIn"
          :primary="!followState.isFollowing(userId)"
          @click="toggleFollow"
        >{{ followState.isFollowing(userId) ? '已关注' : '+ 关注' }}</AppButton>
        <AppButton v-else primary @click="goLogin">+ 关注</AppButton>
      </div>

      <!-- 作品入口：所有人可见（公开作品） -->
      <RouterLink class="works-link" :to="`/profile/${userId}/videos`">
        <span>{{ isMe ? '我的作品' : 'Ta 的作品' }}</span>
        <span class="works-arrow">›</span>
      </RouterLink>

      <!-- 本人的收藏入口 -->
      <RouterLink v-if="isMe" class="works-link" to="/favorites">
        <span>我的收藏</span>
        <span class="works-arrow">›</span>
      </RouterLink>

      <!-- 自己的主页：编辑资料 / 修改密码入口 -->
      <div v-if="isMe" class="actions">
        <AppButton primary class="action" @click="router.push('/profile/edit')">
          编辑资料
        </AppButton>
        <AppButton primary class="action ghost" @click="router.push('/profile/security')">
          修改密码
        </AppButton>
        <button type="button" class="logout" @click="showLogout = true">退出登录</button>
      </div>
    </section>

    <p v-else class="muted not-found">用户不存在</p>

    <AppDialog
      v-model="showLogout"
      title="退出登录"
      message="确定要退出当前账号吗？"
      @confirm="handleLogout"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import AppAvatar from '../components/AppAvatar.vue'
import AppButton from '../components/AppButton.vue'
import AppDialog from '../components/AppDialog.vue'
import { useUserStore } from '../stores/user'
import { getAssetUrl } from '../api/asset'
import { follow, unfollow, followList, fansList } from '../api/feed'
import { useFollow } from '../composables/useFollow'
import { showToast } from '../utils/toast'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const followState = useFollow()

const loading = ref(true)
const showLogout = ref(false)
const followCount = ref(0)
const fansCount = ref(0)

const userId = computed(() => Number(route.params.userId))
const profile = computed(() => userStore.profile)
const isMe = computed(
  () => userStore.isLoggedIn && userId.value === Number(userStore.userId)
)

function formatDate(iso) {
  if (!iso) return ''
  return iso.slice(0, 10) // '2026-08-02'
}

// 未登录点关注 → 跳登录带 redirect（登录后回跳本页）
function goLogin() {
  router.push({ path: '/login', query: { redirect: route.fullPath } })
}

// 关注 / 取关：乐观更新本地集合，失败回滚
async function toggleFollow() {
  const target = userId.value
  const wasFollowing = followState.isFollowing(target)
  followState.setFollowing(target, !wasFollowing)
  // 乐观调整统计
  followCount.value = Math.max(0, followCount.value + (wasFollowing ? -1 : 1))
  try {
    if (wasFollowing) await unfollow(target)
    else await follow(target)
    showToast(wasFollowing ? '已取消关注' : '关注成功', 'success')
  } catch (err) {
    followState.setFollowing(target, wasFollowing)
    followCount.value = Math.max(0, followCount.value + (wasFollowing ? 1 : -1))
    showToast(err?.message || '操作失败', 'error')
  }
}

function handleLogout() {
  userStore.logout()
  showToast('已退出登录')
  router.push('/login')
}

onMounted(async () => {
  loading.value = true
  try {
    // 并行拉：资料 + 关注数 + 粉丝数
    const [, fList, fans] = await Promise.all([
      userStore.fetchProfile(userId.value),
      followList(userId.value),
      fansList(userId.value),
    ])
    followCount.value = (fList.users || []).length
    fansCount.value = (fans.users || []).length
  } catch (err) {
    showToast(err?.message || '加载失败', 'error')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.profile-page {
  padding: 0;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 48px;
  border-bottom: 1px solid var(--color-border);
}

.topbar-title {
  font-size: var(--font-md);
  font-weight: 600;
}

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 48px 0;
  color: var(--color-text-secondary);
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px var(--space-page);
}

.name {
  margin-top: 16px;
  font-size: var(--font-lg);
  font-weight: 700;
}

.bio {
  margin-top: 8px;
  font-size: var(--font-sm);
  color: var(--color-text);
  text-align: center;
}

.bio-empty {
  margin-top: 8px;
}

.joined {
  margin-top: 8px;
}

/* 关注 / 粉丝统计 */
.stats {
  display: flex;
  gap: 40px;
  margin-top: 20px;
}

.stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.stat-num {
  font-size: var(--font-lg);
  font-weight: 700;
}

.stat-label {
  font-size: var(--font-xs);
  color: var(--color-text-secondary);
}

.follow-area {
  width: 100%;
  margin-top: 20px;
}

.works-link {
  margin-top: 16px;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: var(--color-input-bg);
  border-radius: var(--radius-btn);
  font-size: var(--font-sm);
  font-weight: 600;
}

.works-arrow {
  color: var(--color-text-secondary);
  font-size: var(--font-lg);
}

.actions {
  display: flex;
  gap: 12px;
  width: 100%;
  margin-top: 32px;
}

.action {
  flex: 1;
}

.action.ghost {
  background: transparent;
  color: var(--color-primary);
  border: 1px solid var(--color-primary);
}

.logout {
  margin-top: 8px;
  font-size: var(--font-sm);
  color: var(--color-danger);
}

.not-found {
  text-align: center;
  padding: 48px 0;
}
</style>
