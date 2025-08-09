package routes

import (
	"time"

	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/config"
	"ai-api-gateway/internal/infrastructure/functioncall"
	"ai-api-gateway/internal/infrastructure/gateway"
	"ai-api-gateway/internal/infrastructure/logger"
	"ai-api-gateway/internal/presentation/handlers"
	"ai-api-gateway/internal/presentation/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "ai-api-gateway/docs" // 导入swagger文档
)

// Router 路由器
type Router struct {
	engine         *gin.Engine
	config         *config.Config
	logger         logger.Logger
	serviceFactory *services.ServiceFactory
	gatewayService gateway.GatewayService
}

// NewRouter 创建路由器
func NewRouter(
	config *config.Config,
	logger logger.Logger,
	serviceFactory *services.ServiceFactory,
	gatewayService gateway.GatewayService,
) *Router {
	// 设置Gin模式
	if config.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	return &Router{
		engine:         engine,
		config:         config,
		logger:         logger,
		serviceFactory: serviceFactory,
		gatewayService: gatewayService,
	}
}

// SetupRoutes 设置路由
func (r *Router) SetupRoutes() {
	// 创建中间件
	authMiddleware := middleware.NewAuthMiddleware(
		r.serviceFactory.APIKeyService(),
		r.serviceFactory.JWTService(),
		r.serviceFactory.UserService(),
		r.serviceFactory.UserRepository(),
		r.logger,
	)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(&r.config.RateLimit, r.logger)
	quotaMiddleware := middleware.NewQuotaMiddleware(r.serviceFactory.QuotaService(), r.logger)
	
	// 创建计费中间件
	billingInterceptor := r.serviceFactory.BillingInterceptor()

	// 全局中间件
	r.engine.Use(middleware.RecoveryMiddleware(r.logger))
	r.engine.Use(middleware.LoggingMiddleware(r.logger))
	r.engine.Use(middleware.CORSMiddleware())
	r.engine.Use(middleware.SecurityMiddleware())
	r.engine.Use(middleware.RequestIDMiddleware())
	r.engine.Use(middleware.TimeoutMiddleware(30 * time.Second))

	// 创建处理器
	// 创建 Function Call 相关服务（始终创建，通过 web_search 参数控制使用）
	searchConfig := &functioncall.SearchConfig{
		Service:        r.config.FunctionCall.SearchService.Service,
		MaxResults:     r.config.FunctionCall.SearchService.MaxResults,
		CrawlResults:   r.config.FunctionCall.SearchService.CrawlResults,
		CrawlContent:   r.config.FunctionCall.SearchService.CrawlContent,
		Search1APIKey:  r.config.FunctionCall.SearchService.Search1APIKey,
		GoogleCX:       r.config.FunctionCall.SearchService.GoogleCX,
		GoogleKey:      r.config.FunctionCall.SearchService.GoogleKey,
		BingKey:        r.config.FunctionCall.SearchService.BingKey,
		SerpAPIKey:     r.config.FunctionCall.SearchService.SerpAPIKey,
		SerperKey:      r.config.FunctionCall.SearchService.SerperKey,
		SearXNGBaseURL: r.config.FunctionCall.SearchService.SearXNGBaseURL,
	}
	searchService := functioncall.NewSearchService(searchConfig, r.logger)
	functionCallHandler := functioncall.NewFunctionCallHandler(searchService, r.logger)

	// 创建HTTP客户端
	httpClient := clients.NewHTTPClient(30 * time.Second)

	// 创建AI提供商客户端
	aiClient := clients.NewAIProviderClient(httpClient)

	aiHandler := handlers.NewAIHandler(
		r.gatewayService,
		r.serviceFactory.ModelService(),
		r.serviceFactory.UsageLogService(),
		r.logger,
		r.config,
		functionCallHandler,
		r.serviceFactory.ProviderModelSupportRepository(),
		httpClient,
		aiClient,
		r.serviceFactory.ThinkingService(),
	)
	userHandler := handlers.NewUserHandler(r.serviceFactory.UserService(), r.logger)
	apiKeyHandler := handlers.NewAPIKeyHandler(
		r.serviceFactory.APIKeyService(),
		r.serviceFactory.UsageLogRepository(),
		r.serviceFactory.BillingRecordRepository(),
		r.serviceFactory.ModelRepository(),
		r.logger,
	)
	healthHandler := handlers.NewHealthHandler(r.gatewayService, r.logger)
	authHandler := handlers.NewAuthHandler(r.serviceFactory.AuthService(), r.logger)
	oauthHandler := handlers.NewOAuthHandler(r.serviceFactory.OAuthService(), r.logger, r.config)
	toolHandler := handlers.NewToolHandler(r.serviceFactory.ToolService(), r.logger)
	quotaHandler := handlers.NewQuotaHandler(r.serviceFactory.QuotaService(), r.logger)
	midjourneyHandler := handlers.NewMidjourneyHandler(
		r.serviceFactory.MidjourneyService(),
		r.serviceFactory.BillingService(),
		r.serviceFactory.ModelRepository(),
		r.serviceFactory.UsageLogRepository(),
		r.serviceFactory.UserService(),
		r.serviceFactory.ProviderRepository(),
		r.serviceFactory.ProviderModelSupportRepository(),
		r.logger,
	)
	fileUploadHandler := handlers.NewFileUploadHandler(
		r.serviceFactory.FileUploadService(),
		&r.config.S3,
		r.logger,
	)
	stabilityHandler := handlers.NewStabilityHandler(r.serviceFactory.StabilityService(), r.logger)
	vectorizerHandler := handlers.NewVectorizerHandler(r.serviceFactory.VectorizerService(), r.logger)
	ai302Handler := handlers.NewAI302Handler(r.serviceFactory.AI302Service(), r.logger)

	// 健康检查路由（无需认证）
	r.engine.GET("/health", healthHandler.HealthCheck)

	// 认证路由（无需认证）
	auth := r.engine.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register)
		auth.POST("/refresh", authHandler.RefreshToken)

		// 验证码相关路由（无需认证）
		auth.POST("/send-verification-code", authHandler.SendVerificationCode)
		auth.POST("/verify-code", authHandler.VerifyCode)
		auth.POST("/register-with-code", authHandler.RegisterWithCode)
		auth.POST("/reset-password", authHandler.ResetPassword)

		// OAuth路由（无需认证）
		oauth := auth.Group("/oauth")
		{
			oauth.GET("/:provider/url", oauthHandler.GetAuthURL)
			oauth.POST("/:provider/callback", oauthHandler.HandleCallback)
			oauth.GET("/:provider/redirect", oauthHandler.GetAuthURLFromQuery)
			oauth.GET("/:provider/callback", oauthHandler.HandleCallbackFromQuery)
		}

		// 需要认证的认证路由
		authProtected := auth.Group("/")
		authProtected.Use(authMiddleware.Authenticate())
		{
			authProtected.GET("/profile", authHandler.GetProfile)
			authProtected.POST("/change-password", authHandler.ChangePassword)
			authProtected.POST("/recharge", authHandler.Recharge)
		}
	}

	// Swagger文档路由（无需认证）
	swaggerGroup := r.engine.Group("/swagger")
	swaggerGroup.Use(func(c *gin.Context) {
		// 设置 CSP 头部以允许 Swagger UI 正常工作
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'")
		c.Next()
	})
	swaggerGroup.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// OpenAI兼容的API路由
	v1 := r.engine.Group("/v1")
	v1.Use(rateLimitMiddleware.IPRateLimit(100)) // IP级别限流
	{
		// AI请求路由（需要认证、计费预检查、配额检查）
		aiRoutes := v1.Group("/")
		aiRoutes.Use(authMiddleware.Authenticate())
		aiRoutes.Use(rateLimitMiddleware.RateLimit())
		aiRoutes.Use(billingInterceptor.PreRequestMiddleware()) // 计费预检查
		aiRoutes.Use(quotaMiddleware.CheckQuota())
		aiRoutes.Use(quotaMiddleware.ConsumeQuota()) // 在请求完成后消费配额
		aiRoutes.Use(billingInterceptor.PostRequestMiddleware()) // 计费处理
		{
			aiRoutes.POST("/chat/completions", aiHandler.ChatCompletions)
			aiRoutes.POST("/completions", aiHandler.Completions)
			aiRoutes.POST("/messages", aiHandler.AnthropicMessages) // Anthropic Messages API
		}

		// 信息查询路由（需要认证但不消费配额）
		infoRoutes := v1.Group("/")
		infoRoutes.Use(authMiddleware.Authenticate())
		infoRoutes.Use(rateLimitMiddleware.RateLimit())
		{
			infoRoutes.GET("/models", aiHandler.Models)
			infoRoutes.GET("/usage", aiHandler.Usage)
		}
	}

	// 文件管理API路由
	files := r.engine.Group("/api/files")
	files.Use(rateLimitMiddleware.RateLimit())
	files.Use(authMiddleware.Authenticate()) // 需要JWT认证
	{
		files.POST("/upload", fileUploadHandler.UploadFile)
		files.DELETE("/delete", fileUploadHandler.DeleteFile)
	}

	// 管理API路由
	admin := r.engine.Group("/admin")
	admin.Use(rateLimitMiddleware.CustomRateLimit(200)) // 管理API更高的限流
	admin.Use(authMiddleware.Authenticate())            // 需要JWT认证
	{
		// 用户管理路由
		users := admin.Group("/users")
		{
			users.POST("/", userHandler.CreateUser)
			users.GET("/", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
			users.POST("/:id/balance", userHandler.UpdateBalance)
			// 用户的API密钥路由
			users.GET("/:id/api-keys", apiKeyHandler.GetUserAPIKeys)
		}

		// API密钥管理路由
		apiKeys := admin.Group("/api-keys")
		{
			apiKeys.POST("/", apiKeyHandler.CreateAPIKey)
			apiKeys.GET("/", apiKeyHandler.ListAPIKeys)
			apiKeys.GET("/:id/usage-logs", apiKeyHandler.GetAPIKeyUsageLogs)
			apiKeys.GET("/:id/billing-records", apiKeyHandler.GetAPIKeyBillingRecords)
			apiKeys.POST("/:id/revoke", apiKeyHandler.RevokeAPIKey)
			apiKeys.GET("/:id", apiKeyHandler.GetAPIKey)
			apiKeys.PUT("/:id", apiKeyHandler.UpdateAPIKey)
			apiKeys.DELETE("/:id", apiKeyHandler.DeleteAPIKey)

			// API密钥配额管理路由
			apiKeys.GET("/:id/quotas", quotaHandler.GetAPIKeyQuotas)
			apiKeys.POST("/:id/quotas", quotaHandler.CreateAPIKeyQuota)
			apiKeys.GET("/:id/quota-status", quotaHandler.GetQuotaStatus)
		}

		// 配额管理路由
		quotas := admin.Group("/quotas")
		{
			quotas.PUT("/:quota_id", quotaHandler.UpdateQuota)
			quotas.DELETE("/:quota_id", quotaHandler.DeleteQuota)
		}

		// 工具管理路由
		tools := admin.Group("/tools")
		{
			// 用户工具实例路由
			tools.GET("/", toolHandler.GetUserToolInstances)
			tools.POST("/", toolHandler.CreateUserToolInstance)
			tools.GET("/:id", toolHandler.GetUserToolInstance)
			tools.PUT("/:id", toolHandler.UpdateUserToolInstance)
			tools.DELETE("/:id", toolHandler.DeleteUserToolInstance)
			tools.POST("/:id/usage", toolHandler.IncrementUsage)

			// 工具相关资源路由
			tools.GET("/api-keys", toolHandler.GetUserAPIKeys)
		}

	}

	// 公开工具路由（无需认证）
	publicTools := r.engine.Group("/tools")
	{
		publicTools.GET("/types", toolHandler.GetTools)
		publicTools.GET("/models", toolHandler.GetModels)
		publicTools.GET("/public", toolHandler.GetPublicTools)
		publicTools.GET("/share/:token", toolHandler.GetSharedToolInstance)
		publicTools.GET("/by-code/:code", toolHandler.GetToolInstanceByCode) // 通过code获取工具信息
	}

	// Midjourney兼容的API路由（302AI格式）
	mj := r.engine.Group("/mj")
	mj.Use(rateLimitMiddleware.IPRateLimit(50)) // IP级别限流
	{
		// Midjourney提交路由（需要认证、计费预检查、配额检查）
		mjSubmit := mj.Group("/submit")
		mjSubmit.Use(authMiddleware.Authenticate())
		mjSubmit.Use(rateLimitMiddleware.RateLimit())
		mjSubmit.Use(billingInterceptor.PreRequestMiddleware()) // 计费预检查
		mjSubmit.Use(quotaMiddleware.CheckQuota())
		mjSubmit.Use(quotaMiddleware.ConsumeQuota()) // 在请求完成后消费配额
		mjSubmit.Use(billingInterceptor.PostRequestMiddleware()) // 计费处理
		{
			mjSubmit.POST("/imagine", midjourneyHandler.Imagine)
			mjSubmit.POST("/action", midjourneyHandler.Action)
			mjSubmit.POST("/blend", midjourneyHandler.Blend)
			mjSubmit.POST("/describe", midjourneyHandler.Describe)
			mjSubmit.POST("/modal", midjourneyHandler.Modal)
			mjSubmit.POST("/cancel", midjourneyHandler.Cancel)
		}

		// Midjourney任务查询路由（需要认证但不消费配额）
		mjTask := mj.Group("/task")
		mjTask.Use(authMiddleware.Authenticate())
		mjTask.Use(rateLimitMiddleware.RateLimit())
		{
			mjTask.GET("/:id/fetch", midjourneyHandler.Fetch)
		}
	}

	// Stability.ai兼容的API路由（302AI格式）
	sd := r.engine.Group("/sd")
	sd.Use(rateLimitMiddleware.IPRateLimit(50)) // IP级别限流
	{
		// Stability.ai图像生成路由（需要认证、计费预检查、配额检查）
		sdV1 := sd.Group("/v1/generation")
		sdV1.Use(authMiddleware.Authenticate())
		sdV1.Use(rateLimitMiddleware.RateLimit())
		sdV1.Use(billingInterceptor.PreRequestMiddleware()) // 计费预检查
		sdV1.Use(quotaMiddleware.CheckQuota())
		sdV1.Use(quotaMiddleware.ConsumeQuota()) // 在请求完成后消费配额
		sdV1.Use(billingInterceptor.PostRequestMiddleware()) // 计费处理
		{
			// V1 Text-to-Image (原有接口)
			sdV1.POST("/stable-diffusion-xl-1024-v1-0/text-to-image", stabilityHandler.TextToImage)
		}

		// Stability.ai V2 Beta API路由
		sdV2Beta := sd.Group("/v2beta/stable-image")
		sdV2Beta.Use(authMiddleware.Authenticate())
		sdV2Beta.Use(rateLimitMiddleware.RateLimit())
		sdV2Beta.Use(billingInterceptor.PreRequestMiddleware()) // 计费预检查
		sdV2Beta.Use(quotaMiddleware.CheckQuota())
		sdV2Beta.Use(quotaMiddleware.ConsumeQuota())
		sdV2Beta.Use(billingInterceptor.PostRequestMiddleware()) // 计费处理
		{
			// 图片生成接口
			generate := sdV2Beta.Group("/generate")
			{
				generate.POST("/sd", stabilityHandler.GenerateSD2)
				generate.POST("/sd3", stabilityHandler.GenerateSD3)
				generate.POST("/ultra", stabilityHandler.GenerateSD3Ultra)
				generate.POST("/sd3-large", stabilityHandler.GenerateSD35Large)
				generate.POST("/sd3-medium", stabilityHandler.GenerateSD35Medium)
			}

			// 图生图接口
			control := sdV2Beta.Group("/control")
			{
				control.POST("/sd3", stabilityHandler.ImageToImageSD3)
				control.POST("/sd3-large", stabilityHandler.ImageToImageSD35Large)
				control.POST("/sd3-medium", stabilityHandler.ImageToImageSD35Medium)
				control.POST("/sketch", stabilityHandler.Sketch)
				control.POST("/structure", stabilityHandler.Structure)
				control.POST("/style", stabilityHandler.Style)
			}

			// 图片放大接口
			upscale := sdV2Beta.Group("/upscale")
			{
				upscale.POST("/fast", stabilityHandler.FastUpscale)
				upscale.POST("/creative", stabilityHandler.CreativeUpscale)
				upscale.POST("/conservative", stabilityHandler.ConservativeUpscale)
				upscale.GET("/creative/result/:id", stabilityHandler.FetchCreativeUpscale)
			}

			// 图片编辑接口
			edit := sdV2Beta.Group("/edit")
			{
				edit.POST("/erase", stabilityHandler.Erase)
				edit.POST("/inpaint", stabilityHandler.Inpaint)
				edit.POST("/outpaint", stabilityHandler.Outpaint)
				edit.POST("/search-and-replace", stabilityHandler.SearchAndReplace)
				edit.POST("/search-and-recolor", stabilityHandler.SearchAndRecolor)
				edit.POST("/remove-background", stabilityHandler.RemoveBackground)
				edit.POST("/style-transfer", stabilityHandler.StyleTransfer)
				edit.POST("/replace-background", stabilityHandler.ReplaceBackground)
			}
		}
	}

	// 302.AI API路由
	ai302 := r.engine.Group("/ai")
	ai302.Use(rateLimitMiddleware.IPRateLimit(50)) // IP级别限流
	{
		// 图片处理路由（需要认证、计费预检查、配额检查）
		ai302.Use(authMiddleware.Authenticate())
		ai302.Use(rateLimitMiddleware.RateLimit())
		ai302.Use(billingInterceptor.PreRequestMiddleware()) // 计费预检查
		ai302.Use(quotaMiddleware.CheckQuota())
		ai302.Use(quotaMiddleware.ConsumeQuota()) // 在请求完成后消费配额
		ai302.Use(billingInterceptor.PostRequestMiddleware()) // 计费处理
		{
			ai302.POST("/upscale", ai302Handler.Upscale)
		}
	}

	// Vectorizer API路由
	vectorizer := r.engine.Group("/vectorizer")
	vectorizer.Use(authMiddleware.Authenticate())
	vectorizer.Use(rateLimitMiddleware.RateLimit())
	vectorizer.Use(billingInterceptor.PreRequestMiddleware()) // 计费预检查
	vectorizer.Use(quotaMiddleware.CheckQuota())
	vectorizer.Use(quotaMiddleware.ConsumeQuota())
	vectorizer.Use(billingInterceptor.PostRequestMiddleware()) // 计费处理
	{
		vectorizerV1 := vectorizer.Group("/api/v1")
		{
			vectorizerV1.POST("/vectorize", vectorizerHandler.Vectorize)
		}
	}

	// 404处理
	r.engine.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Endpoint not found",
			},
			"timestamp": time.Now(),
		})
	})

	// 405处理
	r.engine.NoMethod(func(c *gin.Context) {
		c.JSON(405, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "METHOD_NOT_ALLOWED",
				"message": "Method not allowed",
			},
			"timestamp": time.Now(),
		})
	})
}

// GetEngine 获取Gin引擎
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// Start 启动服务器
func (r *Router) Start() error {
	address := r.config.Server.GetAddress()
	r.logger.WithField("address", address).Info("Starting HTTP server")

	return r.engine.Run(address)
}
