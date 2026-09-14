// package config 声明本文件属于 config 包（配置模块）。
// Go 里同一目录下的 .go 文件必须属于同一个 package。
// 本包只做一件事：定义配置结构体 + 从环境变量加载配置。
package config

import (
	"fmt"     // 用于格式化错误信息，例如 Load() 校验失败时 return fmt.Errorf(...)
	"os"      // 读取操作系统环境变量，核心函数 os.Getenv("变量名")
	"strconv" // 字符串转数字：Atoi 转 int，ParseFloat 转 float64，ParseInt 转 int64
	"strings" // 字符串工具：TrimSpace 去首尾空格，ToLower 转小写，拼接等
)

// Config 是整个后端程序的「配置快照」。
// 用法：main.go 里执行 cfg, err := config.Load()，成功后用 cfg.字段名 读取，不再到处 os.Getenv。
type Config struct {
	// ServerPort HTTP 服务监听端口（纯数字字符串，不含冒号），默认 "8080"。
	// 环境变量：SERVER_PORT
	// 用在哪：main.go 拼成 ":8080" 传给 http.ListenAndServe 启动 Web 服务。
	ServerPort string

	// AllowedOrigin 允许跨域（CORS）的前端地址，默认 "http://localhost:3000"。
	// 环境变量：ALLOWED_ORIGIN
	// 用在哪：api/router.go 设置响应头 Access-Control-Allow-Origin，浏览器前端才能调 API。
	AllowedOrigin string

	// AdminEmail 管理员邮箱（加载时会转小写、去空格），拥有后台管理权限。
	// 环境变量：ADMIN_EMAIL
	// 用在哪：AuthService 判断管理员；UserService、AdminService 做权限校验；Router 中间件。
	AdminEmail string

	// ModelProviders 已启用的 LLM 提供商集合。
	// key = 模型标识（如 "deepseek"），value = 该提供商的 BaseURL/Path/APIKey/Model。
	// 由 loadProviderConfigs() 根据 .env 里各厂商的 *_API_KEY 自动填充；没配 Key 的厂商不会出现。
	// 用在哪：main.go 遍历此 map，为每个模型创建 provider.OpenAICompatibleClient。
	ModelProviders map[string]ProviderConfig

	// ModelOrder 前端展示/切换模型时的顺序，只包含「已配置 API Key」的模型。
	// 顺序与 providerCatalog 遍历顺序一致（先 deepseek，再 doubao…）。
	// 用在哪：service.NewChatService(..., cfg.ModelOrder, ...) 决定可用模型列表。
	ModelOrder []string

	// MaxTokens 单次调用上游 LLM 时允许生成的最大 token 数，默认 2048。
	// 环境变量：UPSTREAM_MAX_TOKENS
	// 用在哪：创建 OpenAICompatibleClient 时传入，控制 AI 回复长度上限。
	MaxTokens int

	// Temperature 采样温度（0~1 常见），越高越随机，越低越稳定，默认 0.7。
	// 环境变量：UPSTREAM_TEMPERATURE
	// 用在哪：同上，写入 LLM 请求体，影响回复风格。
	Temperature float64

	// UpstreamRequestTimeoutSeconds 调用 DeepSeek/豆包等上游 API 的最长等待秒数，默认 120。
	// 环境变量：UPSTREAM_REQUEST_TIMEOUT_SECONDS
	// 用在哪：main.go 转成 time.Duration 后传给 LLM 客户端，防止请求永久挂起。
	UpstreamRequestTimeoutSeconds int

	// MySQLDSN MySQL 数据库连接串，必填。
	// 格式示例：user:password@tcp(127.0.0.1:3306)/dbname?parseTime=true
	// 环境变量：MYSQL_DSN
	// 用在哪：database.OpenMySQL(cfg.MySQLDSN)；为空时 Load() 直接报错，程序无法启动。
	MySQLDSN string

	// JWTSecret 签发/校验 JWT 登录 token 的密钥，必填且应足够随机。
	// 环境变量：JWT_SECRET
	// 用在哪：AuthService 签发 token；Router/VisitHandler/FeedbackHandler 校验登录态。
	JWTSecret string

	// JWTExpiryHours 登录 token 有效小时数，默认 168（7 天）。
	// 环境变量：JWT_EXPIRY_HOURS
	// 用在哪：service.NewAuthService(..., cfg.JWTExpiryHours, ...) 设置过期时间。
	JWTExpiryHours int

	// AvatarUploadDir 用户头像文件保存目录，默认 "./uploads/avatars"。
	// 环境变量：AVATAR_UPLOAD_DIR
	// 用在哪：UserHandler 保存上传文件；Router 用其父目录做静态文件访问（/uploads/...）。
	AvatarUploadDir string

	// AvatarMaxBytes 单次头像上传允许的最大字节数，默认 3*1024*1024（3MB）。
	// 环境变量：AVATAR_MAX_BYTES
	// 用在哪：UserHandler 上传前校验大小，防止过大文件占满磁盘。
	AvatarMaxBytes int64

	// WebSearchProvider 联网搜索实现名称（小写），默认 "tavily"。
	// 环境变量：WEB_SEARCH_PROVIDER
	// 用在哪：main.go 若为 "tavily" 且 TavilyAPIKey 非空，则创建 Tavily 搜索客户端注入 LLM。
	WebSearchProvider string

	// WebSearchMaxResults 每次联网搜索返回的最大条数，默认 5。
	// 环境变量：WEB_SEARCH_MAX_RESULTS
	// 用在哪：传给 OpenAICompatibleClient，搜索结果会塞进 prompt 上下文。
	WebSearchMaxResults int

	// TavilyBaseURL Tavily 搜索 API 根地址，默认 "https://api.tavily.com"。
	// 环境变量：TAVILY_BASE_URL
	// 用在哪：websearch.NewTavilyClient(cfg.TavilyBaseURL, cfg.TavilyAPIKey)。
	TavilyBaseURL string

	// TavilyAPIKey Tavily 服务 API 密钥；为空则不会启用联网搜索（即使 WebSearchProvider=tavily）。
	// 环境变量：TAVILY_API_KEY
	// 用在哪：同上，与 WebSearchProvider 一起决定是否创建 searchClient。
	TavilyAPIKey string
}

// ProviderConfig 描述「某一个 LLM 提供商」如何连接上游 API（OpenAI 兼容格式）。
// 不会直接从环境变量一次性读出，而是由 loadProviderConfigs() 按 providerCatalog 逐项组装。
type ProviderConfig struct {
	// BaseURL 上游 API 的域名+路径前缀，例如 "https://api.deepseek.com"。
	// 环境变量：{前缀}_BASE_URL，如 DEEPSEEK_BASE_URL；未设置则用 providerCatalog 里的默认值。
	// 用在哪：HTTP 客户端与 Path 拼接成完整请求 URL。
	BaseURL string

	// Path 聊天补全接口路径，例如 "/v1/chat/completions"。
	// 环境变量：{前缀}_API_PATH，如 DEEPSEEK_API_PATH。
	// 用在哪：与 BaseURL 拼接；不同厂商路径可能不同（见 providerCatalog）。
	Path string

	// APIKey 该提供商的密钥；只有非空时该模型才会注册进 ModelProviders。
	// 环境变量：{前缀}_API_KEY，如 DEEPSEEK_API_KEY（必填才启用该模型）。
	// 用在哪：HTTP 请求头 Authorization: Bearer <APIKey>。
	APIKey string

	// Model 实际调用的模型名，例如 "deepseek-chat"、"qwen-plus"。
	// 环境变量：{前缀}_MODEL，如 DEEPSEEK_MODEL；豆包等需在 .env 自行填写。
	// 用在哪：写入请求 JSON 的 "model" 字段。
	Model string
}

// ProviderBlueprint 是「内置支持的 LLM 列表」的模板，定义环境变量命名规则与默认值。
// 不直接实例化客户端，只用来驱动 loadProviderConfigs() 扫描 .env。
type ProviderBlueprint struct {
	// Key 模型在 map 里的键名，也会出现在 ModelOrder，例如 "deepseek"。
	Key string

	// EnvPrefix 环境变量前缀，例如 "DEEPSEEK" → DEEPSEEK_API_KEY、DEEPSEEK_BASE_URL 等。
	EnvPrefix string

	// DefaultBaseURL 未设置 *_BASE_URL 时使用的默认上游地址。
	DefaultBaseURL string

	// DefaultPath 未设置 *_API_PATH 时使用的默认接口路径。
	DefaultPath string

	// DefaultModel 未设置 *_MODEL 时使用的默认模型名；空字符串表示必须在 .env 里手动配置。
	DefaultModel string
}

// providerCatalog 程序内置支持的所有 LLM 厂商清单。
// 遍历顺序 = ModelOrder 的默认顺序；想启用某模型：在 .env 配置对应 *_API_KEY。
var providerCatalog = []ProviderBlueprint{
	{
		Key:            "deepseek",                 // 聊天/前端里模型 id
		EnvPrefix:      "DEEPSEEK",                 // 读 DEEPSEEK_API_KEY 等
		DefaultBaseURL: "https://api.deepseek.com", // DeepSeek 官方 API
		DefaultPath:    "/v1/chat/completions",     // OpenAI 兼容聊天接口
		DefaultModel:   "deepseek-chat",            // 默认模型名
	},
	{
		Key:            "doubao",
		EnvPrefix:      "DOUBAO",
		DefaultBaseURL: "https://ark.cn-beijing.volces.com/api/v3", // 火山引擎豆包
		DefaultPath:    "/chat/completions",
		DefaultModel:   "", // 需在 .env 设置 DOUBAO_MODEL（接入点 id）
	},
	{
		Key:            "kimi",
		EnvPrefix:      "KIMI",
		DefaultBaseURL: "https://api.moonshot.cn", // Moonshot Kimi
		DefaultPath:    "/v1/chat/completions",
		DefaultModel:   "moonshot-v1-8k",
	},
	{
		Key:            "qwen",
		EnvPrefix:      "QWEN",
		DefaultBaseURL: "https://dashscope.aliyuncs.com/compatible-mode", // 阿里通义兼容模式
		DefaultPath:    "/v1/chat/completions",
		DefaultModel:   "qwen-plus",
	},
	{
		Key:            "glm",
		EnvPrefix:      "GLM",
		DefaultBaseURL: "https://open.bigmodel.cn/api/paas", // 智谱 GLM
		DefaultPath:    "/v4/chat/completions",
		DefaultModel:   "glm-4-flash",
	},
}

// Load 从环境变量读取全部配置，校验必填项，成功返回 Config 和 nil error。
// 这是 config 包对外的唯一入口；main.go 启动时第一个调用的就是 config.Load()。
// 用法：cfg, err := config.Load(); if err != nil { log.Fatalf(...) }
func Load() (Config, error) {
	// 先构造 Config 结构体，用 getEnv/getEnvInt 等从环境变量填字段；查不到的用默认值。
	cfg := Config{
		ServerPort:                    getEnv("SERVER_PORT", "8080"),                                                    // HTTP 端口
		AllowedOrigin:                 getEnv("ALLOWED_ORIGIN", "http://localhost:3000"),                                // CORS 来源
		AdminEmail:                    strings.ToLower(strings.TrimSpace(getEnv("ADMIN_EMAIL", "17582495726@163.com"))), // 管理员邮箱，统一小写
		MaxTokens:                     getEnvInt("UPSTREAM_MAX_TOKENS", 2048),                                           // LLM 最大 token
		Temperature:                   getEnvFloat("UPSTREAM_TEMPERATURE", 0.7),                                         // LLM 温度
		UpstreamRequestTimeoutSeconds: getEnvInt("UPSTREAM_REQUEST_TIMEOUT_SECONDS", 120),                               // 上游超时秒数
		MySQLDSN:                      strings.TrimSpace(getEnv("MYSQL_DSN", "")),                                       // 数据库连接串
		JWTSecret:                     strings.TrimSpace(getEnv("JWT_SECRET", "")),                                      // JWT 密钥
		JWTExpiryHours:                getEnvInt("JWT_EXPIRY_HOURS", 168),                                               // token 有效期（小时）
		AvatarUploadDir:               getEnv("AVATAR_UPLOAD_DIR", "./uploads/avatars"),                                 // 头像目录
		AvatarMaxBytes:                getEnvInt64("AVATAR_MAX_BYTES", 3*1024*1024),                                     // 头像大小上限 3MB
		WebSearchProvider: strings.ToLower(strings.TrimSpace( // 联网搜索提供商，转小写便于比较
			getEnv("WEB_SEARCH_PROVIDER", "tavily"),
		)),
		WebSearchMaxResults: getEnvInt("WEB_SEARCH_MAX_RESULTS", 5),                                 // 搜索条数上限
		TavilyBaseURL:       strings.TrimSpace(getEnv("TAVILY_BASE_URL", "https://api.tavily.com")), // Tavily API 地址
		TavilyAPIKey:        strings.TrimSpace(getEnv("TAVILY_API_KEY", "")),                        // Tavily 密钥，空=不启用搜索
	}

	// 扫描 providerCatalog，把配置了 API Key 的厂商填入 ModelProviders 和 ModelOrder。
	cfg.ModelProviders, cfg.ModelOrder = loadProviderConfigs()
	// 至少需要一个 LLM 的 API Key，否则无法聊天。
	if len(cfg.ModelProviders) == 0 {
		return Config{}, fmt.Errorf("at least one provider API key is required") // 返回零值 Config + 错误
	}
	// MySQL 连接串必填，否则无法存用户/聊天记录。
	if cfg.MySQLDSN == "" {
		return Config{}, fmt.Errorf("MYSQL_DSN is required")
	}
	// JWT 密钥必填，否则无法登录鉴权。
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	// 管理员邮箱必填，用于区分管理员权限。
	if cfg.AdminEmail == "" {
		return Config{}, fmt.Errorf("ADMIN_EMAIL is required")
	}
	// 防御性修正：若超时配置成 0 或负数，强制改回 120 秒。
	if cfg.UpstreamRequestTimeoutSeconds <= 0 {
		cfg.UpstreamRequestTimeoutSeconds = 120
	}

	return cfg, nil // 校验通过，返回完整配置
}

// getEnv 读取字符串环境变量；若未设置或为空字符串，则返回 fallback 默认值。
// 内部工具函数，不导出（小写开头），仅供本包 Load/loadProviderConfigs 使用。
func getEnv(key, fallback string) string {
	v := os.Getenv(key) // 从操作系统/ .env（由 main 里 godotenv 加载）读取
	if v == "" {        // 空字符串视为「未配置」
		return fallback // 使用代码里的默认值
	}
	return v // 使用环境变量里的值
}

// loadProviderConfigs 遍历 providerCatalog，把「配了 API Key」的厂商注册到 map 和顺序切片。
// 返回值1：map[模型key]ProviderConfig，供 main 创建 LLM 客户端。
// 返回值2：[]string 模型顺序，供 ChatService 展示可用模型列表。
func loadProviderConfigs() (map[string]ProviderConfig, []string) {
	providers := map[string]ProviderConfig{}         // 存放已启用的提供商
	order := make([]string, 0, len(providerCatalog)) // 预分配容量，减少扩容；记录启用顺序
	// register 是「闭包函数」：处理 catalog 里某一个厂商，有 Key 就注册，没有就跳过。
	register := func(spec ProviderBlueprint) {
		// 例如 DEEPSEEK_API_KEY；TrimSpace 去掉 .env 里 accidental 空格
		apiKey := strings.TrimSpace(getEnv(spec.EnvPrefix+"_API_KEY", ""))
		if apiKey == "" {
			return // 没配 Key = 不启用该模型，直接 return 跳过
		}
		key := strings.ToLower(strings.TrimSpace(spec.Key)) // 模型 id 统一小写，如 "deepseek"
		providers[key] = ProviderConfig{
			BaseURL: strings.TrimSpace(getEnv(spec.EnvPrefix+"_BASE_URL", spec.DefaultBaseURL)), // 可覆盖默认 URL
			Path:    strings.TrimSpace(getEnv(spec.EnvPrefix+"_API_PATH", spec.DefaultPath)),    // 可覆盖默认路径
			APIKey:  apiKey,                                                                     // 必填，上面已校验非空
			Model:   strings.TrimSpace(getEnv(spec.EnvPrefix+"_MODEL", spec.DefaultModel)),      // 可覆盖默认模型名
		}
		order = append(order, key) // 按 catalog 顺序追加到 ModelOrder
	}

	for _, item := range providerCatalog { // 依次处理 deepseek、doubao、kimi、qwen、glm
		register(item)
	}
	return providers, order
}

// getEnvInt 读取整数环境变量；未设置、为空或解析失败时返回 fallback。
// 用在哪：MaxTokens、JWTExpiryHours、超时秒数、搜索条数等整型配置。
func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v) // 字符串 → int，失败 err != nil
	if err != nil {
		return fallback // 例如写了 "abc" 则用默认值，避免程序崩溃
	}
	return n
}

// getEnvFloat 读取浮点数环境变量；未设置、为空或解析失败时返回 fallback。
// 用在哪：Temperature 等 float64 配置。
func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(v, 64) // 64 表示 float64 精度
	if err != nil {
		return fallback
	}
	return n
}

// getEnvInt64 读取 int64 环境变量；未设置、为空或解析失败时返回 fallback。
// 用在哪：AvatarMaxBytes 等可能超过 int 范围的字节数配置。
func getEnvInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64) // 10 进制，64 位
	if err != nil {
		return fallback
	}
	return n
}
