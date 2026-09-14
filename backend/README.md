# Go Backend (Beginner Friendly)

这个目录是前端聊天页面对应的 Go 后端代理层。

## 为什么要加后端？

- 前端不再保存 API Key（更安全）
- 可以统一处理错误、限流、日志
- 后续接数据库更方便

## 目录结构

```text
backend/
├── cmd/server/main.go                  # 程序入口：组装所有层并启动 HTTP 服务
├── internal/config/config.go           # 读取环境变量配置
├── internal/model/chat.go              # 请求/响应的数据结构
├── internal/provider/openai_client.go  # 调用上游大模型接口
├── internal/service/chat_service.go    # 业务逻辑（校验、调用 provider）
├── internal/api/router.go              # 路由 + CORS
└── internal/api/handlers/chat_handler.go # HTTP Handler
```

## 启动步骤

1. 复制环境变量模板

```bash
cp backend/.env.example backend/.env
```

2. 编辑 `backend/.env`，填入你的 `UPSTREAM_API_KEY`

3. 启动后端

```bash
cd backend
go run ./cmd/server
```

4. 保持前端仍然运行（`npm start`）

前端默认请求 `http://localhost:8080/api/chat`。

## 测试接口

- 健康检查：`GET http://localhost:8080/api/health`
- 聊天接口：`POST http://localhost:8080/api/chat`
