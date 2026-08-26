## 项目介绍

一个基于 Go 语言、Gin 框架和 GORM 构建的轻量级 BBS（论坛/广场）项目。  
学习 Go 语法、Gin 基础、GORM 之后，用来检验学习成果的实战练习。

## 项目目标

- 将 Go 语法、Gin 路由与中间件、GORM 数据库操作串联起来
- 实现具备用户体系和帖子管理的简易 BBS
- 熟悉分层架构（Handler → Service → Repository）与模块化组装
- 掌握 JWT 认证、密码加密、统一响应、业务异常等常见能力

## 技术栈

| 技术 | 用途 |
|---|---|
| Go 1.26+ | 后端主语言 |
| Gin | Web 框架，路由与中间件 |
| GORM + MySQL | ORM 与数据持久化 |
| Redis + go-redis | Token 撤销存储（必需依赖） |
| JWT | 带独立 JTI 的登录认证 |
| bcrypt | 密码哈希 |
| 标准库 log + channel | 异步日志落盘 |

## 项目结构

```text
my-bbs/
├── .github/workflows/ci.yml    # PR / main 的后端、前端检查与发布包构建
├── cmd/main.go                 # 入口：配置、日志、DB、优雅启停
├── config/.env                 # 本地配置（不入库）
├── internal/
│   ├── config/                 # 配置加载
│   ├── database/               # DB 初始化与迁移
│   ├── redisstore/             # Redis 配置、连接池与生命周期
│   ├── authsession/            # Token 撤销的 Redis 语义
│   ├── logger/                 # 异步日志
│   ├── model/                  # User / Post / Comment / PostLike / BaseModel
│   ├── repository/             # Repository Port 接口与通用错误
│   │   └── gormrepo/           # GORM Repository Adapter
│   ├── service/                # 业务逻辑，只依赖必要的抽象契约
│   ├── handler/                # HTTP 处理
│   │   ├── httprequest/        # 独立请求模型、严格 JSON 绑定与验证错误转换
│   │   └── httpresponse/       # 独立对外响应模型及边界转换
│   ├── middleware/             # Auth / OptionalAuth / 日志 / Recovery / ErrorHandler
│   ├── modules/                # 模块 DI + 路由注册（user/post/comment/like）
│   └── router/                 # 路由组装
├── pkg/
│   ├── bizerr/                 # 业务异常
│   ├── response/               # 统一响应
│   ├── jwt/                    # jwt加解密
│   └── bcrypt/                 # 密码加解密
├── web/                        # React + TypeScript 前端
├── logs/                       # 日志目录（gitignore）
└── readme.md
```

## 核心功能

### 用户
- ✅ 注册（用户名唯一索引，密码 bcrypt）
- ✅ 登录（返回带独立 JTI 的 JWT）
- ✅ 管理员账号由 `ADMIN_USERNAMES` 配置（逗号分隔，仅授权已有账号）
- ✅ 管理员可禁言和解除禁言普通用户
- ✅ 当前用户资料 `GET /user/me`
- ✅ 修改昵称 / 个人介绍
- ✅ 退出（服务端撤销当前 Token，不影响同一用户的其他会话）

### 帖子
- ✅ 发布、修改、删除（作者可删除自己的帖子，管理员可删除任意帖子）
- ✅ 广场列表（仅 `visible=1`，分页）
- ✅ 详情（仅公开帖；私密帖请走个人主页；含点赞/评论数与 `is_liked`）
- ✅ 个人主页帖子列表（分页）
- ✅ 设置可见性（0 仅自己 / 1 所有人）
- ✅ 管理员置顶公开帖子（默认 24 小时，可提前取消）
- ✅ 今日最热榜单（昨日与今日的公开帖子，固定 Top 10）

### 互动
- ✅ 评论（发表 / 分页列表 / 作者删除）
- ✅ 点赞（切换；返回 `liked` + `like_count`）

### 工程能力
- ✅ 统一响应 `{code,message,data}` + `bizerr`
- ✅ 独立 HTTP Response，避免直接序列化 GORM / Service 对象
- ✅ 独立 HTTP Request：限制 body、拒绝未知字段/多 JSON、统一 400/413/415
- ✅ 统一错误生命周期：Handler 上报、中间件记录原始 cause、安全响应
- ✅ `X-Request-ID` 生成/透传，关联请求日志和错误日志
- ✅ Repository Port 与 GORM Adapter 分离，Service 只依赖 Repository 契约
- ✅ 分页 `pageNo/pageSize/hasMore`（无限下拉，不做 total count）
- ✅ 异步日志（控制台 + `logs/日期.log`）
- ✅ 优雅启停（SIGINT/SIGTERM）
- ✅ 请求 `context.Context` 全链路传递与数据库取消
- ✅ HTTP 超时、数据库连接池与启动连通性检查
- ✅ 存活检查 `/livez`、仅本机可访问的 MySQL + Redis 就绪检查 `/readyz`
- ✅ 登录、注册、读接口和写接口的分级限流（429 + `Retry-After`）
- ✅ JWT 当前 Token 撤销（Redis 记录保留到 Token 原始过期时间）
- ✅ 统一模型字段：`create_time` / `update_time` / `deleted`

## 统一响应示例

成功：

```json
{ "code": 0, "message": "ok", "data": { "token": "..." } }
```

失败：

```json
{ "code": 40105, "message": "用户名或密码错误" }
```

每个响应都包含 `X-Request-ID` Header。客户端可以在反馈服务端错误时提供该值，
用于关联请求日志。带 JSON body 的接口仅接受 `application/json`，请求体上限为 1 MiB。

## 快速开始

### 方式 A：Docker（推荐，最容易复现）

前提：已安装 Docker Desktop。执行：

```bash
docker compose up --build -d
```

这会启动 MySQL、Redis 和应用，数据库表会在应用启动时自动迁移。Redis 开启
AOF，并通过 `redis_data` volume 持久化尚未过期的 Token 撤销记录。服务启动后访问
`http://localhost:8080`。查看应用日志：

```bash
docker compose logs -f app
```

停止服务但保留 MySQL 和 Redis 数据：

```bash
docker compose down
```

> `docker-compose.yml` 中的 MySQL/Redis 密码和 JWT 密钥仅供本地开发；部署到公网前必须替换。

### 方式 B：本地运行

前提：Go 1.26+、本地 MySQL 8+、Redis 7+，且已创建数据库 `my_bbs`。
MySQL 和 Redis 都是启动必需依赖：任一连接检查失败，应用都会拒绝启动。

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 配置

```bash
cp config/.env.example config/.env
```

编辑 `config/.env`（将数据库密码、Redis 密码和 `JWT_SECRET` 替换为自己的值）：

```bash
DB_DSN="root:密码@tcp(127.0.0.1:3306)/my_bbs?charset=utf8mb4&parseTime=True&loc=Local"
DB_MAX_OPEN_CONNS="25"
DB_MAX_IDLE_CONNS="10"
DB_CONN_MAX_LIFETIME="30m"
DB_CONN_MAX_IDLE_TIME="5m"
APP_MODE="debug"
APP_PORT="8080"
LOG_DIR="logs"
JWT_SECRET="换成一段足够长的随机密钥"
ADMIN_USERNAMES="admin"
HTTP_READ_HEADER_TIMEOUT="5s"
HTTP_READ_TIMEOUT="10s"
HTTP_WRITE_TIMEOUT="15s"
HTTP_IDLE_TIMEOUT="60s"
HTTP_SHUTDOWN_TIMEOUT="10s"
HEALTH_CHECK_TIMEOUT="2s"
SEARCH_TIMEOUT="1s"
RATE_LIMIT_MAX_ENTRIES="20000"
RATE_LIMIT_IDLE_TTL="15m"
RATE_LIMIT_LOGIN_REQUESTS="10"
RATE_LIMIT_LOGIN_WINDOW="1m"
RATE_LIMIT_LOGIN_BURST="5"
RATE_LIMIT_REGISTER_REQUESTS="3"
RATE_LIMIT_REGISTER_WINDOW="1m"
RATE_LIMIT_REGISTER_BURST="2"
RATE_LIMIT_SEARCH_REQUESTS="60"
RATE_LIMIT_SEARCH_WINDOW="1m"
RATE_LIMIT_SEARCH_BURST="15"
RATE_LIMIT_WRITE_REQUESTS="30"
RATE_LIMIT_WRITE_WINDOW="1m"
RATE_LIMIT_WRITE_BURST="10"
RATE_LIMIT_READ_REQUESTS="600"
RATE_LIMIT_READ_WINDOW="1m"
RATE_LIMIT_READ_BURST="120"
REDIS_ADDR="127.0.0.1:6379"
REDIS_PASS=""
```

`ADMIN_USERNAMES` 从环境变量读取管理员账号名，多个账号使用英文逗号分隔，例如
`ADMIN_USERNAMES="admin,moderator"`。应用只给数据库中已经存在且用户名精确匹配的账号
授权，不会创建账号、修改密码或增加数据库角色字段。用户名仍使用原有唯一性校验，重复
注册返回 409。请先确认账号已经存在，再把用户名加入配置；配置为空或账号不存在时应用
拒绝启动，修改名单后需重启服务才能生效。

直接运行应用时会读取 `config/.env`；使用 Docker Compose 时，请在仓库根目录的 `.env`
或部署平台环境变量中设置同名配置。

时长配置使用 Go `time.ParseDuration` 格式，例如 `500ms`、`10s`、`5m`。

限流使用进程内有界令牌桶：登录为每 IP 每分钟持续 10 次、突发 5 次；注册为
每 IP 每分钟持续 3 次、突发 2 次；搜索为每 IP 每分钟持续 60 次、突发 15 次；
写接口优先按已签名用户计数，每分钟持续 30 次、突发 10 次；普通读接口按 IP
每分钟持续 600 次、突发 120 次。超过额度
统一返回 HTTP 429、业务码 `42900`，并附带 `Retry-After`。`REQUESTS/WINDOW`
表示令牌补充速率，`BURST` 表示可立即使用的令牌上限。

应用只信任来自 `127.0.0.0/8` 或 `::1` 本机反向代理写入的 `X-Real-IP`；公网
直连请求携带的同名 Header 会被忽略。若反向代理不在本机，需要先调整代码中的
可信代理边界，不能直接信任任意代理地址。进程内限流不跨应用实例共享，多实例
部署仍需在 Nginx、网关或共享存储层增加全局限流。

生产 Nginx 模板位于 `deploy/nginx/`：入口对通用 API、写操作、登录、注册和
`/livez` 分级限流，每 IP 并发 API 请求上限为 20；Nginx 产生的 429 也返回统一
JSON 业务码 `42900`。共享限流区文件只能安装一次，修改配置后必须先执行
`nginx -t`，通过后再 reload。

> 若表仍是旧字段（`created_at` 等），开发环境可删表后重启，让 AutoMigrate 重建；或手动改列名。

### 3. 运行

```bash
go run cmd/main.go
```

服务监听 `http://localhost:{APP_PORT}`。`Ctrl+C` 触发优雅关闭。

也可使用 Makefile：

```bash
make run       # 本地启动
make test      # 运行测试
make vet       # 静态检查
make up        # Docker 启动
make down      # 停止 Docker 服务
```

### 测试原则

- 以公开函数、接口和 HTTP 行为为测试边界，优先使用 `xxx_test` 外部测试包。
- 不以覆盖每个私有函数为目标；私有实现重构不应迫使契约测试跟着修改。
- 数据库 Adapter 的行为通过 Repository 公开接口验证；MySQL 专属错误码应由真实 MySQL 集成测试覆盖，而不是直接调用私有错误转换函数。
- 本项目统一将测试放在 `tests/`，按被测层或包分目录，并使用 `xxx_test` 外部测试包验证公开契约。

### 持续集成与发布包

`.github/workflows/ci.yml` 在 Pull Request 和 `main` 更新时分别执行后端测试、
`go vet`、Linux/amd64 编译，以及前端类型检查和生产构建。`Backend` 与
`Frontend` 是分支保护使用的稳定检查名称。

只有可信的 `main` 运行会继续执行 `Package`：它不会重新构建，而是把同一次
运行中已经通过检查的后端二进制和前端构建产物组合成与完整 Git commit SHA
绑定的不可变发布包。发布包不含 `.env`、数据库、Redis、Nginx、systemd 或 TLS
密钥，也不包含服务器上的部署脚本或配置文件。

### Redis 生命周期封装

`internal/redisstore` 只扩展配置、连接池、启动 `PING` 和幂等关闭，不重复包装
go-redis 的各类命令。`*redisstore.Client` 本身实现 `redis.UniversalClient`，可以直接
调用 String、Hash、List、Set、ZSet、Stream、Lua、Pipeline、订阅等完整 API。
向 Service 注入时，构造参数使用更窄的官方 `redis.Cmdable`，避免把 `Close` 纳入
Service 的依赖契约。

```go
startupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

redisClient, err := redisstore.Open(startupCtx, redisstore.Config{
	Addr:         cfg.RedisAddr,
	Password:     cfg.RedisPass,
	PoolSize:     20,
	MinIdleConns: 2,
})
if err != nil {
	return err
}
defer redisClient.Close()

if err := redisClient.HSet(ctx, "demo:hash", "name", "alice").Err(); err != nil {
	return err
}
_, err = redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
	pipe.SAdd(ctx, "demo:set", "go", "redis")
	pipe.ZIncrBy(ctx, "demo:ranking", 1, "item:1")
	return nil
})
```

向 Service 注入 Redis 时只传入命令接口：

```go
type ExampleService struct {
	redis redis.Cmdable
}

func NewExampleService(redisClient redis.Cmdable) *ExampleService {
	return &ExampleService{redis: redisClient}
}

func Initialize(redisClient redis.Cmdable) *Module {
	svc := NewExampleService(redisClient) // 直接传入，不需要额外取 Client
	return &Module{Service: svc}
}
```

依赖链为 `main → redisstore.Open/Close → module → service(redis.Cmdable)`；Service
不导入 `internal/redisstore`，也不创建或关闭连接。缺失 Key 保留 go-redis 原生的
`redis.Nil` 语义。当前只将 Redis 注入 Token 撤销所需的 Service 和认证中间件。

#### Token 撤销

Redis 当前只用于 Token 撤销。登录生成的 JWT 包含独立 JTI；
`POST /api/logout` 会将当前 JTI 写入 `mybbs:v1:auth:revoked:{jti}`，
TTL 等于 Token 剩余有效期。当前 Token 立即失效，其他登录会话不受影响。
Redis 不可用时认证请求返回 503；旧版本签发的无 JTI Token 需重新登录。

## 接口一览

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | `/livez` | 否 | 进程存活检查，不访问外部依赖 |
| GET | `/readyz` | 仅本机 | 服务就绪检查，限定时间内检查 MySQL 和 Redis；公网统一隐藏为 404 |
| POST | `/api/register` | 否 | 注册 |
| POST | `/api/login` | 否 | 登录，返回带独立 JTI 的 Token |
| POST | `/api/logout` | 是 | 立即撤销当前 Token |
| GET | `/api/user/me` | 是 | 当前登录用户资料 |
| POST | `/api/user/profile` | 是 | 修改昵称/介绍 |
| GET | `/api/search` | 否 | 搜索用户和公开帖子，支持 `q`/`scope`/`pageNo`/`pageSize` |
| GET | `/api/posts` | 否 | 广场（公开帖，支持 `pageNo`/`pageSize`） |
| GET | `/api/posts/hot` | 否 | 今日最热榜单（最多 10 篇） |
| GET | `/api/posts/:id` | 可选 | 详情（仅公开帖；带 Token 时返回 `is_liked`） |
| POST | `/api/posts/create` | 是 | 发帖 |
| POST | `/api/posts/update/:id` | 是 | 修改（作者） |
| POST | `/api/posts/del/:id` | 是 | 删除（作者或管理员） |
| POST | `/api/posts/pin/:id` | 管理员 | 置顶公开帖子 24 小时 |
| POST | `/api/posts/unpin/:id` | 管理员 | 提前取消置顶 |
| POST | `/api/posts/visible/:id` | 是 | 设置可见性 |
| POST | `/api/users/:id/mute` | 管理员 | 禁言普通用户 |
| POST | `/api/users/:id/unmute` | 管理员 | 解除普通用户禁言 |
| POST | `/api/user/posts` | 是 | 我的帖子（支持 `pageNo`/`pageSize`） |
| GET | `/api/posts/:id/comments` | 否 | 评论列表（公开帖，分页） |
| POST | `/api/posts/:id/comments/create` | 是 | 发表评论 |
| POST | `/api/comments/del/:id` | 是 | 删除评论（评论作者） |
| POST | `/api/posts/:id/like` | 是 | 切换点赞 |

认证 Header：`Authorization: Bearer <token>`

健康检查成功返回 `200 {"status":"ok"}`；MySQL、Redis 任一不可用或检查超时，
`/readyz` 返回 `503 {"status":"unavailable"}`。`/readyz` 只接受 loopback 客户端；
通过本机 Nginx 转发的公网请求仍按真实客户端 IP 判断并返回 404。生产监控应直接请求
`http://127.0.0.1:{APP_PORT}/readyz`。

退出当前会话：

```bash
curl -X POST http://localhost:8080/api/logout \
  -H 'Authorization: Bearer <token>'
```

分页查询（广场 / 我的帖子）：

```text
GET  /api/posts?pageNo=1&pageSize=10
POST /api/user/posts?pageNo=1&pageSize=10
```

响应 `data`：

```json
{
  "list": [],
  "pageNo": 1,
  "pageSize": 10,
  "hasMore": true
}
```

默认 `pageNo=1`，`pageSize=10`，`pageSize` 上限 50；分页起始位置（`(pageNo-1)*pageSize`）上限为 5000。非正整数、非数字或超过上限返回 400。
客户端根据 `hasMore` 决定是否继续下拉加载，无需 total；到达允许的最深分页时，
即使本页刚好填满也会返回 `hasMore=false`，不会引导客户端请求必然被拒绝的下一页。
帖子列表项只返回标题、作者、可见性和互动计数等摘要字段，不返回 `content`；需要正文时调用 `GET /api/posts/:id`。
广场会把尚未到期的置顶帖排在普通帖子之前；列表和详情通过 `is_pinned`、
`pinned_until` 表示当前置顶状态，置顶到期后统一返回 `false` 和 `null`。

今日最热从应用本地时区的昨日 00:00 到明日 00:00 之间发布的公开帖子中选取，
服务端按 `评论数 × 0.600 + 点赞数 × 0.400` 计算三位小数热度。排序依次为热度、
评论数、发布时间和帖子 ID 降序；最多返回 10 篇，不使用更早的帖子补位，空榜返回
`list: []`。前端展示榜单顺序及评论、点赞数量，不展示热度数值。

全局搜索：

```text
GET /api/search?q=Go&scope=all&pageNo=1&pageSize=10
```

`q` 去除并合并空白后为 2～50 个字符；`scope` 支持 `all`（默认）、`users`、
`posts`。搜索分页大小上限为 20、起始位置上限为 1000。响应将用户和公开帖子
放在两个独立分页对象中；私密帖、软删除记录和帖子完整正文不会返回。搜索查询
超过 `SEARCH_TIMEOUT` 时返回 503，超过独立搜索额度时返回 429。

设置可见性 body：

```json
{ "visible": 0 }
```

更新帖子 body（部分更新，至少传一个字段；未传字段保持不变）：

```json
{ "title": "新标题" }
```

`title` 或 `content` 不能是空字符串或仅空白字符。

帖子详情 `data`：

```json
{
  "post": { "id": 1, "title": "...", "user": { "id": 1, "username": "..." } },
  "like_count": 3,
  "comment_count": 5,
  "is_liked": false
}
```

发表评论 body：

```json
{ "content": "说得对" }
```

点赞切换响应 `data`：

```json
{ "liked": true, "like_count": 4 }
```

## 后续计划

- □ 业务数据缓存
- □ 提高边界场景测试覆盖率
- □ OpenAPI / Swagger

## 反馈

欢迎交流学习心得；发现问题可提 Issue / PR。  
本项目主要用于个人学习，不计划直接用于生产环境。
