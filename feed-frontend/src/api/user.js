// 账号模块接口封装（对应后端 docs/账号模块.md）
// 成功 resolve 的即为后端 data 字段

import request from './request'

// 注册：公开，返回 { token, userId }（注册即登录）
export function register(data) {
  return request.post('/user/register', data)
}

// 登录：公开，返回 { token, userId }
export function login(data) {
  return request.post('/user/login', data)
}

// 获取用户资料：公开，query 传 userId，返回 UserProfile
export function getProfile(userId) {
  return request.get('/user/getProfile', { params: { userId } })
}

// 修改资料：需登录，body { name?, bio? }
export function updateProfile(data) {
  return request.post('/user/updateProfile', data)
}

// 修改密码：需登录，body { oldPassword, newPassword }
export function changePassword(data) {
  return request.post('/user/changePassword', data)
}

// 上传头像：需登录，multipart/form-data，字段名 file，返回 { avatarUrl }
export function uploadAvatar(file) {
  const formData = new FormData()
  formData.append('file', file)
  return request.post('/user/uploadAvatar', formData)
}
