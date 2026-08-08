<template>
  <div class="page list-page">
    <header class="topbar">
      <button type="button" class="back" @click="router.back()">←</button>
      <span class="topbar-title">{{ isFans ? '粉丝' : '关注' }}</span>
      <span class="topbar-spacer"></span>
    </header>

    <div v-if="loading" class="loading muted">加载中...</div>

    <template v-else>
      <p v-if="!users.length" class="empty muted">
        {{ isFans ? '还没有粉丝' : '还没有关注任何人' }}
      </p>

      <ul v-else class="user-list">
        <!-- 点击整行跳作者主页；关注按钮单独 @click.stop 不触发跳转 -->
        <li
          v-for="u in users"
          :key="u.id"
          class="user-row"
          @click="router.push(`/profile/${u.id}`)"
        >
          <AppAvatar
            :src="u.avatar ? getAssetUrl(u.avatar) : ''"
            :name="u.name"
            :size="44"
          />
          <span class="user-name">{{ u.name }}</span>

          <!-- 非本人可关注/取关（不能关注自己） -->
          <button
            v-if="userStore.isLoggedIn && Number(u.id) !== Number(userStore.userId)"
            type="button"
            class="follow-btn"
            :class="{ followed: followState.isFollowing(u.id) }"
            @click.stop="toggleFollow(u)"
          >{{ followState.isFollowing(u.id) ? '已关注' : '+ 关注' }}</button>
        </li>
      </ul>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppAvatar from '../components/AppAvatar.vue'
import { useUserStore } from '../stores/user'
import { followList, fansList, follow, unfollow } from '../api/feed'
import { getAssetUrl } from '../api/asset'
import { useFollow } from '../composables/useFollow'
import { showToast } from '../utils/toast'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const followState = useFollow()

const userId = computed(() => Number(route.params.userId))
const isFans = computed(() => route.name === 'fans-list')

const users = ref([])
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  loading.value = true
  error.value = ''
  try {
    // 粉丝 / 关注列表都是全量返回（无分页）
    const data = isFans.value
      ? await fansList(userId.value)
      : await followList(userId.value)
    users.value = data.users || []
  } catch (err) {
    error.value = err?.message || '加载失败'
    showToast(error.value, 'error')
  } finally {
    loading.value = false
  }
})

// 关注/取关：乐观更新本地集合，失败回滚
async function toggleFollow(u) {
  const target = Number(u.id)
  const wasFollowing = followState.isFollowing(target)
  followState.setFollowing(target, !wasFollowing)
  try {
    if (wasFollowing) await unfollow(target)
    else await follow(target)
    showToast(wasFollowing ? '已取消关注' : '关注成功', 'success')
  } catch (err) {
    followState.setFollowing(target, wasFollowing)
    showToast(err?.message || '操作失败', 'error')
  }
}
</script>

<style scoped>
.list-page {
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
  text-align: center;
  padding: 48px 0;
}

.empty {
  text-align: center;
  padding: 48px 0;
}

.user-list {
  list-style: none;
  padding: 8px var(--space-page);
}

.user-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid var(--color-border);
  cursor: pointer;
}

.user-name {
  flex: 1;
  font-size: var(--font-sm);
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.follow-btn {
  flex-shrink: 0;
  padding: 6px 16px;
  border-radius: 999px;
  border: 1px solid var(--color-primary);
  color: var(--color-primary);
  font-size: var(--font-xs);
  font-weight: 600;
}

.follow-btn.followed {
  border-color: var(--color-border);
  color: var(--color-text-secondary);
  background: var(--color-input-bg);
}
</style>
