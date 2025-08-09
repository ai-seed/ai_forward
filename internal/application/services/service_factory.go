package services

import (
	"net/http"
	"time"

	"ai-api-gateway/internal/billing/middleware"
	billingService "ai-api-gateway/internal/billing/service"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/domain/services"
	"ai-api-gateway/internal/infrastructure/async"
	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/config"
	"ai-api-gateway/internal/infrastructure/email"
	"ai-api-gateway/internal/infrastructure/logger"
	redisInfra "ai-api-gateway/internal/infrastructure/redis"
	infraRepos "ai-api-gateway/internal/infrastructure/repositories"
	"ai-api-gateway/internal/infrastructure/storage"
	"ai-api-gateway/internal/infrastructure/verification"

	"github.com/sirupsen/logrus"
)

// ServiceFactory 服务工厂
type ServiceFactory struct {
	repoFactory  *infraRepos.RepositoryFactory
	redisFactory *redisInfra.RedisFactory
	config       *config.Config
	logger       logger.Logger
}

// NewServiceFactory 创建服务工厂
func NewServiceFactory(repoFactory *infraRepos.RepositoryFactory, redisFactory *redisInfra.RedisFactory, cfg *config.Config, log logger.Logger) *ServiceFactory {
	return &ServiceFactory{
		repoFactory:  repoFactory,
		redisFactory: redisFactory,
		config:       cfg,
		logger:       log,
	}
}

// UserService 获取用户服务
func (f *ServiceFactory) UserService() UserService {
	var cache *redisInfra.CacheService
	var lockService *redisInfra.DistributedLockService

	if f.redisFactory != nil {
		cache = f.redisFactory.GetCacheService()
		lockService = f.redisFactory.GetLockService()
	}

	return NewUserService(f.repoFactory.UserRepository(), cache, lockService)
}

// APIKeyService 获取API密钥服务
func (f *ServiceFactory) APIKeyService() APIKeyService {
	return NewAPIKeyService(
		f.repoFactory.APIKeyRepository(),
		f.repoFactory.UserRepository(),
		f.repoFactory.UsageLogRepository(),
	)
}

// ProviderService 获取提供商服务
func (f *ServiceFactory) ProviderService() ProviderService {
	return NewProviderService(
		f.repoFactory.ProviderRepository(),
		f.repoFactory.ModelRepository(),
	)
}

// ModelService 获取模型服务
func (f *ServiceFactory) ModelService() ModelService {
	return NewModelService(
		f.repoFactory.ModelRepository(),
		f.repoFactory.ModelPricingRepository(),
		f.repoFactory.ProviderRepository(),
	)
}

// QuotaService 获取配额服务
func (f *ServiceFactory) QuotaService() services.QuotaService {
	// 检查是否启用异步配额处理
	if f.isAsyncQuotaEnabled() && f.redisFactory != nil {
		// 创建异步配额服务
		asyncService, err := f.createAsyncQuotaService()
		if err != nil {
			f.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Failed to create async quota service, falling back to sync")
		} else {
			return asyncService
		}
	}

	// 如果有Redis工厂，创建带缓存的配额服务
	if f.redisFactory != nil {
		return NewQuotaServiceWithCache(
			f.repoFactory.QuotaRepository(),
			f.repoFactory.QuotaUsageRepository(),
			f.repoFactory.UserRepository(),
			f.redisFactory.GetCacheService(),
			f.redisFactory.GetInvalidationService(),
			f.logger,
		)
	}

	// 否则创建普通的配额服务
	return NewQuotaService(
		f.repoFactory.QuotaRepository(),
		f.repoFactory.QuotaUsageRepository(),
		f.repoFactory.UserRepository(),
		f.logger,
	)
}

// BillingService 获取计费服务
func (f *ServiceFactory) BillingService() BillingService {
	return NewBillingService(
		f.repoFactory.BillingRecordRepository(),
		f.repoFactory.UsageLogRepository(),
		f.repoFactory.ModelPricingRepository(),
		f.repoFactory.UserRepository(),
	)
}

// UsageLogService 获取使用日志服务
func (f *ServiceFactory) UsageLogService() UsageLogService {
	return NewUsageLogService(
		f.repoFactory.UsageLogRepository(),
		f.repoFactory.UserRepository(),
		f.repoFactory.APIKeyRepository(),
		f.repoFactory.ProviderRepository(),
		f.repoFactory.ModelRepository(),
	)
}

// JWTService 获取JWT服务
func (f *ServiceFactory) JWTService() JWTService {
	return NewJWTService(&f.config.JWT)
}

// EmailService 获取邮件服务
func (f *ServiceFactory) EmailService() EmailService {
	template := email.NewTemplateService()
	// 创建一个logrus logger实例
	logrusLogger := logrus.New()
	return email.NewBrevoService(template, logrusLogger)
}

// VerificationService 获取验证码服务
func (f *ServiceFactory) VerificationService() VerificationService {
	if f.redisFactory == nil {
		return nil
	}
	redisClient := f.redisFactory.GetClient()
	// 创建一个logrus logger实例
	logrusLogger := logrus.New()
	return verification.NewRedisVerificationService(redisClient.GetClient(), logrusLogger)
}

// AuthService 获取认证服务
func (f *ServiceFactory) AuthService() AuthService {
	return NewAuthService(
		f.repoFactory.UserRepository(),
		f.JWTService(),
		f.EmailService(),
		f.VerificationService(),
	)
}

// OAuthService 获取OAuth服务
func (f *ServiceFactory) OAuthService() OAuthService {
	return NewOAuthService(
		f.repoFactory.UserRepository(),
		f.JWTService(),
		f.config,
	)
}

// ToolService 获取工具服务
func (f *ServiceFactory) ToolService() *ToolService {
	return NewToolService(
		f.repoFactory.ToolRepository(),
		f.repoFactory.APIKeyRepository(),
		f.repoFactory.ModelRepository(),
		f.repoFactory.ModelProviderRepository(),
		f.repoFactory.ModelPricingRepository(),
	)
}

// MidjourneyService 获取Midjourney服务
func (f *ServiceFactory) MidjourneyService() MidjourneyService {
	return NewMidjourneyService(
		f.repoFactory.MidjourneyJobRepository(),
		f.MidjourneyQueueService(),
		f.logger,
	)
}

// MidjourneyQueueService 获取Midjourney队列服务
func (f *ServiceFactory) MidjourneyQueueService() MidjourneyQueueService {
	return NewMidjourneyQueueService(
		f.repoFactory.MidjourneyJobRepository(),
		f.redisFactory.GetCacheService(),
		f.WebhookService(),
		f.ImageGenerationService(),
		f.repoFactory.ProviderModelSupportRepository(),
		f.repoFactory.ProviderRepository(),
		f.BillingService(),
		f.logger,
	)
}

// WebhookService 获取Webhook服务
func (f *ServiceFactory) WebhookService() WebhookService {
	return NewWebhookService(f.logger)
}

// ImageGenerationService 获取图像生成服务
func (f *ServiceFactory) ImageGenerationService() ImageGenerationService {
	// 现在 Midjourney 通过数据库配置提供商，这里只提供模拟服务用于向后兼容
	return NewMockImageGenerationService(f.logger)
}

// UsageLogRepository 获取使用日志仓储
func (f *ServiceFactory) UsageLogRepository() repositories.UsageLogRepository {
	return f.repoFactory.UsageLogRepository()
}

// BillingRecordRepository 获取计费记录仓储
func (f *ServiceFactory) BillingRecordRepository() repositories.BillingRecordRepository {
	return f.repoFactory.BillingRecordRepository()
}

// UserRepository 获取用户仓储
func (f *ServiceFactory) UserRepository() repositories.UserRepository {
	return f.repoFactory.UserRepository()
}

// ModelRepository 获取模型仓储
func (f *ServiceFactory) ModelRepository() repositories.ModelRepository {
	return f.repoFactory.ModelRepository()
}

// ProviderRepository 获取提供商仓储
func (f *ServiceFactory) ProviderRepository() repositories.ProviderRepository {
	return f.repoFactory.ProviderRepository()
}

// ProviderModelSupportRepository 获取提供商模型支持仓储
func (f *ServiceFactory) ProviderModelSupportRepository() repositories.ProviderModelSupportRepository {
	return f.repoFactory.ProviderModelSupportRepository()
}

// FileUploadService 获取文件上传服务
func (f *ServiceFactory) FileUploadService() FileUploadService {
	// 创建S3服务
	s3Service, err := storage.NewS3Service(&f.config.S3, f.logger)
	if err != nil {
		f.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to create S3 service")
		// 返回一个禁用的服务
		s3Service, _ = storage.NewS3Service(&config.S3Config{Enabled: false}, f.logger)
	}

	return NewFileUploadService(s3Service, f.logger)
}

// StabilityService 获取Stability.ai服务
func (f *ServiceFactory) StabilityService() StabilityService {
	// 创建HTTP客户端
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 创建Stability客户端
	stabilityClient := clients.NewStabilityClient(httpClient)

	return NewStabilityService(
		stabilityClient,
		f.repoFactory.ProviderRepository(),
		f.repoFactory.ModelRepository(),
		f.repoFactory.APIKeyRepository(),
		f.repoFactory.UserRepository(),
		f.repoFactory.ProviderModelSupportRepository(),
		f.BillingService(),
		f.UsageLogService(),
		f.logger,
	)
}

// VectorizerService 获取Vectorizer服务
func (f *ServiceFactory) VectorizerService() VectorizerService {
	// 创建HTTP客户端
	httpClient := &http.Client{
		Timeout: 60 * time.Second, // 矢量化可能需要更长时间
	}

	// 创建Vectorizer客户端
	vectorizerClient := clients.NewVectorizerClient(httpClient)

	return NewVectorizerService(
		vectorizerClient,
		f.repoFactory.ProviderRepository(),
		f.repoFactory.ModelRepository(),
		f.repoFactory.APIKeyRepository(),
		f.repoFactory.UserRepository(),
		f.repoFactory.ProviderModelSupportRepository(),
		f.BillingService(),
		f.UsageLogService(),
		f.logger,
	)
}

// AI302Service 获取302.AI服务
func (f *ServiceFactory) AI302Service() AI302Service {
	// 创建HTTP客户端
	httpClient := &http.Client{
		Timeout: 60 * time.Second, // 302.AI图片处理需要8-10秒，设置较长超时
	}

	// 创建302.AI客户端
	ai302Client := clients.NewAI302Client(httpClient)

	// 创建S3服务
	s3Service, err := storage.NewS3Service(&f.config.S3, f.logger)
	if err != nil {
		f.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to create S3 service for AI302Service")
		// 返回一个禁用的服务
		s3Service, _ = storage.NewS3Service(&config.S3Config{Enabled: false}, f.logger)
	}

	return NewAI302Service(
		ai302Client,
		f.repoFactory.ProviderRepository(),
		f.repoFactory.ModelRepository(),
		f.repoFactory.ModelPricingRepository(),
		f.repoFactory.ProviderModelSupportRepository(),
		f.repoFactory.UsageLogRepository(),
		s3Service,
		f.logger,
	)
}

// isAsyncQuotaEnabled 检查是否启用异步配额处理
func (f *ServiceFactory) isAsyncQuotaEnabled() bool {
	// 暂时硬编码返回true来启用异步处理
	// 在实际项目中应该从配置文件读取: viper.GetBool("async_quota.enabled")
	return true
}

// createAsyncQuotaService 创建异步配额服务
func (f *ServiceFactory) createAsyncQuotaService() (services.QuotaService, error) {
	// 创建异步消费者配置
	config := f.getAsyncQuotaConfig()

	// 创建异步配额服务
	return NewAsyncQuotaService(
		f.repoFactory.QuotaRepository(),
		f.repoFactory.QuotaUsageRepository(),
		f.repoFactory.UserRepository(),
		f.redisFactory.GetCacheService(),
		f.redisFactory.GetInvalidationService(),
		config,
		f.logger,
	)
}

// ThinkingService 获取思考服务
func (f *ServiceFactory) ThinkingService() ThinkingService {
	return NewThinkingService(f.logger)
}

// BillingInterceptor 获取计费拦截器
func (f *ServiceFactory) BillingInterceptor() *middleware.BillingInterceptor {
	// 创建计费管理器
	billingManager := billingService.NewBillingManager(
		f.QuotaService(),
		f.repoFactory.UsageLogRepository(),
		f.repoFactory.UserRepository(),
		f.repoFactory.BillingRecordRepository(),
		f.repoFactory.ModelPricingRepository(),
		f.logger,
	)

	// 创建计费拦截器
	return middleware.NewBillingInterceptor(billingManager, f.logger, f.repoFactory.ModelRepository())
}

// getAsyncQuotaConfig 获取异步配额配置
func (f *ServiceFactory) getAsyncQuotaConfig() *async.QuotaConsumerConfig {
	// 暂时使用默认配置
	// 在实际项目中应该从配置文件读取
	return &async.QuotaConsumerConfig{
		WorkerCount:   3,                      // 3个工作协程
		ChannelSize:   1000,                   // 1000个事件缓冲
		BatchSize:     10,                     // 每批处理10个事件
		FlushInterval: 5 * time.Second,        // 5秒强制刷新
		RetryAttempts: 3,                      // 重试3次
		RetryDelay:    100 * time.Millisecond, // 100ms重试延迟
	}
}
