// 展示格式化：计数 / 时长

// 播放量：≥1万 → 1.2万，否则原样
export function formatCount(n) {
  const num = Number(n) || 0
  if (num >= 10000) return (num / 10000).toFixed(1).replace(/\.0$/, '') + '万'
  return String(num)
}

// 时长角标：83 → 1:23
export function formatDuration(seconds) {
  const s = Math.round(Number(seconds) || 0)
  const m = Math.floor(s / 60)
  const r = s % 60
  return `${m}:${String(r).padStart(2, '0')}`
}
