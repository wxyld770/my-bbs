# 野集前端

论坛前端使用 React、TypeScript 与 Vite 构建，所有业务请求均使用同源 `/api` 路径。

## 本地开发

```bash
npm ci
npm run dev
```

开发服务器会把 `/api`、`/livez` 和 `/readyz` 代理到 `127.0.0.1:18080`。

## 验证与构建

```bash
npm run test
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

帖子点赞使用幂等目标状态接口：`PUT /api/posts/:id/like` 确保已点赞，
`DELETE /api/posts/:id/like` 确保未点赞。旧的 `POST` 点赞切换接口已移除。

所有 API 请求默认 20 秒超时，并分别报告网络失败、超时和调用方主动取消。
登录只在 Token 与用户资料均成功取得后提交会话；退出会先清理本地凭据。

访问登录接口后，JWT 保存在浏览器本地存储中。生产环境应使用 HTTPS，避免密码和 Token 通过明文 HTTP 传输。
