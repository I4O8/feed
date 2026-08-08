<template>
  <div id="app-root">
    <div class="page-wrap" :class="{ 'has-tab': showTab }">
      <RouterView />
    </div>

    <!-- 底部 Tab：仅主入口页显示（首页 / 关注 / 消息 / 我） -->
    <nav v-if="showTab" class="tab-bar">
      <RouterLink to="/" class="tab-item" :class="{ active: isActive('/') }">
        <span class="tab-icon">🏠</span>
        <span class="tab-label">首页</span>
      </RouterLink>
      <RouterLink to="/feed" class="tab-item" :class="{ active: isActive('/feed') }">
        <span class="tab-icon">❤️</span>
        <span class="tab-label">关注</span>
      </RouterLink>
      <RouterLink to="/message" class="tab-item" :class="{ active: isActive('/message') }">
        <span class="tab-icon">✉️</span>
        <span class="tab-label">消息</span>
      </RouterLink>
      <RouterLink :to="mePath" class="tab-item" :class="{ active: isActive('/profile') }">
        <span class="tab-icon">👤</span>
        <span class="tab-label">我</span>
      </RouterLink>
    </nav>

    <AppToast />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import AppToast from './components/AppToast.vue'
import { useUserStore } from './stores/user'

const route = useRoute()
const userStore = useUserStore()

// 仅带 showTab 标记的路由显示底部 Tab（登录/注册/编辑/改密是全屏页）
const showTab = computed(() => route.meta.showTab === true)
// 未登录点「我」→ 去登录页
const mePath = computed(() =>
  userStore.isLoggedIn ? `/profile/${userStore.userId}` : '/login'
)

function isActive(path) {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}
</script>

<style scoped>
#app-root {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.page-wrap {
  flex: 1;
  overflow-y: auto;
}

.page-wrap.has-tab {
  padding-bottom: 64px;
}

.tab-bar {
  position: fixed;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 100%;
  max-width: 480px;
  height: 56px;
  display: flex;
  background: #fff;
  border-top: 1px solid var(--color-border);
  z-index: 100;
}

.tab-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  color: var(--color-text-secondary);
  font-size: var(--font-xs);
  transition: color 0.2s;
}

.tab-item.active {
  color: var(--color-primary);
}

.tab-icon {
  font-size: 20px;
  line-height: 1;
}

.tab-label {
  line-height: 1;
}
</style>
