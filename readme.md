## 项目介绍
一个基于 Go 语言、Gin 框架和 GORM 构建的轻量级 BBS（论坛/广场）项目。
 - 学习 Go 语法、Gin 基础、GORM 之后，用来检验学习成果的实战练习。 

## 项目目标

这是我在完成 Go 语言基础学习（包括标准库、并发模型、结构体与接口）后，进一步学习 Gin 和 GORM 时产生的练手项目。主要目标：
- 将所学的 Go 语法、Gin 路由与中间件、GORM 数据库操作串联起来
- 实现一个具备完整用户体系和帖子管理功能的简易 BBS
- 熟悉分层架构（Handler → Service → Repository）在实际项目中的运用
- 掌握 JWT 认证、密码加密、RESTful API 设计等常见业务场景

## 技术栈

|技术 / 库|版本|用途|
|---|---|---|
|Go|1.21+|后端主语言|
|Gin|v1.10.0|Web 框架，路由与中间件|
|GORM|v1.25.5|ORM，数据库操作|
|GORM MySQL Driver|v1.5.2|连接 MySQL 数据库|
|golang-jwt/jwt|v5.2.0|生成与解析 JWT Token|
|[golang.org/x/crypto/bcrypt](https://golang.org/x/crypto/bcrypt)|v0.18.0|用户密码加密|
## 项目结构

```text
my-bbs/
├── cmd/
│   └── main.go                # 程序入口
├── internal/
│   ├── config/                # 配置加载（数据库连接、JWT密钥等）
│   ├── model/                 # 数据模型（User、Post）
│   ├── repository/            # 数据访问层（CRUD）
│   ├── service/               # 业务逻辑层（注册、发帖等）
│   ├── handler/               # HTTP 处理器（解析请求、返回响应）
│   ├── middleware/            # 中间件（JWT 认证）
│   └── router/                # 路由注册
├── pkg/                       # 可复用工具（JWT、密码加密）
├── go.mod
└── README.md
```

> 分层设计参考了 Go 社区常见的项目布局，有利于代码维护和测试。

## ✨ 核心功能（规划 / 已实现）

### 用户模块
- ✅ 用户注册（用户名唯一，密码 bcrypt 加密）
- ✅ 用户登录（验证密码，返回 JWT Token）
- ✅ 用户退出（客户端移除 Token，服务端无状态）
### 帖子模块
- ✅ 发布帖子（需登录）
- ✅ 查看所有帖子（广场）
- ✅ 查看单篇帖子（可公开访问）
- ✅ 查看个人所有帖子（个人主页）
- ✅ 修改帖子（验证当前用户是否为作者）
- ✅ 删除帖子（验证当前用户是否为作者）

> 注：当前为学习阶段实现，未来可扩展评论、点赞、关注等功能。

---

## 🚀 快速开始

### 1. 克隆项目

bash

git clone https://github.com/wxyld770/my-bbs.git
cd my-bbs

### 2. 安装依赖

bash

go mod tidy

### 3. 配置数据库

- 创建 MySQL 数据库（如 `my_bbs`）
    
- 修改 `config/.env.example` 为  `config/.env` 
- 修改 ` .env ` 中的数据库连接
    

### 4. 运行服务

bash

go run cmd/main.go

服务默认运行在 `http://localhost:8080`。

### 5. 接口测试（使用 Postman 或 curl）

- `POST /api/register` – 注册
    
- `POST /api/login` – 登录
    
- `GET /api/posts/list` – 广场（公开）
    
- `POST /api/posts/create` – 发帖（需认证，Header: `Authorization: Bearer <token>`）
    
- `POST /api/user/posts` – 个人主页（需认证）
    
- `POST /api/posts/update/:id` – 修改（需认证且为作者）
    
- `POST /api/posts/del/:id` – 删除（需认证且为作者）
    
---

## 📚 学习收获

通过这个项目，我加深了对以下知识的理解：

- **Go 语法进阶**：结构体方法、接口、错误处理、包管理
    
- **Gin 框架**：路由分组、中间件、参数绑定、响应渲染
    
- **GORM**：模型定义、自动迁移、关联查询（后续可扩展）
    
- **项目组织**：分层架构、依赖注入思想、配置管理
    
- **实战技巧**：JWT 无状态认证、密码加密、RESTful API 设计
    

---

## 📌 后续计划（个人扩展）
    
- □ 
    
    增加评论功能（帖子详情页）
    
- □ 
    
    使用 `golang.org/x/sync/errgroup` 优化并发查询
    
- □ 
    
    编写单元测试（针对 service 和 repository）
    
- □ 
    
    添加 Swagger 文档（使用 `swaggo/gin-swagger`）
    

---

## 💬 反馈与交流

如果你也是 Go 初学者，欢迎交流学习心得；如果发现了代码中的不足或 bug，也欢迎提 Issue / PR。

> 本项目主要用于个人学习，目前不计划用于生产环境，但代码风格和结构可以供初学者参考。