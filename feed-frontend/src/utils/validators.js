// 表单校验工具：前端预检，最终合法性由后端判定（错误码兜底）

// 后端规则：含 @ 按邮箱处理，否则按手机号处理
export function classifyAccount(account) {
  return account.includes('@') ? 'email' : 'phone'
}

export const isEmail = (v) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v)

// 11 位大陆手机号，1 开头
export const isPhone = (v) => /^1\d{10}$/.test(v)

// 密码 6-32 位
export const isPasswordValid = (v) =>
  typeof v === 'string' && v.length >= 6 && v.length <= 32

// 返回空字符串表示通过，否则返回错误提示
export function validateAccount(account) {
  if (!account) return '请输入手机号或邮箱'
  if (classifyAccount(account) === 'email') {
    return isEmail(account) ? '' : '邮箱格式不正确'
  }
  return isPhone(account) ? '' : '手机号格式不正确（11 位，1 开头）'
}
