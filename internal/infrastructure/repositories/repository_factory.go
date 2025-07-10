package repositories

import (
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/redis"

	"gorm.io/gorm"
)

// RepositoryFactory 仓储工厂（基于GORM）
type RepositoryFactory struct {
	gormDB *gorm.DB
	cache  *redis.CacheService
}

// NewRepositoryFactory 创建GORM仓储工厂
func NewRepositoryFactory(gormDB *gorm.DB) *RepositoryFactory {
	return &RepositoryFactory{
		gormDB: gormDB,
		cache:  nil,
	}
}

// NewRepositoryFactoryWithCache 创建带缓存的GORM仓储工厂
func NewRepositoryFactoryWithCache(gormDB *gorm.DB, cache *redis.CacheService) *RepositoryFactory {
	return &RepositoryFactory{
		gormDB: gormDB,
		cache:  cache,
	}
}

// UserRepository 获取用户仓储
func (f *RepositoryFactory) UserRepository() repositories.UserRepository {
	return NewUserRepositoryGorm(f.gormDB, f.cache)
}

// APIKeyRepository 获取API密钥仓储
func (f *RepositoryFactory) APIKeyRepository() repositories.APIKeyRepository {
	return NewAPIKeyRepositoryGorm(f.gormDB, f.cache)
}

// ProviderRepository 获取提供商仓储
func (f *RepositoryFactory) ProviderRepository() repositories.ProviderRepository {
	return NewProviderRepositoryGorm(f.gormDB, f.cache)
}

// ModelRepository 获取模型仓储
func (f *RepositoryFactory) ModelRepository() repositories.ModelRepository {
	return NewModelRepositoryGorm(f.gormDB, f.cache)
}

// ModelPricingRepository 获取模型定价仓储
func (f *RepositoryFactory) ModelPricingRepository() repositories.ModelPricingRepository {
	return NewModelPricingRepositoryGorm(f.gormDB, f.cache)
}

// ProviderModelSupportRepository 获取提供商模型支持仓储
func (f *RepositoryFactory) ProviderModelSupportRepository() repositories.ProviderModelSupportRepository {
	return NewProviderModelSupportRepositoryGorm(f.gormDB, f.cache)
}

// QuotaRepository 获取配额仓储
func (f *RepositoryFactory) QuotaRepository() repositories.QuotaRepository {
	return NewQuotaRepositoryGorm(f.gormDB, f.cache)
}

// QuotaUsageRepository 获取配额使用仓储
func (f *RepositoryFactory) QuotaUsageRepository() repositories.QuotaUsageRepository {
	return NewQuotaUsageRepositoryGorm(f.gormDB, f.cache)
}

// UsageLogRepository 获取使用日志仓储
func (f *RepositoryFactory) UsageLogRepository() repositories.UsageLogRepository {
	return NewUsageLogRepositoryGorm(f.gormDB, f.cache)
}

// BillingRecordRepository 获取计费记录仓储
func (f *RepositoryFactory) BillingRecordRepository() repositories.BillingRecordRepository {
	return NewBillingRecordRepositoryGorm(f.gormDB, f.cache)
}

// ToolRepository 获取工具仓储
func (f *RepositoryFactory) ToolRepository() repositories.ToolRepository {
	return NewToolRepositoryGorm(f.gormDB, f.cache)
}

// GormDB 获取GORM数据库连接
func (f *RepositoryFactory) GormDB() *gorm.DB {
	return f.gormDB
}
