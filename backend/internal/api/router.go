// package api 声明本文件属于 api 包（HTTP 路由层）。
// 职责：把 URL 路径映射到对应的 Handler，并套上 CORS、登录鉴权、管理员鉴权等中间件。
// 用法：main.go 里 router := api.NewRouter(...)，再交给 http.ListenAndServe(addr, router) 启动服务。
package api

import (
	"encoding/json" // 把 Go 结构体/map 编码成 JSON 字符串写回 HTTP 响应（如 health 接口）
	"net/http"      // Go 标准 HTTP 库：ServeMux 路由、Handler、Request/ResponseWriter 等
	"path/filepath" // 安全处理文件路径，Clean 去掉多余 .. 等，防止目录穿越
	"strings"       // 字符串分割、去空格，用于解析多个 CORS 允许来源

	"gemini-clone/backend/internal/api/handlers" // 各业务 HTTP 处理器：登录、聊天、管理后台等
	"gemini-clone/backend/internal/middleware"   // 中间件：RequireAuth 要求登录，RequireAdmin 要求管理员
	"gemini-clone/backend/internal/repository"   // 数据访问层；RequireAdmin 需查数据库确认用户邮箱是否为管理员
)

// NewRouter 创建并返回整个后端的 HTTP 路由树（实现了 http.Handler 接口）。
// 这是 api 包对外的唯一入口；main.go 组装好所有 Handler 和配置后调用此函数。
//
// 参数说明（均由 main.go 传入，大多来自 config.Load() 读到的 cfg）：
//   - chatHandler/authHandler/...：各 API 的具体处理逻辑（解析请求 → 调 Service → 写响应）
//   - userRepo：管理员鉴权时查用户表，比对邮箱是否等于 AdminEmail
//   - allowedOrigin：CORS 允许的前端地址，来自 cfg.AllowedOrigin
//   - jwtSecret：校验 JWT 的密钥，来自 cfg.JWTSecret
//   - adminEmail：管理员邮箱，来自 cfg.AdminEmail
//   - uploadsRoot：静态文件根目录（头像等），来自 filepath.Dir(cfg.AvatarUploadDir)
//
// 返回值 http.Handler：可被 http.ListenAndServe 直接使用；外层包了 withCORS 处理跨域。
func NewRouter(
	chatHandler *handlers.ChatHandler,       // 聊天相关：POST /api/chat、会话列表/详情
	authHandler *handlers.AuthHandler,       // 注册/登录：/api/auth/register、/api/auth/login
	userHandler *handlers.UserHandler,       // 当前用户资料与头像：/api/me、/api/me/avatar
	usageHandler *handlers.UsageHandler,     // 用量统计：GET /api/usage
	adminHandler *handlers.AdminHandler,     // 管理后台：用户列表、访问/token 统计
	visitHandler *handlers.VisitHandler,     // 访问记录：POST /api/visit（前端埋点）
	feedbackHandler *handlers.FeedbackHandler, // 用户反馈 + 管理员查看/处理反馈
	userRepo *repository.UserRepository,     // RequireAdmin 中间件查用户身份用
	allowedOrigin string,                    // 允许跨域的前端 Origin，如 http://localhost:3000
	jwtSecret string,                        // JWT 签名密钥，RequireAuth/RequireAdmin 解析 Authorization 头
	adminEmail string,                       // 管理员邮箱，与登录用户邮箱比对以放行 /api/admin/*
	uploadsRoot string,                      // 上传文件静态目录根路径，映射到 URL /uploads/
) http.Handler {
	// mux 是 Go 内置的多路复用器：根据请求 URL 路径，分发给不同的处理函数。
	// http.NewServeMux() 创建空路由表；后面用 HandleFunc/Handle 逐条注册。
	mux := http.NewServeMux()

	// ---------- 公开接口（无需登录） ----------

	// GET /api/health — 健康检查，部署/监控用；main.go 启动时会打印此地址。
	// 匿名函数作为 Handler：收到请求就返回 JSON {"status":"ok"}。
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8") // 告诉浏览器响应是 UTF-8 JSON
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})  // Encode 写入响应体；错误用 _ 忽略（health 极少失败）
	})

	// POST /api/auth/register — 用户注册；前端注册页调用，成功后通常再 login 拿 JWT。
	mux.HandleFunc("/api/auth/register", authHandler.PostRegister)

	// POST /api/auth/login — 用户登录；返回 JWT，前端存 localStorage，后续请求带 Authorization: Bearer <token>。
	mux.HandleFunc("/api/auth/login", authHandler.PostLogin)

	// POST /api/visit — 记录页面访问（埋点）；一般无需登录，供统计 PV/UV。
	mux.HandleFunc("/api/visit", visitHandler.PostVisit)

	// POST /api/feedback — 提交用户反馈；feedbackHandler 内部可能用 JWT 识别用户（见 handler 实现）。
	mux.HandleFunc("/api/feedback", feedbackHandler.PostFeedback)

	// ---------- 需登录接口（RequireAuth 中间件） ----------
	// RequireAuth 流程：读 Authorization 头 → 解析 JWT → 把 userID 放进 request.Context → 再调真正的 Handler。
	// Handler 里用 middleware.UserIDFromContext(r.Context()) 取当前登录用户 id。

	// /api/me — GET 读资料、PATCH 改资料；同一路径两种方法，用 routeMethods 按 r.Method 分发。
	mux.HandleFunc("/api/me", middleware.RequireAuth(jwtSecret, routeMethods(userHandler.GetMe, userHandler.PatchMe)))

	// POST /api/me/avatar — 上传头像；文件存 cfg.AvatarUploadDir，浏览器通过 /uploads/... 访问。
	mux.HandleFunc("/api/me/avatar", middleware.RequireAuth(jwtSecret, userHandler.PostAvatar))

	// GET /api/usage — 当前用户的 token/API 用量摘要。
	mux.HandleFunc("/api/usage", middleware.RequireAuth(jwtSecret, usageHandler.GetUsageSummary))

	// POST /api/chat — 发送聊天消息（可能流式 SSE）；核心对话接口。
	mux.HandleFunc("/api/chat", middleware.RequireAuth(jwtSecret, chatHandler.PostChat))

	// GET /api/chat/sessions — 当前用户的会话列表。
	mux.HandleFunc("/api/chat/sessions", middleware.RequireAuth(jwtSecret, chatHandler.GetSessions))

	// GET /api/chat/sessions/{id} — 某条会话详情；路径带尾部 / 时 ServeMux 会匹配子路径前缀。
	mux.HandleFunc("/api/chat/sessions/", middleware.RequireAuth(jwtSecret, chatHandler.GetSessionDetail))

	// ---------- 管理员接口（RequireAdmin 中间件） ----------
	// RequireAdmin = 先 RequireAuth，再查 userRepo：用户邮箱必须等于 adminEmail，否则 403。
	// 前端 admin.html 管理后台页面调用这些接口。

	// GET /api/admin/users — 用户列表（分页等逻辑在 adminHandler 内）。
	mux.HandleFunc(
		"/api/admin/users",
		middleware.RequireAdmin(jwtSecret, adminEmail, userRepo, adminHandler.GetUsers),
	)

	// /api/admin/users/{id} — 对单个用户的操作（禁用、改角色等）；路径末尾 / 匹配子路径。
	mux.HandleFunc(
		"/api/admin/users/",
		middleware.RequireAdmin(jwtSecret, adminEmail, userRepo, adminHandler.HandleUserActions),
	)

	// GET /api/admin/stats/visits — 访问统计（与 PostVisit 埋点数据对应）。
	mux.HandleFunc(
		"/api/admin/stats/visits",
		middleware.RequireAdmin(jwtSecret, adminEmail, userRepo, adminHandler.GetVisitStats),
	)

	// GET /api/admin/stats/tokens — Token 用量总览。
	mux.HandleFunc(
		"/api/admin/stats/tokens",
		middleware.RequireAdmin(jwtSecret, adminEmail, userRepo, adminHandler.GetTokenOverview),
	)

	// GET /api/admin/feedback — 管理员查看反馈列表。
	mux.HandleFunc(
		"/api/admin/feedback",
		middleware.RequireAdmin(jwtSecret, adminEmail, userRepo, feedbackHandler.GetFeedback),
	)

	// /api/admin/feedback/{id} — 处理单条反馈（标记已读、回复等）。
	mux.HandleFunc(
		"/api/admin/feedback/",
		middleware.RequireAdmin(jwtSecret, adminEmail, userRepo, feedbackHandler.HandleFeedbackActions),
	)

	// ---------- 静态文件 ----------
	// URL 前缀 /uploads/ 映射到磁盘 uploadsRoot 目录（main 传入的是 AvatarUploadDir 的父目录）。
	// StripPrefix：浏览器请求 /uploads/avatars/xxx.png → 文件服务器在 uploadsRoot 下找 avatars/xxx.png。
	// filepath.Clean：规范化路径，避免 ../ 等不安全片段。
	// Handle（非 HandleFunc）：这里挂载的是 http.Handler 接口，FileServer 已实现该接口。
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(filepath.Clean(uploadsRoot)))))

	// 用 withCORS 包装整个 mux：所有响应都会加 CORS 头，并处理浏览器预检 OPTIONS 请求。
	// 前端（如 localhost:3000）才能跨域调用 localhost:8080 的 API。
	return withCORS(mux, allowedOrigin)
}

// routeMethods 把「同一路径、不同 HTTP 方法」合并成一个 HandlerFunc。
// Go 的 ServeMux 默认不区分 Method，所以需要在 Handler 内部自己判断 r.Method。
//
// 参数：
//   - getHandler：处理 GET 的函数（如 userHandler.GetMe）
//   - patchHandler：处理 PATCH 的函数（如 userHandler.PatchMe）
//
// 用在哪：仅 /api/me 一处；其他路径在各自 Handler 里自行判断 Method 或只支持一种方法。
func routeMethods(getHandler, patchHandler http.HandlerFunc) http.HandlerFunc {
	// 返回的新函数才是最终注册到 mux 上的 Handler。
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet { // http.MethodGet 常量等于 "GET"
			getHandler(w, r) // 转发给 GET 处理函数
			return             // 处理完毕，不再往下执行
		}
		if r.Method == http.MethodPatch { // PATCH 常用于部分更新用户资料
			patchHandler(w, r)
			return
		}
		// 其他方法（POST/DELETE 等）不允许，返回 405 Method Not Allowed + JSON 错误体。
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed) // HTTP 状态码 405
		_, _ = w.Write([]byte(`{"error":{"message":"method not allowed"}}`))
	}
}

// withCORS 为所有路由加上跨域（CORS）支持，包装成外层 http.Handler。
//
// 为什么需要：浏览器前端（不同端口/域名）调 API 时，会先检查服务端是否允许该 Origin；
// 不允许则 JS 读不到响应，表现为「Network 正常但前端报错」。
//
// 参数：
//   - next：内层真正的路由（上面的 mux）
//   - allowedOrigin：来自 config.AllowedOrigin，可以是单个 URL 或逗号分隔多个
//
// 用在哪：NewRouter 最后一行 return withCORS(mux, allowedOrigin)，ListenAndServe 用的是这个外层 Handler。
func withCORS(next http.Handler, allowedOrigin string) http.Handler {
	// 把配置字符串解析成 map，便于 O(1) 判断某个 Origin 是否在白名单里。
	allowedOrigins := parseAllowedOrigins(allowedOrigin)

	// http.HandlerFunc(...) 是函数类型，实现 ServeHTTP 方法，可当作 http.Handler 使用。
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 浏览器跨域请求会带 Origin 头，例如 http://localhost:3000。
		origin := r.Header.Get("Origin")
		// 只有白名单内的 Origin 才回写 Access-Control-Allow-Origin（不能随意写 * 当带 Cookie/Authorization 时）。
		if isOriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin) // 回显具体 Origin，浏览器才放行
		}
		// 告诉浏览器允许哪些 HTTP 方法和请求头（需与前端实际发送的一致）。
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization") // Authorization 用于 JWT
		w.Header().Set("Vary", "Origin") // 缓存提示：响应因 Origin 不同而不同

		// 预检请求：复杂跨域前浏览器先发 OPTIONS，不需执行业务逻辑，204 即可。
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent) // 204 No Content
			return
		}

		// 非 OPTIONS：把请求交给内层 mux，走具体 API Handler。
		next.ServeHTTP(w, r)
	})
}

// parseAllowedOrigins 把环境变量 ALLOWED_ORIGIN 解析成「允许的前端地址集合」。
//
// 支持格式：
//   - 空字符串 → 默认只允许 http://localhost:3000
//   - 单个 URL：http://localhost:3000
//   - 多个 URL 逗号分隔：http://localhost:3000,https://app.example.com
//
// 返回值 map[string]struct{}：用 struct{} 作 value 表示「只关心 key 是否存在」，不占额外内存（常见 Go 惯用法）。
func parseAllowedOrigins(raw string) map[string]struct{} {
	origins := map[string]struct{}{} // 空 map，后面逐个填入

	// 配置为空时给本地开发一个安全默认值，避免 CORS 完全失效或误放行。
	if strings.TrimSpace(raw) == "" {
		origins["http://localhost:3000"] = struct{}{} // struct{}{} 表示 map 里「有这个 key」
		return origins
	}

	// 按逗号拆分多个 Origin，逐个 TrimSpace 去首尾空格。
	for _, item := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(item)
		if origin == "" {
			continue // 跳过空段，例如配置里多写了逗号 ",,"
		}
		origins[origin] = struct{}{} // 加入白名单
	}

	return origins
}

// isOriginAllowed 判断请求头里的 Origin 是否在白名单 map 中。
//
// 参数 origin：来自 r.Header.Get("Origin")，同源简单请求可能为空。
// 参数 allowed：parseAllowedOrigins 的返回值。
//
// 返回 false 时 withCORS 不会设置 Access-Control-Allow-Origin，浏览器会拦截跨域响应。
func isOriginAllowed(origin string, allowed map[string]struct{}) bool {
	if origin == "" {
		return false // 无 Origin 通常不是需要 CORS 的跨域场景，不主动放行
	}
	_, ok := allowed[origin] // map 查找：ok 为 true 表示 origin 在白名单里
	return ok
}
