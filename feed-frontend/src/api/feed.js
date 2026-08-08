// Feed 模块接口封装（对应后端 docs/feed模块.md）
// 全部需登录；成功 resolve 的即为后端 data 字段

import request from './request'

// 关注：path 传目标用户 id，返回空（重复关注幂等成功）
export function follow(id) {
  return request.post(`/follow/${id}`)
}

// 取关：path 传目标用户 id，返回空（未关注幂等成功）
export function unfollow(id) {
  return request.delete(`/follow/${id}`)
}

// 关注列表：query userId=查谁，返回 { users: [{id, name, avatar}] }
export function followList(userId) {
  return request.get('/follow/list', { params: { userId } })
}

// 粉丝列表：同上结构
export function fansList(userId) {
  return request.get('/follow/fans', { params: { userId } })
}

// 关注流：Redis List 下标分页，返回 { videos, nextOffset }
// offset = 上次返回的 nextOffset；nextOffset 为 0 = 没有更多
export function getFeed({ offset = 0, count = 20 } = {}) {
  return request.get('/feed', { params: { offset, count } })
}
