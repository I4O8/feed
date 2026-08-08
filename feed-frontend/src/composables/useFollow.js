// 关注状态：拉自己的关注列表（全量无分页）判断「是否已关注某用户」
// 后端无「是否已关注」接口，采用 docs/前端查漏补缺.md 决策 3
// 模块级缓存：首次使用拉一次；follow/unfollow 成功后由调用方 setFollowing 本地更新，不重复请求

import { ref } from 'vue'
import { useUserStore } from '../stores/user'
import { followList } from '../api/feed'

const followingIds = ref(new Set()) // 已关注用户 ID 集合
let loaded = false                  // 是否已从后端拉取过
let loading = false
let error = null

// 拉取自己的关注列表，重建集合（登出时清空）
export async function refreshFollowing() {
  const userStore = useUserStore()
  if (!userStore.isLoggedIn) {
    followingIds.value = new Set()
    loaded = false
    error = null
    return
  }
  loading = true
  error = null
  try {
    const { users } = await followList(userStore.userId)
    followingIds.value = new Set(users.map((u) => Number(u.id)))
    loaded = true
  } catch (e) {
    error = e?.message || '获取关注列表失败'
  } finally {
    loading = false
  }
}

export function useFollow() {
  const userStore = useUserStore()

  // 未登录：清空并跳过加载
  if (!userStore.isLoggedIn) {
    followingIds.value = new Set()
    loaded = false
  } else if (!loaded && !loading) {
    refreshFollowing() // 懒加载一次
  }

  function isFollowing(id) {
    return followingIds.value.has(Number(id))
  }

  // 关注/取关成功后本地更新集合（不重新拉接口）
  function setFollowing(id, nowFollowing) {
    const nid = Number(id)
    if (nowFollowing) followingIds.value.add(nid)
    else followingIds.value.delete(nid)
  }

  return { followingIds, isFollowing, setFollowing, refreshFollowing, loading, error }
}
