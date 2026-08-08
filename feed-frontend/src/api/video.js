// 视频模块接口封装（对应后端 docs/视频模块.md）
// 成功 resolve 的即为后端 data 字段

import request from './request'

// 视频列表：公开，游标分页；cursor 为 0/空 = 第一页；userId 空 = 全站
export function getVideoList({ cursor, limit = 20, userId } = {}) {
  return request.get('/video/list', {
    params: { cursor: cursor || undefined, limit, userId: userId || undefined },
  })
}

// 视频详情：公开，query 传 videoId
export function getVideoDetail(videoId) {
  return request.get('/video/detail', { params: { videoId } })
}

// 上传视频：需登录，multipart（file/title/describe?/cover?），支持上传进度回调
export function uploadVideo(formData, onProgress) {
  return request.post('/video/upload', formData, {
    onUploadProgress: (e) => {
      if (onProgress && e.total) {
        onProgress(Math.round((e.loaded / e.total) * 100))
      }
    },
  })
}

// 播放上报：无需登录，播放到 50% 时调用一次
export function reportPlay(videoId) {
  return request.post('/video/reportPlay', { videoId })
}

// 删除视频：需登录，仅作者本人
export function deleteVideo(id) {
  return request.delete(`/video/${id}`)
}
