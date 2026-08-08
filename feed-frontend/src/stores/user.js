// 用户登录态 store：token / userId / profile
// token 与 localStorage 双写双读，刷新后状态不丢

import { defineStore } from 'pinia'
import { getToken, getUserId, setAuth as persistAuth, clearAuth } from '../utils/auth'
import { getAssetUrl } from '../api/asset'
import * as userApi from '../api/user'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: getToken(),        // 初始化时从 localStorage 恢复
    userId: getUserId(),
    profile: null,            // 当前查看用户的资料（个人主页数据）
  }),

  getters: {
    isLoggedIn: (s) => !!s.token,
    // 头像完整地址；未设置时为空串（组件用默认占位图兜底）
    avatarUrl: (s) => (s.profile?.avatar ? getAssetUrl(s.profile.avatar) : ''),
  },

  actions: {
    // 登录 / 注册成功后统一写入（store + localStorage）
    setAuth({ token, userId }) {
      this.token = token
      this.userId = userId
      persistAuth(token, userId)
    },

    async login(account, password) {
      const data = await userApi.login({ account, password })
      this.setAuth(data)
    },

    async register(account, password) {
      const data = await userApi.register({ account, password })
      this.setAuth(data)
    },

    // 登出 = 纯前端动作（后端无登出接口）：清状态，跳转由调用方页面负责
    logout() {
      this.token = ''
      this.userId = null
      this.profile = null
      clearAuth()
    },

    // 拉取个人主页资料（公开接口，可查任意 userId）
    async fetchProfile(userId) {
      this.profile = await userApi.getProfile(userId)
      return this.profile
    },

    // 修改资料：成功后同步本地 profile，保证个人主页响应式刷新
    async updateProfile(data) {
      await userApi.updateProfile(data)
      if (this.profile) this.profile = { ...this.profile, ...data }
    },

    // 上传头像：成功后同步本地 profile 的 avatar
    async uploadAvatar(file) {
      const { avatarUrl } = await userApi.uploadAvatar(file)
      if (this.profile) this.profile = { ...this.profile, avatar: avatarUrl }
      return avatarUrl
    },

    // 修改密码：成功后旧 token 仍有效，不做任何登出
    async changePassword(data) {
      await userApi.changePassword(data)
    },
  },
})
