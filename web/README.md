# 野集前端

论坛前端使用 React、TypeScript 与 Vite 构建，所有业务请求均使用同源 `/api` 路径。

## 本地开发

```bash
npm ci
npm run dev
```

开发服务器会把 `/api`、`/livez` 和 `/readyz` 代理到 `127.0.0.1:18080`。

## 构建

```bash
npm run typecheck
npm run build
```

构建产物位于 `dist/`，由 nginx 托管；SPA 路由需要回退到 `index.html`。

## 页面

- `/`：公开广场
- `/search?q=关键词&scope=all`：搜索用户与公开帖子
- `/post/:id`：帖子详情、点赞与评论
- `/me`：个人资料与帖子管理
- `/u/:id`：公开用户主页

访问登录接口后，JWT 保存在浏览器本地存储中。生产环境应使用 HTTPS，避免密码和 Token 通过明文 HTTP 传输。
