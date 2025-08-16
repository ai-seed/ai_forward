package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type dbConfig struct {
	Driver            string        `mapstructure:"driver"`
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	User              string        `mapstructure:"user"`
	Password          string        `mapstructure:"password"`
	DBName            string        `mapstructure:"dbname"`
	SSLMode           string        `mapstructure:"sslmode"`
	MaxOpenConns      int           `mapstructure:"max_open_conns"`
	MaxIdleConns      int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime   time.Duration `mapstructure:"conn_max_lifetime"`
	KeepAliveInterval time.Duration `mapstructure:"keep_alive_interval"` // 连接保活间隔
}

// Config 应用配置
type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Database     dbConfig           `mapstructure:"database"`
	Logging      LoggingConfig      `mapstructure:"logging"`
	RateLimit    RateLimitConfig    `mapstructure:"rate_limiting"`
	LoadBalance  LoadBalanceConfig  `mapstructure:"load_balancer"`
	Monitoring   MonitoringConfig   `mapstructure:"monitoring"`
	Billing      BillingConfig      `mapstructure:"billing"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	OAuth        OAuthConfig        `mapstructure:"oauth"`
	FunctionCall FunctionCallConfig `mapstructure:"function_call"`
	Thinking     ThinkingConfig     `mapstructure:"thinking"`
	S3           S3Config           `mapstructure:"s3"`
	Midjourney   MidjourneyConfig   `mapstructure:"midjourney"`
	UPay         UPayConfig         `mapstructure:"upay"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	DefaultRequestsPerMinute int `mapstructure:"default_requests_per_minute"`
	DefaultRequestsPerHour   int `mapstructure:"default_requests_per_hour"`
	DefaultRequestsPerDay    int `mapstructure:"default_requests_per_day"`
}

// LoadBalanceConfig 负载均衡配置
type LoadBalanceConfig struct {
	Strategy           string `mapstructure:"strategy"`
	HealthCheckEnabled bool   `mapstructure:"health_check_enabled"`
	FailoverEnabled    bool   `mapstructure:"failover_enabled"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	HealthCheckPath string `mapstructure:"health_check_path"`
}

// BillingConfig 计费配置
type BillingConfig struct {
	Currency  string `mapstructure:"currency"`
	Precision int    `mapstructure:"precision"`
	BatchSize int    `mapstructure:"batch_size"`
}

// JWTConfig JWT认证配置
type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	Issuer          string        `mapstructure:"issuer"`
	Audience        string        `mapstructure:"audience"`
}

// OAuthConfig OAuth认证配置
type OAuthConfig struct {
	Google      OAuthProviderConfig `mapstructure:"google"`
	GitHub      OAuthProviderConfig `mapstructure:"github"`
	FrontendURL string              `mapstructure:"frontend_url"`
}

// OAuthProviderConfig OAuth提供商配置
type OAuthProviderConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	Enabled      bool   `mapstructure:"enabled"`
}

// FunctionCallConfig Function Call 配置
type FunctionCallConfig struct {
	Enabled       bool         `mapstructure:"enabled"`
	SearchService SearchConfig `mapstructure:"search_service"`
}

// SearchConfig 搜索服务配置
type SearchConfig struct {
	Service        string `mapstructure:"service"`          // 搜索服务类型
	MaxResults     int    `mapstructure:"max_results"`      // 最大结果数
	CrawlResults   int    `mapstructure:"crawl_results"`    // 深度搜索数量
	CrawlContent   bool   `mapstructure:"crawl_content"`    // 是否爬取网页内容并转换为Markdown
	Search1APIKey  string `mapstructure:"search1api_key"`   // Search1API密钥
	GoogleCX       string `mapstructure:"google_cx"`        // Google自定义搜索引擎ID
	GoogleKey      string `mapstructure:"google_key"`       // Google API密钥
	BingKey        string `mapstructure:"bing_key"`         // Bing搜索API密钥
	SerpAPIKey     string `mapstructure:"serpapi_key"`      // SerpAPI密钥
	SerperKey      string `mapstructure:"serper_key"`       // Serper密钥
	SearXNGBaseURL string `mapstructure:"searxng_base_url"` // SearXNG服务地址
}

// ThinkingConfig 深度思考配置
type ThinkingConfig struct {
	Enabled     bool                      `mapstructure:"enabled"`     // 是否启用深度思考功能
	Default     ThinkingDefaultConfig     `mapstructure:"default"`     // 默认配置
	Performance ThinkingPerformanceConfig `mapstructure:"performance"` // 性能配置
}

// ThinkingDefaultConfig 思考默认配置
type ThinkingDefaultConfig struct {
	ShowProcess bool   `mapstructure:"show_process"` // 是否默认显示思考过程
	MaxTokens   int    `mapstructure:"max_tokens"`   // 默认最大思考token数
	Language    string `mapstructure:"language"`     // 默认思考语言
}

// ThinkingPerformanceConfig 思考性能配置
type ThinkingPerformanceConfig struct {
	Timeout     time.Duration `mapstructure:"timeout"`      // 思考处理超时时间
	EnableCache bool          `mapstructure:"enable_cache"` // 是否启用思考过程缓存
	CacheTTL    time.Duration `mapstructure:"cache_ttl"`    // 缓存TTL
}

// S3Config S3存储配置
type S3Config struct {
	Enabled         bool     `mapstructure:"enabled"`           // 是否启用S3存储
	Region          string   `mapstructure:"region"`            // AWS区域
	Bucket          string   `mapstructure:"bucket"`            // S3存储桶名称
	AccessKeyID     string   `mapstructure:"access_key_id"`     // AWS访问密钥ID
	SecretAccessKey string   `mapstructure:"secret_access_key"` // AWS秘密访问密钥
	Endpoint        string   `mapstructure:"endpoint"`          // 自定义端点（用于兼容S3的服务）
	UsePathStyle    bool     `mapstructure:"use_path_style"`    // 是否使用路径样式URL
	MaxFileSize     int64    `mapstructure:"max_file_size"`     // 最大文件大小（字节）
	AllowedTypes    []string `mapstructure:"allowed_types"`     // 允许的文件类型
}

// MidjourneyConfig Midjourney队列配置
type MidjourneyConfig struct {
	ChannelSize  int `mapstructure:"channel_size"`  // 任务队列channel大小
	WorkerCount  int `mapstructure:"worker_count"`  // 工作进程数量
	MaxRetries   int `mapstructure:"max_retries"`   // 最大重试次数
	PollInterval int `mapstructure:"poll_interval"` // 轮询间隔（秒）
}

// UPayConfig UPay支付配置
type UPayConfig struct {
	Enabled   bool   `mapstructure:"enabled"`    // 是否启用UPay支付
	AppID     string `mapstructure:"app_id"`     // UPay应用ID
	AppSecret string `mapstructure:"app_secret"` // UPay应用密钥

	NotifyURL       string  `mapstructure:"notify_url"`       // 支付回调通知URL
	DefaultChain    string  `mapstructure:"default_chain"`    // 默认链路类型 (1:TRC20, 2:ERC20, 3:PYUSD)
	DefaultCurrency string  `mapstructure:"default_currency"` // 默认法币类型
	MinAmount       float64 `mapstructure:"min_amount"`       // 最小支付金额
	MaxAmount       float64 `mapstructure:"max_amount"`       // 最大支付金额
}

// LoadConfig 加载配置
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")
	}

	// 设置环境变量
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析配置
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认值
func setDefaults() {
	// 服务器默认值
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "60s")

	// 数据库默认值
	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "300s")
	viper.SetDefault("database.keep_alive_interval", "30s") // 默认30秒保活间隔

	// 日志默认值
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")

	// 速率限制默认值
	viper.SetDefault("rate_limiting.default_requests_per_minute", 60)
	viper.SetDefault("rate_limiting.default_requests_per_hour", 1000)
	viper.SetDefault("rate_limiting.default_requests_per_day", 10000)

	// 负载均衡默认值
	viper.SetDefault("load_balancer.strategy", "round_robin")
	viper.SetDefault("load_balancer.health_check_enabled", true)
	viper.SetDefault("load_balancer.failover_enabled", true)

	// 监控默认值
	viper.SetDefault("monitoring.health_check_path", "/health")

	// 计费默认值
	viper.SetDefault("billing.currency", "USD")
	viper.SetDefault("billing.precision", 6)
	viper.SetDefault("billing.batch_size", 100)

	// JWT默认值
	viper.SetDefault("jwt.secret", "your-super-secret-jwt-key-change-this-in-production")
	viper.SetDefault("jwt.access_token_ttl", "24h")
	viper.SetDefault("jwt.refresh_token_ttl", "168h")
	viper.SetDefault("jwt.issuer", "ai-api-gateway")
	viper.SetDefault("jwt.audience", "ai-api-gateway-users")

	// OAuth默认值
	viper.SetDefault("oauth.frontend_url", "http://localhost:3000")
	viper.SetDefault("oauth.google.enabled", false)
	viper.SetDefault("oauth.google.client_id", "")
	viper.SetDefault("oauth.google.client_secret", "")
	viper.SetDefault("oauth.google.redirect_url", "http://localhost:8080/auth/oauth/google/callback")
	viper.SetDefault("oauth.github.enabled", false)
	viper.SetDefault("oauth.github.client_id", "")
	viper.SetDefault("oauth.github.client_secret", "")
	viper.SetDefault("oauth.github.redirect_url", "http://localhost:8080/auth/oauth/github/callback")

	// Function Call默认值
	viper.SetDefault("function_call.enabled", false)
	viper.SetDefault("function_call.search_service.service", "duckduckgo")
	viper.SetDefault("function_call.search_service.max_results", 10)
	viper.SetDefault("function_call.search_service.crawl_results", 0)

	// S3存储默认值
	viper.SetDefault("s3.enabled", false)
	viper.SetDefault("s3.region", "us-east-1")
	viper.SetDefault("s3.bucket", "")
	viper.SetDefault("s3.access_key_id", "")
	viper.SetDefault("s3.secret_access_key", "")
	viper.SetDefault("s3.endpoint", "")
	viper.SetDefault("s3.use_path_style", false)
	viper.SetDefault("s3.max_file_size", 10*1024*1024) // 10MB
	viper.SetDefault("s3.allowed_types", []string{"image/jpeg", "image/png", "image/gif", "image/webp", "application/pdf", "text/plain"})

	// Midjourney默认值
	viper.SetDefault("midjourney.channel_size", 1000)
	viper.SetDefault("midjourney.worker_count", 3)
	viper.SetDefault("midjourney.max_retries", 60)
	viper.SetDefault("midjourney.poll_interval", 5)

	// UPay默认值
	viper.SetDefault("upay.enabled", false)
	viper.SetDefault("upay.app_id", "")
	viper.SetDefault("upay.app_secret", "")
	viper.SetDefault("upay.notify_url", "")
	viper.SetDefault("upay.default_chain", "1")      // 默认TRC20
	viper.SetDefault("upay.default_currency", "USD") // 默认美元
	viper.SetDefault("upay.min_amount", 1.0)
	viper.SetDefault("upay.max_amount", 10000.0)
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// 验证服务器配置
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	// 验证数据库配置
	if config.Database.Driver == "" {
		return fmt.Errorf("database driver is required")
	}

	// 验证日志配置
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLogLevels[config.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	// 验证负载均衡策略
	validStrategies := map[string]bool{
		"round_robin": true, "weighted": true, "least_connections": true, "random": true,
	}
	if !validStrategies[config.LoadBalance.Strategy] {
		return fmt.Errorf("invalid load balance strategy: %s", config.LoadBalance.Strategy)
	}

	return nil
}

// GetAddress 获取服务器地址
func (c *ServerConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
