package service

const unlockScript = `
  if redis.call("GET", KEYS[1]) == ARGV[1] then
      return redis.call("DEL", KEYS[1])
  else
      return 0
  end
  `

// vodAcquireScript 点播开播的并发判定：清死会话 + 判并发 + 加入，三步必须原子
// 拆成三条命令在高并发下会「超卖」（多个请求同时 ZCARD 都看到没超）
// KEYS[1] = vod:sess:<uid>  ARGV = now, 心跳容忍秒, 并发上限, sessionId
// 返回 1 = 准入；0 = 并发超限
const vodAcquireScript = `
  local key    = KEYS[1]
  local now    = tonumber(ARGV[1])
  local ttl    = tonumber(ARGV[2])
  local max    = tonumber(ARGV[3])
  local member = ARGV[4]
  redis.call("ZREMRANGEBYSCORE", key, "-inf", now - ttl)
  if redis.call("ZCARD", key) >= max then
      return 0
  end
  redis.call("ZADD", key, now, member)
  redis.call("EXPIRE", key, ttl * 2)
  return 1
  `

// vodHeartbeatScript 心跳续期：只更新已在集合里的成员，不新增（防止心跳凭空造会话）
// 返回 1 = 续期成功；0 = 会话不存在（已被清理或吊销）
const vodHeartbeatScript = `
  local key    = KEYS[1]
  local now    = tonumber(ARGV[1])
  local ttl    = tonumber(ARGV[2])
  local member = ARGV[3]
  if redis.call("ZSCORE", key, member) then
      redis.call("ZADD", key, now, member)
      redis.call("EXPIRE", key, ttl * 2)
      return 1
  end
  return 0
  `
