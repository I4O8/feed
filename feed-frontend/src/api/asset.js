// 静态资源地址拼接：后端返回相对路径（如 /uploads/avatar/x.jpg）
// 所有 <img> 的 src 必须经过 getAssetUrl，禁止直接使用相对路径

import { BASE_URL } from './config'

export function getAssetUrl(path) {
  if (!path) return ''
  // 已是完整地址则原样返回（为后续接云存储预留）
  if (/^https?:\/\//.test(path)) return path
  return BASE_URL + path
}
