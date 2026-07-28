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
│   ├── model/                  # User / Post / BaseModel
│   ├── repository/             # 数据访问
│   ├── service/                # 业务逻辑
│   ├── handler/                # HTTP 处理
│   ├── middleware/             # Auth / 日志 / Recovery / ErrorHandler
│   ├── modules/                # 模块 DI + 路由注册
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
- ✅ 修改昵称 / 个人介绍
- ✅ 退出（客户端丢弃 Token）

### 帖子
- ✅ 发布、修改、删除（作者校验）
- ✅ 广场列表（仅 `visible=1`）
- ✅ 详情（仅公开帖；私密帖请走个人主页）
- ✅ 个人主页帖子列表
- ✅ 设置可见性（0 仅自己 / 1 所有人）

### 工程能力
- ✅ 统一响应 `{code,message,data}` + `bizerr`
- ✅ 异步日志（控制台 + `logs/日期.log`）
- ✅ 优雅启停（SIGINT/SIGTERM）
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

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 配置

```bash
cp config/.env.example config/.env
```

编辑 `config/.env`：

```bash
DB_DSN="root:密码@tcp(127.0.0.1:3306)/库名?charset=utf8mb4&parseTime=True"
APP_MODE="debug"
APP_PORT="8080"
LOG_DIR="logs"
JWT_SECRET="换成你的密钥"
```

> 若表仍是旧字段（`created_at` 等），开发环境可删表后重启，让 AutoMigrate 重建；或手动改列名。

### 3. 运行

```bash
go run cmd/main.go
```

服务监听 `http://localhost:{APP_PORT}`。`Ctrl+C` 触发优雅关闭。

## 接口一览

| 方法 | 路径 | 认证 | 说明 |
|---|---|---|---|
| POST | `/api/register` | 否 | 注册 |
| POST | `/api/login` | 否 | 登录，返回 token |
| POST | `/api/user/profile` | 是 | 修改昵称/介绍 |
| GET | `/api/posts` | 否 | 广场（公开帖） |
| GET | `/api/posts/:id` | 否 | 详情（仅公开帖；私密帖 404） |
| POST | `/api/posts/create` | 是 | 发帖 |
| POST | `/api/posts/update/:id` | 是 | 修改（作者） |
| POST | `/api/posts/del/:id` | 是 | 删除（作者） |
| POST | `/api/posts/visible/:id` | 是 | 设置可见性 |
| POST | `/api/user/posts` | 是 | 我的帖子 |

认证 Header：`Authorization: Bearer <token>`

设置可见性 body：

```json
{ "visible": 0 }
```

## 后续计划

- □ 分页 + errgroup 并发 count
- □ 评论 / 点赞
- □ Redis 缓存
- □ 单测 + Swagger + Docker

## 反馈

欢迎交流学习心得；发现问题可提 Issue / PR。  
本项目主要用于个人学习，不计划直接用于生产环境。
