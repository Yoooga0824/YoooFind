package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerPort    string
	AllowedOrigin string
	AdminEmail    string

	ModelProviders                map[string]ProviderConfig
	ModelOrder                    []string
	MaxTokens                     int
	Temperature                   float64
	UpstreamRequestTimeoutSeconds int

	MySQLDSN       string
	JWTSecret      string
	JWTExpiryHours int

	AvatarUploadDir string
	AvatarMaxBytes  int64

	WebSearchProvider   string
	WebSearchMaxResults int
	TavilyBaseURL       string
	TavilyAPIKey        string
}

type ProviderConfig struct {
	BaseURL string
	Path    string
	APIKey  string
	Model   string
}

type ProviderBlueprint struct {
	Key            string // Key 模型在 map 里的键名，也会出现在 ModelOrder，例如 "deepseek"。
	EnvPrefix      string // EnvPrefix 环境变量前缀，例如 "DEEPSEEK" → DEEPSEEK_API_KEY、DEEPSEEK_BASE_URL 等。
	DefaultBaseURL string
	DefaultPath    string
	DefaultModel   string
}

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
		DefaultModel:   "",
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

func Load() (Config, error) {
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
		AvatarMaxBytes:                getEnvInt64("AVATAR_MAX_BYTES", 20*1024*1024),                                    // 头像大小上限 3MB
		WebSearchProvider:             strings.ToLower(strings.TrimSpace(getEnv("WEB_SEARCH_PROVIDER", "tavily"))),
		WebSearchMaxResults:           getEnvInt("WEB_SEARCH_MAX_RESULTS", 5),                                 // 搜索条数上限
		TavilyBaseURL:                 strings.TrimSpace(getEnv("TAVILY_BASE_URL", "https://api.tavily.com")), // Tavily API 地址
		TavilyAPIKey:                  strings.TrimSpace(getEnv("TAVILY_API_KEY", "")),                        // Tavily 密钥，空=不启用搜索
	}

	cfg.ModelProviders, cfg.ModelOrder = loadProviderConfigs()
	if len(cfg.ModelProviders) == 0 {
		return Config{}, fmt.Errorf("At least one Model Provider API is required")
	}
	if cfg.MySQLDSN == "" {
		return Config{}, fmt.Errorf("MYSQL_DSN is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.AdminEmail == "" {
		return Config{}, fmt.Errorf("ADMIN_EMAIL is required")
	}

	//预防超时配置非法
	if cfg.UpstreamRequestTimeoutSeconds <= 0 {
		cfg.UpstreamRequestTimeoutSeconds = 120
	}

	return cfg, nil
}

func loadProviderConfigs() (map[string]ProviderConfig, []string) {
	providers := map[string]ProviderConfig{}
	order := make([]string, 0, len(providerCatalog))

	register := func(spec ProviderBlueprint) {
		apiKey := strings.TrimSpace(getEnv(spec.EnvPrefix+"_API_KEY", ""))
		if apiKey == "" {
			return
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

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
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
