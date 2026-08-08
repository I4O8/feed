# Feed 关注视频流项目

## 简介

一个用于学习和面试展示的 **feed 关注视频流后端项目**，采用 Go + Gin + GORM + MySQL 技术栈，深度实践了 Redis 缓存与 RabbitMQ 消息队列。

项目按 **MVC 分层**（handler → service → dao → model）开发，包含完整的 **优化版 / 无优化版后端对比** 与 **性能压测工具**，可以用数据直观量化每一项优化的价值（Redis 缓存、N+1 批量查询、分布式锁、MQ 异步扩散等）。

## 功能模块

| 模块 | 功能 |
|------|------|
| **账号** | 注册 / 登录 / 个人资料 / 修改密码 / 上传头像 |
| **视频** | 上传 / 列表 / 详情 / 删除 / 播放上报 |
| **关注** | 关注 / 取关 / 关注列表 / 粉丝列表 |
| **Feed** | 关注流 |
| **点赞** | 点赞 / 取消 / 是否已赞 |
| **收藏** | 收藏 / 取消 / 是否已收藏 / 我的收藏 |
| **前端** | 登录注册 / 视频流 / 个人主页 / 点赞收藏 |

## 接口名单

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
├── data/                # 静态资源（头像/视频，运行时生成）
├── docs/                # 设计文档与接口文档（前端构建依据）
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
| Redis | — | 缓存 / 计数 / 限流 |
| RabbitMQ | — | feed 异步扩散（优化版必需）|
| ffmpeg | — | 视频抽帧、提取时长（上传必需）|

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

### 2. 启动前端

```bash
cd feed-frontend
npm install
npm run dev
# 打开 http://localhost:5173
```

- 后端已配置 CORS 允许 `http://localhost:5173`
- 前端需要登录后使用；接口文档见 `docs/`（各模块 .md）

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

