// 登录态本地持久化：token / userId 的读写清除
// 约定：store 与 localStorage 双写双读，初始化时从这里恢复

const TOKEN_KEY = 'feed_token'
const USER_ID_KEY = 'feed_userId'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function getUserId() {
  const v = localStorage.getItem(USER_ID_KEY)
  return v ? Number(v) : null
}

export function setAuth(token, userId) {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USER_ID_KEY, String(userId))
}

export function clearAuth() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_ID_KEY)
}
