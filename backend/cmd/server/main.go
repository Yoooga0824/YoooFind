// package main 声明这是「可执行程序」的入口包。
// Go 规定：只有 package main 且含有 func main() 的目录才能 go run / go build 成二进制。
package main

import (
	"fmt"           // 格式化输出，例如 fmt.Printf 打印启动信息
	"log"           // 标准日志：log.Println 普通日志，log.Fatalf 打印后 os.Exit(1) 退出程序
	"net/http"      // Go 标准库 HTTP 服务器，ListenAndServe 监听端口处理请求
	"path/filepath" // 跨平台路径拼接 Join、取目录 Dir，比字符串 "+" 更安全
	"time"          // 时间类型，这里把「秒数」转成 time.Duration 给 LLM 客户端用

	// 以下都是本项目 internal 包（仅本模块内使用，外部项目不应 import）
	"gemini-clone/backend/internal/api"          // 注册 HTTP 路由、中间件（CORS、JWT 等）
	"gemini-clone/backend/internal/api/handlers" // 各 API 的 Handler：登录、聊天、管理后台等
	"gemini-clone/backend/internal/config"       // 读取 .env/环境变量 → Config 结构体
	"gemini-clone/backend/internal/database"     // MySQL 连接 + 执行 SQL 迁移脚本
	"gemini-clone/backend/internal/provider"     // 封装调用各 LLM 上游 API 的 HTTP 客户端
	"gemini-clone/backend/internal/repository"   // 数据访问层：User/Chat/Usage 等表的 CRUD
	"gemini-clone/backend/internal/service"      // 业务逻辑层：Auth、Chat、Admin 等，Handler 只调 Service
	"gemini-clone/backend/internal/websearch"    // 联网搜索（Tavily）客户端接口与实现

	"github.com/joho/godotenv" // 第三方库：启动时把项目根目录 .env 文件加载进 os.Getenv 能读到的环境变量
)

// main 是程序入口，Go 运行时会自动调用；按顺序：加载配置 → 连数据库 → 组装各层 → 启动 HTTP 服务。
func main() {
	// 尝试加载当前工作目录下的 .env 文件；没有 .env 不算致命错误（可用系统环境变量）。
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables") // 提示后继续，不退出
	}

	// 从环境变量组装 Config，并校验 MYSQL_DSN、JWT_SECRET、至少一个 LLM API Key 等。
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err) // 配置无效则无法启动，Fatalf 打印错误并 exit(1)
	}

	// searchClient 是联网搜索客户端，类型为接口 websearch.Client；零值为 nil 表示「不启用搜索」。
	var searchClient websearch.Client
	// 仅当配置为 tavily 且配了 TavilyAPIKey 时才创建客户端，否则 LLM 只能纯本地知识回答。
	if cfg.WebSearchProvider == "tavily" && cfg.TavilyAPIKey != "" {
		searchClient = websearch.NewTavilyClient(cfg.TavilyBaseURL, cfg.TavilyAPIKey)
	}

	// llmClients：map[模型key]*OpenAICompatibleClient，每个已启用的模型对应一个上游 HTTP 客户端。
	// make 第二个参数是容量 hint，len(cfg.ModelProviders) 减少 map 扩容次数。
	llmClients := make(map[string]*provider.OpenAICompatibleClient, len(cfg.ModelProviders))
	for modelKey, providerCfg := range cfg.ModelProviders { // 遍历 config 里注册的所有 LLM
		llmClients[modelKey] = provider.NewOpenAICompatibleClient(
			providerCfg.BaseURL, // 上游 API 根地址
			providerCfg.Path,    // 聊天接口路径
			providerCfg.APIKey,  // Bearer Token
			providerCfg.Model,   // 模型名写入请求 JSON
			cfg.MaxTokens,       // 全局最大生成 token
			cfg.Temperature,     // 全局温度
			time.Duration(cfg.UpstreamRequestTimeoutSeconds)*time.Second, // int 秒 → time.Duration
			searchClient,            // 可为 nil；非 nil 时 LLM 可先联网再回答
			cfg.WebSearchMaxResults, // 搜索条数上限
		)
	}

	// 打开 MySQL 连接池；DSN 来自 .env 的 MYSQL_DSN。
	db, err := database.OpenMySQL(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("open database failed: %v", err)
	}
	defer db.Close() // main 退出前关闭连接，释放资源；defer 表示「函数返回时执行」

	// 执行初始化 SQL：建表等；路径相对「进程当前工作目录」，通常应在 backend 目录下启动。
	if err := database.RunInitMigration(db, filepath.Join("migrations", "001_init.sql")); err != nil {
		log.Fatalf("run migration failed: %v", err)
	}
	if err := database.RunInitMigration(db, filepath.Join("migrations", "002_feedback.sql")); err != nil {
		log.Fatalf("run feedback migration failed: %v", err)
	}

	// Repository 层：封装 SQL，Service/Handler 不直接写 SQL。
	userRepo := repository.NewUserRepository(db)         // 用户表
	usageRepo := repository.NewUsageRepository(db)       // API 用量统计
	chatRepo := repository.NewChatRepository(db)         // 聊天记录
	adminRepo := repository.NewAdminRepository(db)       // 管理后台数据
	feedbackRepo := repository.NewFeedbackRepository(db) // 用户反馈

	// Service 层：业务规则；依赖 Repo + 部分 cfg 字段。
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiryHours, cfg.AdminEmail) // 注册/登录/JWT
	userService := service.NewUserService(userRepo, cfg.AdminEmail)                                    // 用户资料
	usageService := service.NewUsageService(usageRepo, userRepo)                                       // 用量
	adminService := service.NewAdminService(userRepo, adminRepo, cfg.AdminEmail)                       // 后台管理
	feedbackService := service.NewFeedbackService(feedbackRepo, userRepo)                              // 反馈

	// ChatService 需要 map[模型key]Generator 接口；把 llmClients 转成该 map（client 实现了 Generator）。
	modelGenerators := make(map[string]service.Generator, len(llmClients))
	for modelKey, client := range llmClients {
		modelGenerators[modelKey] = client // 同一 client 指针，多态调用 Generate
	}
	chatService := service.NewChatService(modelGenerators, cfg.ModelOrder, chatRepo, usageService) // 聊天核心

	// Handler 层：解析 HTTP 请求/响应 JSON，调用 Service；不含复杂业务逻辑。
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService, cfg.AvatarUploadDir, cfg.AvatarMaxBytes) // 头像路径与大小限制
	usageHandler := handlers.NewUsageHandler(usageService)
	chatHandler := handlers.NewChatHandler(chatService)                   // 流式/非流式聊天 API
	adminHandler := handlers.NewAdminHandler(adminService)                // /api/admin/*
	visitHandler := handlers.NewVisitHandler(adminService, cfg.JWTSecret) // 访问统计等
	feedbackHandler := handlers.NewFeedbackHandler(feedbackService, cfg.JWTSecret)

	// Router 把所有 Handler 挂到路径上，并注入 CORS、JWT 中间件、静态文件目录等。
	router := api.NewRouter(
		chatHandler,
		authHandler,
		userHandler,
		usageHandler,
		adminHandler,
		visitHandler,
		feedbackHandler,
		userRepo,                          // 中间件里有时需查用户
		cfg.AllowedOrigin,                 // CORS
		cfg.JWTSecret,                     // 校验 Authorization Bearer token
		cfg.AdminEmail,                    // 管理员路由鉴权
		filepath.Dir(cfg.AvatarUploadDir), // 静态服务 uploads 父目录，浏览器可访问头像 URL
	)

	addr := ":" + cfg.ServerPort // 例如 ":8080"，ListenAndServe 监听所有网卡的该端口
	fmt.Printf("Go backend listening on http://localhost%s\n", addr)
	fmt.Printf("Health check: http://localhost%s/api/health\n", addr)
	// 阻塞运行直到出错（如端口被占用）；第二个参数 router 实现 http.Handler 接口。
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
