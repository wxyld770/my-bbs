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
| JWT | 无状态登录认证 |
| bcrypt | 密码哈希 |
| 标准库 log + channel | 异步日志落盘 |

## 项目结构

```text
my-bbs/
├── cmd/main.go                 # 入口：配置、日志、DB、优雅启停
├── config/.env                 # 本地配置（不入库）
├── internal/
│   ├── config/                 # 配置加载
│   ├── database/               # DB 初始化与迁移
│   ├── logger/                 # 异步日志
│   ├── model/                  # User / Post / Comment / PostLike / BaseModel
│   ├── repository/             # Repository Port 接口与通用错误
│   │   └── gormrepo/           # GORM Repository Adapter
│   ├── service/                # 业务逻辑，只依赖 Repository Port
│   ├── handler/                # HTTP 处理
│   │   └── httpresponse/       # 独立对外响应模型及边界转换
│   ├── middleware/             # Auth / OptionalAuth / 日志 / Recovery / ErrorHandler
│   ├── modules/                # 模块 DI + 路由注册（user/post/comment/like）
│   └── router/                 # 路由组装
├── pkg/
│   ├── bizerr/                 # 业务异常
│   ├── response/               # 统一响应
│   ├── jwt/                    # jwt加解密
│   └── bcrypt/                 # 密码加解密
├── logs/                       # 日志目录（gitignore）
└── readme.md
```

## 核心功能

### 用户
- ✅ 注册（用户名唯一索引，密码 bcrypt）
- ✅ 登录（返回 JWT）
- ✅ 当前用户资料 `GET /user/me`
- ✅ 修改昵称 / 个人介绍
- ✅ 退出（客户端丢弃 Token）

### 帖子
- ✅ 发布、修改、删除（作者校验）
- ✅ 广场列表（仅 `visible=1`，分页）
- ✅ 详情（仅公开帖；私密帖请走个人主页；含点赞/评论数与 `is_liked`）
- ✅ 个人主页帖子列表（分页）
- ✅ 设置可见性（0 仅自己 / 1 所有人）

### 互动
- ✅ 评论（发表 / 分页列表 / 作者删除）
- ✅ 点赞（切换；返回 `liked` + `like_count`）

### 工程能力
- ✅ 统一响应 `{code,message,data}` + `bizerr`
- ✅ 独立 HTTP Response，避免直接序列化 GORM / Service 对象
- ✅ Repository Port 与 GORM Adapter 分离，Service 只依赖 Repository 契约
- ✅ 分页 `pageNo/pageSize/hasMore`（无限下拉，不做 total count）
- ✅ 异步日志（控制台 + `logs/日期.log`）
- ✅ 优雅启停（SIGINT/SIGTERM）
- ✅ 请求 `context.Context` 全链路传递与数据库取消
- ✅ HTTP 超时、数据库连接池与启动连通性检查
- ✅ 存活检查 `/livez`、数据库就绪检查 `/readyz`
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

## 快速开始

### 方式 A：Docker（推荐，最容易复现）

前提：已安装 Docker Desktop。执行：

```bash
docker compose up --build -d
```

这会启动 MySQL 和应用，数据库表会在应用启动时自动迁移。服务启动后访问
`http://localhost:8080`。查看应用日志：

```bash
docker compose logs -f app
```

停止服务但保留数据库数据：

```bash
docker compose down
```

> `docker-compose.yml` 中的数据库密码和 JWT 密钥仅供本地开发；部署到公网前必须替换。

### 方式 B：本地运行

前提：Go 1.26+、本地 MySQL 8+，且已创建数据库 `my_bbs`。

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 配置

```bash
cp config/.env.example config/.env
```

编辑 `config/.env`（将 `DB_DSN` 中的密码和 `JWT_SECRET` 替换为自己的值）：

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
HTTP_READ_HEADER_TIMEOUT="5s"
HTTP_READ_TIMEOUT="10s"
HTTP_WRITE_TIMEOUT="15s"
HTTP_IDLE_TIMEOUT="60s"
HTTP_SHUTDOWN_TIMEOUT="10s"
HEALTH_CHECK_TIMEOUT="2s"
```

时长配置使用 Go `time.ParseDuration` 格式，例如 `500ms`、`10s`、`5m`。

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

## 接口一览

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| GET | `/livez` | 否 | 进程存活检查，不访问外部依赖 |
| GET | `/readyz` | 否 | 服务就绪检查，限定时间内执行数据库 Ping |
| POST | `/api/register` | 否 | 注册 |
| POST | `/api/login` | 否 | 登录，返回 token |
| GET | `/api/user/me` | 是 | 当前登录用户资料 |
| POST | `/api/user/profile` | 是 | 修改昵称/介绍 |
| GET | `/api/posts` | 否 | 广场（公开帖，支持 `pageNo`/`pageSize`） |
| GET | `/api/posts/:id` | 可选 | 详情（仅公开帖；带 Token 时返回 `is_liked`） |
| POST | `/api/posts/create` | 是 | 发帖 |
| POST | `/api/posts/update/:id` | 是 | 修改（作者） |
| POST | `/api/posts/del/:id` | 是 | 删除（作者） |
| POST | `/api/posts/visible/:id` | 是 | 设置可见性 |
| POST | `/api/user/posts` | 是 | 我的帖子（支持 `pageNo`/`pageSize`） |
| GET | `/api/posts/:id/comments` | 否 | 评论列表（公开帖，分页） |
| POST | `/api/posts/:id/comments/create` | 是 | 发表评论 |
| POST | `/api/comments/del/:id` | 是 | 删除评论（评论作者） |
| POST | `/api/posts/:id/like` | 是 | 切换点赞 |

认证 Header：`Authorization: Bearer <token>`

健康检查成功返回 `200 {"status":"ok"}`；数据库不可用或检查超时，`/readyz`
返回 `503 {"status":"unavailable"}`。

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

默认 `pageNo=1`，`pageSize=10`，`pageSize` 上限 50。  
客户端根据 `hasMore` 决定是否继续下拉加载，无需 total。

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

- □ Redis 缓存
- □ 提高边界场景测试覆盖率
- □ OpenAPI / Swagger

## 反馈

欢迎交流学习心得；发现问题可提 Issue / PR。  
本项目主要用于个人学习，不计划直接用于生产环境。
