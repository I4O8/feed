// 点赞与收藏模块接口封装（对应后端 docs/点赞与收藏模块.md）
// 全部需登录；成功 resolve 的即为后端 data 字段

import request from './request'

// 点赞：path 传视频 id，返回空（已赞幂等成功）
export function like(id) {
  return request.post(`/like/${id}`)
}

// 取消点赞：path 传视频 id，返回空（未赞幂等成功）
export function unlike(id) {
  return request.delete(`/like/${id}`)
}

// 是否已点赞：query videoId，返回 { liked: boolean }
export function likeStatus(videoId) {
  return request.get('/like/status', { params: { videoId } })
}

// 收藏：path 传视频 id，返回空（已藏幂等成功）
export function fav(id) {
  return request.post(`/fav/${id}`)
}

// 取消收藏：返回空（未藏幂等成功）
export function unfav(id) {
  return request.delete(`/fav/${id}`)
}

// 是否已收藏：query videoId，返回 { favored: boolean }
export function favStatus(videoId) {
  return request.get('/fav/status', { params: { videoId } })
}

// 我的收藏列表：返回 { videos, nextOffset }
// offset = 上次返回的 nextOffset（收藏记录 ID 游标）；nextOffset 为 0 = 没有更多
export function favList({ offset = 0, count = 20 } = {}) {
  return request.get('/fav/list', { params: { offset, count } })
}
