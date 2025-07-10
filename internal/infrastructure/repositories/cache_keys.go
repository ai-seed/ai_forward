package repositories

import "fmt"

// 缓存key常量定义
const (
	// 用户相关缓存key
	CacheKeyUserByID       = "user:id:%d"
	CacheKeyUserByUsername = "user:username:%s"
	CacheKeyUserByEmail    = "user:email:%s"
	CacheKeyActiveUsers    = "users:active"

	// API密钥相关缓存key
	CacheKeyAPIKeyByKey    = "apikey:key:%s"
	CacheKeyAPIKeyByID     = "apikey:id:%d"
	CacheKeyAPIKeysByUser  = "apikeys:user:%d"
	CacheKeyActiveAPIKeys  = "apikeys:active:user:%d"

	// 模型相关缓存key
	CacheKeyModelByID         = "model:id:%d"
	CacheKeyModelBySlug       = "model:slug:%s"
	CacheKeyActiveModels      = "models:active"
	CacheKeyAvailableModels   = "models:available"
	CacheKeyModelsByType      = "models:type:%s"
	CacheKeyModelsByProvider  = "models:provider:%d"

	// 提供商相关缓存key
	CacheKeyProviderByID              = "provider:id:%d"
	CacheKeyProviderBySlug            = "provider:slug:%s"
	CacheKeyActiveProviders           = "providers:active"
	CacheKeyAvailableProviders        = "providers:available"
	CacheKeyProvidersByPriority       = "providers:priority"
	CacheKeyProvidersNeedingHealthCheck = "providers:health_check"

	// 模型定价相关缓存key
	CacheKeyModelPricingByID    = "model_pricing:id:%d"
	CacheKeyModelPricingByModel = "model_pricing:model:%d"

	// 提供商模型支持相关缓存key
	CacheKeyProviderModelSupportByID           = "provider_model_support:id:%d"
	CacheKeyProviderModelSupportByProviderModel = "provider_model_support:provider:%d:model:%s"
	CacheKeySupportingProviders                = "supporting_providers:model:%s"
	CacheKeyProviderSupportedModels            = "provider_supported_models:provider:%d"

	// 配额相关缓存key
	CacheKeyQuotaByID              = "quota:id:%d"
	CacheKeyQuotasByAPIKey         = "quotas:apikey:%d"
	CacheKeyActiveQuotas           = "quotas:active:apikey:%d"
	CacheKeyQuotaByAPIKeyAndType   = "quota:apikey:%d:type:%s:period:%s"

	// 配额使用相关缓存key
	CacheKeyQuotaUsageByID          = "quota_usage:id:%d"
	CacheKeyQuotaUsageByQuotaPeriod = "quota_usage:apikey:%d:quota:%d:period:%s"
	CacheKeyCurrentQuotaUsage       = "quota_usage:current:apikey:%d:quota:%d"

	// 工具相关缓存key
	CacheKeyToolByID                = "tool:id:%s"
	CacheKeyActiveTools             = "tools:active"
	CacheKeyUserToolInstanceByID    = "user_tool_instance:id:%s"
	CacheKeyUserToolInstances       = "user_tool_instances:user:%d"
	CacheKeyUserToolInstancesByTool = "user_tool_instances:user:%d:tool:%s"

	// 使用日志相关缓存key（通常不缓存，但可以缓存统计数据）
	CacheKeyUsageStatsByUser  = "usage_stats:user:%d:period:%s"
	CacheKeyUsageStatsByModel = "usage_stats:model:%d:period:%s"

	// 计费记录相关缓存key（通常不缓存，但可以缓存统计数据）
	CacheKeyBillingStatsByUser = "billing_stats:user:%d:period:%s"
	CacheKeyPendingBillingRecords = "billing_records:pending"
)

// 缓存key生成函数
func GetUserCacheKey(userID int64) string {
	return fmt.Sprintf(CacheKeyUserByID, userID)
}

func GetUserByUsernameCacheKey(username string) string {
	return fmt.Sprintf(CacheKeyUserByUsername, username)
}

func GetUserByEmailCacheKey(email string) string {
	return fmt.Sprintf(CacheKeyUserByEmail, email)
}

func GetAPIKeyCacheKey(key string) string {
	return fmt.Sprintf(CacheKeyAPIKeyByKey, key)
}

func GetAPIKeyByIDCacheKey(id int64) string {
	return fmt.Sprintf(CacheKeyAPIKeyByID, id)
}

func GetAPIKeysByUserCacheKey(userID int64) string {
	return fmt.Sprintf(CacheKeyAPIKeysByUser, userID)
}

func GetActiveAPIKeysCacheKey(userID int64) string {
	return fmt.Sprintf(CacheKeyActiveAPIKeys, userID)
}

func GetModelCacheKey(modelID int64) string {
	return fmt.Sprintf(CacheKeyModelByID, modelID)
}

func GetModelBySlugCacheKey(slug string) string {
	return fmt.Sprintf(CacheKeyModelBySlug, slug)
}

func GetModelsByTypeCacheKey(modelType string) string {
	return fmt.Sprintf(CacheKeyModelsByType, modelType)
}

func GetModelsByProviderCacheKey(providerID int64) string {
	return fmt.Sprintf(CacheKeyModelsByProvider, providerID)
}

func GetProviderCacheKey(providerID int64) string {
	return fmt.Sprintf(CacheKeyProviderByID, providerID)
}

func GetProviderBySlugCacheKey(slug string) string {
	return fmt.Sprintf(CacheKeyProviderBySlug, slug)
}

func GetModelPricingCacheKey(pricingID int64) string {
	return fmt.Sprintf(CacheKeyModelPricingByID, pricingID)
}

func GetModelPricingByModelCacheKey(modelID int64) string {
	return fmt.Sprintf(CacheKeyModelPricingByModel, modelID)
}

func GetProviderModelSupportCacheKey(supportID int64) string {
	return fmt.Sprintf(CacheKeyProviderModelSupportByID, supportID)
}

func GetProviderModelSupportByProviderModelCacheKey(providerID int64, modelSlug string) string {
	return fmt.Sprintf(CacheKeyProviderModelSupportByProviderModel, providerID, modelSlug)
}

func GetSupportingProvidersCacheKey(modelSlug string) string {
	return fmt.Sprintf(CacheKeySupportingProviders, modelSlug)
}

func GetProviderSupportedModelsCacheKey(providerID int64) string {
	return fmt.Sprintf(CacheKeyProviderSupportedModels, providerID)
}

func GetQuotaCacheKey(quotaID int64) string {
	return fmt.Sprintf(CacheKeyQuotaByID, quotaID)
}

func GetQuotasByAPIKeyCacheKey(apiKeyID int64) string {
	return fmt.Sprintf(CacheKeyQuotasByAPIKey, apiKeyID)
}

func GetActiveQuotasCacheKey(apiKeyID int64) string {
	return fmt.Sprintf(CacheKeyActiveQuotas, apiKeyID)
}

func GetQuotaByAPIKeyAndTypeCacheKey(apiKeyID int64, quotaType, period string) string {
	return fmt.Sprintf(CacheKeyQuotaByAPIKeyAndType, apiKeyID, quotaType, period)
}

func GetQuotaUsageCacheKey(usageID int64) string {
	return fmt.Sprintf(CacheKeyQuotaUsageByID, usageID)
}

func GetQuotaUsageByQuotaPeriodCacheKey(apiKeyID, quotaID int64, period string) string {
	return fmt.Sprintf(CacheKeyQuotaUsageByQuotaPeriod, apiKeyID, quotaID, period)
}

func GetCurrentQuotaUsageCacheKey(apiKeyID, quotaID int64) string {
	return fmt.Sprintf(CacheKeyCurrentQuotaUsage, apiKeyID, quotaID)
}

func GetToolCacheKey(toolID string) string {
	return fmt.Sprintf(CacheKeyToolByID, toolID)
}

func GetUserToolInstanceCacheKey(instanceID string) string {
	return fmt.Sprintf(CacheKeyUserToolInstanceByID, instanceID)
}

func GetUserToolInstancesCacheKey(userID int64) string {
	return fmt.Sprintf(CacheKeyUserToolInstances, userID)
}

func GetUserToolInstancesByToolCacheKey(userID int64, toolID string) string {
	return fmt.Sprintf(CacheKeyUserToolInstancesByTool, userID, toolID)
}
