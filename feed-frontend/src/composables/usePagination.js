// 游标分页组合函数：首页视频流 / 我的作品页复用
// 核心：记录 nextCursor，为 0 停止；防重复请求；失败记录 error 不中断

import { ref } from 'vue'

// fetcher: ({ cursor, limit }) => Promise<{ videos, nextCursor }>
export function usePagination(fetcher, { limit = 20 } = {}) {
  const list = ref([])
  const cursor = ref(0)
  const loading = ref(false)
  const hasMore = ref(true)
  const error = ref('')

  const loadMore = async () => {
    if (loading.value || !hasMore.value) return
    loading.value = true
    error.value = ''
    try {
      const data = await fetcher({ cursor: cursor.value || undefined, limit })
      list.value.push(...data.videos)
      cursor.value = data.nextCursor
      if (!data.nextCursor) hasMore.value = false
    } catch (e) {
      error.value = e?.message || '加载失败'
    } finally {
      loading.value = false
    }
  }

  const reset = () => {
    list.value = []
    cursor.value = 0
    hasMore.value = true
    error.value = ''
  }

  return { list, loading, hasMore, error, loadMore, reset }
}
