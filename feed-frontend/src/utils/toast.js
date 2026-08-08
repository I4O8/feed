// 全局轻提示：reactive 状态 + AppToast.vue 渲染
// 用法：showToast('保存成功', 'success') / showToast(message) 默认失败样式之外的 info

import { reactive } from 'vue'

export const toastState = reactive({
  visible: false,
  message: '',
  type: 'info', // info | success | error
})

let timer = null

export function showToast(message, type = 'info', duration = 2000) {
  toastState.message = message
  toastState.type = type
  toastState.visible = true
  clearTimeout(timer)
  timer = setTimeout(() => {
    toastState.visible = false
  }, duration)
}
