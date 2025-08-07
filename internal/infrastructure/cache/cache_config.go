package cache

import (
	"time"

	"github.com/spf13/viper"
)

// CacheTTLConfig 缓存TTL配置
type CacheTTLConfig struct {
	// 实体缓存TTL
	UserTTL     time.Duration `mapstructure:"user_ttl"`
	APIKeyTTL   time.Duration `mapstructure:"api_key_ttl"`
	ModelTTL    time.Duration `mapstructure:"model_ttl"`
	ProviderTTL time.Duration `mapstructure:"provider_ttl"`
	QuotaTTL    time.Duration `mapstructure:"quota_ttl"`

	// 查询缓存TTL
	UserLookupTTL    time.Duration `mapstructure:"user_lookup_ttl"`
	ModelListTTL     time.Duration `mapstructure:"model_list_ttl"`
	ProviderListTTL  time.Duration `mapstructure:"provider_list_ttl"`
	QuotaUsageTTL    time.Duration `mapstructure:"quota_usage_ttl"`
	UserQuotaListTTL time.Duration `mapstructure:"user_quota_list_ttl"`
	APIKeyListTTL    time.Duration `mapstructure:"api_key_list_ttl"`
	UsageLogTTL      time.Duration `mapstructure:"usage_log_ttl"`

	// 统计缓存TTL
	CountTTL      time.Duration `mapstructure:"count_ttl"`
	PaginationTTL time.Duration `mapstructure:"pagination_ttl"`

	// 默认TTL
	DefaultTTL time.Duration `mapstructure:"default_ttl"`
}

// CacheTTLManager 缓存TTL配置管理器
type CacheTTLManager struct {
	config *CacheTTLConfig
}

var globalCacheTTLManager *CacheTTLManager

// InitCacheTTLManager 初始化缓存TTL配置管理器
func InitCacheTTLManager() *CacheTTLManager {
	if globalCacheTTLManager != nil {
		return globalCacheTTLManager
	}

	config := &CacheTTLConfig{}

	// 从配置文件加载TTL设置
	if err := viper.UnmarshalKey("cache", config); err != nil {
		// 如果加载失败，使用默认值
		setDefaultTTLConfig(config)
	} else {
		// 确保所有零值字段都有默认值
		fillDefaultTTLValues(config)
	}

	globalCacheTTLManager = &CacheTTLManager{config: config}
	return globalCacheTTLManager
}

// GetCacheTTLManager 获取全局缓存TTL配置管理器
func GetCacheTTLManager() *CacheTTLManager {
	if globalCacheTTLManager == nil {
		return InitCacheTTLManager()
	}
	return globalCacheTTLManager
}

// setDefaultTTLConfig 设置默认的TTL配置
func setDefaultTTLConfig(config *CacheTTLConfig) {
	config.DefaultTTL = 5 * time.Minute

	// 实体缓存TTL
	config.UserTTL = 5 * time.Second
	config.APIKeyTTL = 15 * time.Minute
	config.ModelTTL = 30 * time.Minute
	config.ProviderTTL = 30 * time.Minute
	config.QuotaTTL = 1 * time.Minute

	// 查询缓存TTL
	config.UserLookupTTL = 5 * time.Minute
	config.ModelListTTL = 30 * time.Minute
	config.ProviderListTTL = 30 * time.Minute
	config.QuotaUsageTTL = 2 * time.Minute
	config.UserQuotaListTTL = 5 * time.Minute
	config.APIKeyListTTL = 3 * time.Second
	config.UsageLogTTL = 10 * time.Minute

	// 统计缓存TTL
	config.CountTTL = 10 * time.Minute
	config.PaginationTTL = 5 * time.Minute
}

// fillDefaultTTLValues 填充零值字段的默认值
func fillDefaultTTLValues(config *CacheTTLConfig) {
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 5 * time.Minute
	}

	// 实体缓存TTL
	if config.UserTTL == 0 {
		config.UserTTL = 5 * time.Second
	}
	if config.APIKeyTTL == 0 {
		config.APIKeyTTL = 15 * time.Minute
	}
	if config.ModelTTL == 0 {
		config.ModelTTL = 30 * time.Minute
	}
	if config.ProviderTTL == 0 {
		config.ProviderTTL = 30 * time.Minute
	}
	if config.QuotaTTL == 0 {
		config.QuotaTTL = 1 * time.Minute
	}

	// 查询缓存TTL
	if config.UserLookupTTL == 0 {
		config.UserLookupTTL = 5 * time.Minute
	}
	if config.ModelListTTL == 0 {
		config.ModelListTTL = 30 * time.Minute
	}
	if config.ProviderListTTL == 0 {
		config.ProviderListTTL = 30 * time.Minute
	}
	if config.QuotaUsageTTL == 0 {
		config.QuotaUsageTTL = 2 * time.Minute
	}
	if config.UserQuotaListTTL == 0 {
		config.UserQuotaListTTL = 5 * time.Minute
	}
	if config.APIKeyListTTL == 0 {
		config.APIKeyListTTL = 3 * time.Second
	}
	if config.UsageLogTTL == 0 {
		config.UsageLogTTL = 10 * time.Minute
	}

	// 统计缓存TTL
	if config.CountTTL == 0 {
		config.CountTTL = 10 * time.Minute
	}
	if config.PaginationTTL == 0 {
		config.PaginationTTL = 5 * time.Minute
	}
}

// GetUserTTL 获取用户缓存TTL
func (m *CacheTTLManager) GetUserTTL() time.Duration {
	return m.config.UserTTL
}

// GetAPIKeyTTL 获取API密钥缓存TTL
func (m *CacheTTLManager) GetAPIKeyTTL() time.Duration {
	return m.config.APIKeyTTL
}

// GetModelTTL 获取模型缓存TTL
func (m *CacheTTLManager) GetModelTTL() time.Duration {
	return m.config.ModelTTL
}

// GetProviderTTL 获取提供商缓存TTL
func (m *CacheTTLManager) GetProviderTTL() time.Duration {
	return m.config.ProviderTTL
}

// GetQuotaTTL 获取配额缓存TTL
func (m *CacheTTLManager) GetQuotaTTL() time.Duration {
	return m.config.QuotaTTL
}

// GetUserLookupTTL 获取用户查询缓存TTL
func (m *CacheTTLManager) GetUserLookupTTL() time.Duration {
	return m.config.UserLookupTTL
}

// GetModelListTTL 获取模型列表缓存TTL
func (m *CacheTTLManager) GetModelListTTL() time.Duration {
	return m.config.ModelListTTL
}

// GetProviderListTTL 获取提供商列表缓存TTL
func (m *CacheTTLManager) GetProviderListTTL() time.Duration {
	return m.config.ProviderListTTL
}

// GetQuotaUsageTTL 获取配额使用情况缓存TTL
func (m *CacheTTLManager) GetQuotaUsageTTL() time.Duration {
	return m.config.QuotaUsageTTL
}

// GetUserQuotaListTTL 获取用户配额列表缓存TTL
func (m *CacheTTLManager) GetUserQuotaListTTL() time.Duration {
	return m.config.UserQuotaListTTL
}

// GetAPIKeyListTTL 获取API密钥列表缓存TTL
func (m *CacheTTLManager) GetAPIKeyListTTL() time.Duration {
	return m.config.APIKeyListTTL
}

// GetUsageLogTTL 获取使用日志缓存TTL
func (m *CacheTTLManager) GetUsageLogTTL() time.Duration {
	return m.config.UsageLogTTL
}

// GetCountTTL 获取计数统计缓存TTL
func (m *CacheTTLManager) GetCountTTL() time.Duration {
	return m.config.CountTTL
}

// GetPaginationTTL 获取分页列表缓存TTL
func (m *CacheTTLManager) GetPaginationTTL() time.Duration {
	return m.config.PaginationTTL
}

// GetDefaultTTL 获取默认缓存TTL
func (m *CacheTTLManager) GetDefaultTTL() time.Duration {
	return m.config.DefaultTTL
}

// GetConfig 获取完整的缓存配置
func (m *CacheTTLManager) GetConfig() *CacheTTLConfig {
	return m.config
}
