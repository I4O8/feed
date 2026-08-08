// 路由表 + 全局前置守卫（登录鉴权）
// requiresAuth：无 token 访问 → 跳登录页并记录来源
// guestOnly：已登录访问登录/注册页 → 跳首页
// 注意：页面文件为懒加载，阶段 2 逐个填充实现

import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../stores/user'

const routes = [
  {
    path: '/',
    name: 'home',
    component: () => import('../views/HomeView.vue'),
    meta: { title: '首页', showTab: true },
  },
  {
    path: '/message',
    name: 'message',
    component: () => import('../views/MessageView.vue'),
    meta: { title: '消息', showTab: true },
  },
  {
    path: '/feed',
    name: 'feed',
    component: () => import('../views/FollowFeedView.vue'),
    meta: { title: '关注', showTab: true, requiresAuth: true },
  },
  {
    path: '/favorites',
    name: 'favorites',
    component: () => import('../views/FavoritesView.vue'),
    meta: { title: '我的收藏', requiresAuth: true },
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/LoginView.vue'),
    meta: { title: '登录', guestOnly: true },
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('../views/RegisterView.vue'),
    meta: { title: '注册', guestOnly: true },
  },
  {
    path: '/video/:videoId',
    name: 'video-detail',
    component: () => import('../views/VideoDetailView.vue'),
    meta: { title: '视频详情' },
  },
  {
    path: '/publish',
    name: 'publish',
    component: () => import('../views/PublishView.vue'),
    meta: { title: '发布视频', requiresAuth: true },
  },
  {
    path: '/profile/:userId/videos',
    name: 'my-videos',
    component: () => import('../views/MyVideosView.vue'),
    meta: { title: '作品' },
  },
  {
    path: '/profile/:userId',
    name: 'profile',
    component: () => import('../views/ProfileView.vue'),
    meta: { title: '个人主页', showTab: true },
  },
  {
    path: '/profile/:userId/following',
    name: 'follow-list',
    component: () => import('../views/FollowListPage.vue'),
    meta: { title: '关注列表', requiresAuth: true },
  },
  {
    path: '/profile/:userId/fans',
    name: 'fans-list',
    component: () => import('../views/FollowListPage.vue'),
    meta: { title: '粉丝列表', requiresAuth: true },
  },
  {
    path: '/profile/edit',
    name: 'edit-profile',
    component: () => import('../views/EditProfileView.vue'),
    meta: { title: '编辑资料', requiresAuth: true },
  },
  {
    path: '/profile/security',
    name: 'change-password',
    component: () => import('../views/ChangePasswordView.vue'),
    meta: { title: '修改密码', requiresAuth: true },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 全局前置守卫：useUserStore 在回调内调用（延迟到导航时执行，保证 pinia 已激活）
router.beforeEach((to) => {
  const userStore = useUserStore()

  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  if (to.meta.guestOnly && userStore.isLoggedIn) {
    return { path: '/' }
  }
})

export default router
