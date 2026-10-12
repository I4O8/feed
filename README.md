# Feed 流媒体点播与关注视频流项目

## 简介

一个以 **流媒体点播（VOD）** 为核心的后端项目，同时包含一套「仿短视频」的关注视频流业务；采用 Go + Gin + GORM + MySQL 技术栈，深度实践了 Redis 缓存与 RabbitMQ 消息队列。

**点播模块是一条独立的流媒体安全主线**：把 **DRM / 防盗链 / 反盗号 / 泄密溯源** 落实成可运行的代码 —— 授权闸门、CENC 加密、A/B 取证水印、签名 URL。它与短视频模块并行存在（各占一套表、一套链路），共用账号 / JWT / Redis / RabbitMQ。

短视频部分按 **MVC 分层**（handler → service → dao → model）开发，包含完整的 **优化版 / 无优化版后端对比** 与 **性能压测工具**，可以用数据直观量化每一项优化的价值（Redis 缓存、N+1 批量查询、分布式锁、MQ 异步扩散等）。

## 功能模块

| 模块 | 功能 |
|------|------|
| **点播（VOD）** | 电影上架（NFO 导入）/ 片库浏览 / **播放凭证闸门** / **DRM（CENC+ClearKey、HLS AES-128）** / **A/B 取证水印** / 签名 URL / 电影收藏 / 断点续播 / 观看历史 |
| **账号** | 注册 / 登录 / 个人资料 / 修改密码 / 上传头像 |
| **视频** | 上传 / 列表 / 详情 / 删除 / 播放上报 |
| **关注** | 关注 / 取关 / 关注列表 / 粉丝列表 |
| **Feed** | 关注流 |
| **点赞** | 点赞 / 取消 / 是否已赞 |
| **收藏** | 收藏 / 取消 / 是否已收藏 / 我的收藏 |
| **前端** | 登录注册 / 点播片库与播放 / 视频流 / 个人主页 / 点赞收藏 |

## 接口名单

### 点播 · 管理端

> 鉴权用请求头 **`X-Admin-Key`**（与 `VOD_ADMIN_KEY` 环境变量常量时间比较），不是 JWT。

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/admin/movie/import` | 按 **NFO** 文件导入电影元数据（multipart，字段 `nfo`）| Admin |
| POST | `/admin/movie` | 手动新建电影 | Admin |
| PUT | `/admin/movie/:id` | 修改元数据 | Admin |
| DELETE | `/admin/movie/:id` | 下架 / 删除 | Admin |
| POST | `/admin/movie/:id/source` | 上传**母版 A + B**（字段 `sourceA`/`sourceB`）→ 触发异步打包 | Admin |
| GET | `/admin/movie/:id/status` | 查打包状态 | Admin |

### 点播 · 浏览

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | `/movie/list` | 片库列表（`sort=latest/hot/rating` + 分类 + 分页）| 公开 |
| GET | `/movie/detail` | 电影详情（登录时附「是否收藏 / 观看进度」）| 公开（可选登录）|
| GET | `/movie/search` | 关键词搜索 | 公开 |
| POST | `/movie/reportPlay` | 播放上报（Redis 计数）| 公开 |

### 点播 · 播放链路

> 播放凭证走 **ticket**（不透明随机串 + Redis，可即时吊销），**不是 JWT**；`/vod/play` 用 JWT 换 ticket。

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/vod/play` | 开播：验登录 + 并发检查 + 发 ticket + 分配水印码 | 登录 |
| GET | `/vod/manifest` | 取 **per-user 动态 manifest**（HLS m3u8 / DASH MPD 文本）| ticket |
| GET | `/vod/key` | **HLS AES-128** 密钥（per-session）★闸门 | ticket |
| POST | `/vod/license` | **DASH ClearKey** license（W3C ClearKey JSON）★闸门 | ticket |
| GET | `/vod/segment` | 分片分发（验签名 + **变体白名单**；HLS 线现加密 / CENC 线直出密文）| 签名 URL |
| POST | `/vod/heartbeat` | 心跳续期（会话存活 + 续 ticket）| ticket |
| POST | `/vod/stop` | 结束会话 | ticket |

### 点播 · 观看功能

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/movie/reportProgress` | 上报播放进度（前端节流）| 登录 |
| GET | `/movie/progress` | 取单片进度（续播定位）| 登录 |
| GET | `/movie/history` | 观看历史（按最近观看倒序）| 登录 |
| POST | `/movie/fav/:id` | 收藏电影 | 登录 |
| DELETE | `/movie/fav/:id` | 取消收藏 | 登录 |
| GET | `/movie/fav/status` | 是否已收藏 | 登录 |
| GET | `/movie/fav/list` | 我的电影收藏 | 登录 |

### 账号

| 方法 | 路径 | 说明 | 鉴权 JWT |
|------|------|------|------|
| POST | `/user/register` | 注册（注册即登录，返回 token）| 公开 |
| POST | `/user/login` | 登录，返回 token | 公开 |
| GET | `/user/getProfile` | 获取用户资料 | 公开 |
| POST | `/user/updateProfile` | 修改昵称 / 简介 | 登录 |
| POST | `/user/changePassword` | 修改密码 | 登录 |
| POST | `/user/uploadAvatar` | 上传头像 | 登录 |

### 视频

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/video/upload` | 上传视频（multipart）| 登录 |
| GET | `/video/detail` | 视频详情 | 公开 |
| GET | `/video/list` | 视频列表（游标分页）| 公开 |
| DELETE | `/video/:id` | 删除视频 | 登录 |
| POST | `/video/reportPlay` | 播放上报（播放到 50% 计数 +1）| 公开 |

### 关注

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/follow/:id` | 关注用户 | 登录 |
| DELETE | `/follow/:id` | 取消关注 | 登录 |
| GET | `/follow/list` | 关注列表 | 登录 |
| GET | `/follow/fans` | 粉丝列表 | 登录 |

### Feed

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | `/feed` | 刷关注流 | 登录 |

### 点赞

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/like/:id` | 点赞 | 登录 |
| DELETE | `/like/:id` | 取消点赞 | 登录 |
| GET | `/like/status` | 是否已赞 | 登录 |

### 收藏

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | `/fav/:id` | 收藏 | 登录 |
| DELETE | `/fav/:id` | 取消收藏 | 登录 |
| GET | `/fav/status` | 是否已收藏 | 登录 |
| GET | `/fav/list` | 我的收藏列表 | 登录 |

## 目录结构

```
feed/
├── main.go              # 入口（:2222）
├── handler/             # 接口层
├── service/             # 业务层（Redis / MQ / 事务）
├── dao/                 # 数据访问层
├── model/               # 模型
├── config/              # 配置（环境变量）
├── data/                # 运行时数据：avatar/ video/ vod/（母版·分片）poster/ nfo/
├── feed-baseline/       # 无优化版后端（:8081，性能对比对照组）
├── feed-frontend/       # 前端（Vue 3 + Vite）
└── feed-exp/            # 性能压测工具（seed + bench + scenarios）
```

## 启动教程

### 0. 环境依赖

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | 1.25+ | 后端 |
| Node.js | 18+ | 前端 |
| MySQL | 8.x | 主数据库（需建库，如 `feed`）|
| Redis | — | 缓存 / 计数 / 限流 / 点播会话 |
| RabbitMQ | — | feed 异步扩散 + 点播打包任务（优化版必需）|
| ffmpeg | — | 视频抽帧、提取时长（上传必需）；点播 HLS 分片 |
| GPAC（MP4Box）| — | **仅点播 CENC 加密分片需要**。Fedora 可 `dnf install gpac` |

### 1. 启动后端（优化版）

```bash
cd feed

# 配置环境变量（不设置则用默认值）
export DSN="root:密码@tcp(127.0.0.1:3306)/feed?charset=utf8mb4&parseTime=True&loc=Local"
export JWT_SECRET="你的jwt密钥"

go run .
# 服务启动于 http://localhost:2222
```

- 启动时会自动连接 MySQL / Redis / RabbitMQ，三者都需就绪
- 表结构由 GORM 自动建表（`AutoMigrate`）
- 默认配置见 `config/config.go`，也可用环境变量覆盖（`DSN` / `JWT_SECRET` / `PORT` / `REDIS_ADDR` 等）

**点播模块相关的环境变量**（不设置则用开发默认值，**生产务必覆盖**）：

| 变量 | 默认 | 说明 |
|------|------|------|
| `VOD_DIR` | `./data/vod` | 母版 / 分片 / manifest 根目录 |
| `VOD_URL_SECRET` | `dev-...` | 分片签名 HMAC 密钥 |
| `VOD_KEK` | `dev-...` | 内容密钥**信封加密主密钥** |
| `VOD_ADMIN_KEY` | `dev-admin-key` | 管理端 `X-Admin-Key` |
| `VOD_MAX_SESSIONS` | `2` | 单用户最大并发播放会话 |
| `PUBLIC_BASE_URL` | `http://localhost:2222` | manifest 里 license / 分片的绝对地址前缀 |
| `MP4BOX_BIN` | `MP4Box` | CENC 打包工具路径 |
| `NFO_DIR` / `POSTER_DIR` | `./data/nfo` / `./data/poster` | NFO 留档 / 海报 |

### 2. 启动前端

```bash
cd feed-frontend
npm install
npm run dev
# 打开 http://localhost:5173
```

- 后端已配置 CORS 允许 `http://localhost:5173`（含点播需要的 `X-Admin-Key` 头）
- 前端需要登录后使用；各模块接口见上方「接口名单」
- **点播播放页需 `shaka-player`**（前端已装）；ClearKey 走 EME，**必须在 `https` 或 `localhost` 下运行**

### 3. 启动无优化后端（性能对照组）

```bash
cd feed-baseline

# 同样配置 MySQL（与优化版共用同一份数据，保证压测公平）
export DSN="root:密码@tcp(127.0.0.1:3306)/feed?charset=utf8mb4&parseTime=True&loc=Local"

go run .
# 服务启动于 http://localhost:8081
```

- 这是**退化版**：去掉了 Redis 缓存/计数、MQ 异步扩散、N+1 批量查询、分布式锁等所有优化
- **接口契约与优化版完全一致**，仅内部实现不同——用于性能对比

### 4. 性能压测（feed-exp）

压测工具独立于项目，可单独使用：

```bash
cd feed-exp

# ① 灌数据（先启动优化版/无优化版后端的 MySQL）
cd seed
go run . -mysql-dsn "root:密码@tcp(127.0.0.1:3306)/feed?charset=utf8mb4&parseTime=True&loc=Local" -reset

# ② 压测（两个版本各跑一遍相同场景，结果写入 results/）
cd ../bench
go run . -url http://localhost:2222 -scenario detail_hot -version opt -concurrency 500 -duration 60s
go run . -url http://localhost:8081 -scenario detail_hot -version base -concurrency 500 -duration 60s
```

- 内置 8 个压测场景（`mix` / `detail_hot` / `detail_cold` / `detail_miss` / `list` / `feed` / `reportPlay` / `like`）
- 输出 QPS / 平均延迟 / P50 / P95 / P99 / 错误率，可对比两版差异

## 点播（VOD）模块说明

点播与短视频**并行存在、各占一套表与一套链路**，共用账号 / JWT / Redis / RabbitMQ。

**一部电影 = 一组产物**：上架后经**离线流水线**产出多份 rendition（码率 × 加密方案），分发走 **HLS / DASH 分片流式**。

### 两条并行的加密链路

| 链路 | 加密 | 封装 | 密钥分发 |
|------|------|------|---------|
| **主线** | **CENC**（ISO 23001-7）| **DASH**（MPD）| **ClearKey**（EME 标准，license 返回 JSON）|
| **辅线** | **AES-128** | **HLS**（m3u8）| `#EXT-X-KEY` 指向 key URI |

两条线的**加密模型不同**：HLS 线**分发时按会话现加密**（per-session key，磁盘存明文分片）；CENC 线**打包时一次性静态加密**（内容级同一把 key，磁盘存密文分片）。

### 打包流水线

```
admin 上传母版A + 母版B ──► 落盘 + movie(打包中) ──► 丢 MQ 任务
                                                      ↓
   worker: ① 逐片 IDR 分析（校验 A/B 对齐 + 导出段边界）
           ② remux + 分片（A/B 各一份，段边界由 ① 导出）
           ③ 加密打包（CENC / AES-128；A/B 用同一把 content key）
                                                      ↓
              回写 movie_rendition + content_key + segment_count ──► 状态=就绪
```

- **打包工具链**：HLS 线用 `ffmpeg`（`-c copy` remux + 按段切分片）；CENC 线用 **GPAC `MP4Box`**（`ffmpeg -f dash` 会**静默忽略**加密参数，退出码仍为 0，不可用）。
- **不做分发侧重编码**：A/B 母版由运营离线产出（grain 种子微扰 + qpfile 对齐 IDR），系统只做**转封装 + 分片 + 加密**，码流比特级复制、零质量损失。

### A/B 取证水印

每个分片编码**两份等长的变体 A/B**；服务端为每个播放会话分配一个二进制码，per-user manifest 按码逐段选 A 或 B —— 每个用户拿到的排列独一无二。泄密后逐段比对还原出码 → 查 `playback_session` → **定位到具体会话 / 用户**。

- **本体轴**（差异来源）：编码期的 grain 种子微扰（离线产出）
- **载体轴**（per-user 怎么选）：manifest 层的二进制码
- **CENC 分片的 IV 自包含在每个分片里**，所以「A/B 分片互换后用同一把 key 解密」成立。

### 授权闸门

播放链路的所有安全策略（并发、设备、吊销、时效）都收敛到 `/vod/key` 与 `/vod/license` 两个端点

- **播放凭证**用不透明随机串 + Redis（**不用 JWT**）—— 因为要求**可即时吊销**（发现盗号立刻踢人），无状态在此是负担。
- **并发会话限制**用 ZSET，**原子 Lua** 完成「清死会话 + 判并发 + 加入」（拆三条命令会超卖）。
- **防伪造靠访问控制**：分片 URL 的签名覆盖**变体位**，服务端校验「请求的变体位 == 本会话被分配的那个」，不符即 403 并记异常。

### 已知边界

- **ClearKey 不保护密钥**：license 响应里密钥是明文 base64 —— 这是 W3C 标准如此，不承诺抗密钥提取；真要「密钥只在 TEE 里解出」需要 Widevine L1 级厂商证书。本模块的价值在 CENC 打包 / EME 握手 / license server 架构，不在「密钥保护」。
- **CDM 安全等级（L1/L3）不在范围内** —— ClearKey CDM 是纯软件实现，无硬件信任根。
- **水印抗重编码，但实质性再转码（裁切 / 缩放 / 强降噪）无法溯源** —— 业界同类方案（含 Netflix）同此，是取证水印的固有边界。
- **多账号 / 慢速 / 分布式收集全部变体不可防** —— 靠「变体绑定会话 + 异常检测」抬高成本，不承诺解决。

## 性能对比结果

**压测环境**：500 并发 · 60s · 预热 5s · 每场景 3 轮取均值 · 连接池 `MaxOpenConns=200` · 同机压测。

| 项 | 配置 |
|----|------|
| CPU | Intel Core i7-1165G7 @ 2.80GHz（4 核 8 线程）|
| 内存 | 16 GB |
| 数据量 | 1 万用户 / 5 万视频（seed 默认参数）|
| 存储 | MySQL / Redis / RabbitMQ 均在本机 |

**各场景对比（优化版 → 无优化版）：**

| 场景 | QPS | P99 延迟 | 平均延迟 | 错误率 |
|------|-----|---------|---------|--------|
| 热点详情| 34,195 → 10,449（**3.3×**）| 30 → 124ms | 14.2 → 47.5ms | 0% → 1.9% |
| 冷详情| 32,663 → 10,382（**3.1×**）| 33 → 124ms | 14.9 → 47.8ms | 0% → 2.0% |
| 缓存穿透| 49,027 → 16,334（**3.0×**）| 25 → 79ms | 9.4 → 30.2ms | 0% → 1.5% |
| 频列表| 4,633 → 1,334（**3.5×**）| 282 → 602ms | 107 → 374ms | 2.5% → 15.5% |
| 关注流| 4,306 → 757（**5.7×**）| 286 → 1,030ms | 115 → 658ms | 2.8% → 26.7% |
| 播放上报| 11,509 → 5,998（**1.9×**）| 86 → 279ms | 43.1 → 83.2ms | 2.3% → 56.0% |
| 混合流量| 7,147 → 1,826（**3.9×**）| 430 → 3,166ms | 209 → 820ms | 4.2% → 13.1% |
