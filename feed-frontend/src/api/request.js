// Axios 统一封装：token 注入 / 响应解包 / 40100 兜底跳登录
// 成功时直接 resolve 后端 data；失败时 reject { code, message }，由页面 toast message

import axios from 'axios'
import { getToken, clearAuth } from '../utils/auth'
import { BASE_URL } from './config'
import router from '../router'

const request = axios.create({
  baseURL: BASE_URL,
  timeout: 10000,
})

// 请求拦截器：自动携带 Bearer token
request.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：统一解包 { code, message, data }
request.interceptors.response.use(
  (res) => {
    const body = res.data
    // 成功：直接返回 data
    if (body.code === 0) return body.data

    // 未登录 / token 过期：清 token，跳登录页并记录来源
    if (body.code === 40100) {
      clearAuth()
      router.push({
        path: '/login',
        query: { redirect: router.currentRoute.value.fullPath },
      })
      return Promise.reject({ code: body.code, message: body.message })
    }

    // 其余错误码：交给调用方提示 message
    return Promise.reject({ code: body.code, message: body.message })
  },
  // HTTP 层失败（断网 / 超时 / 后端未启动）
  () => Promise.reject({ code: -1, message: '网络错误，请稍后重试' })
)

export default request
