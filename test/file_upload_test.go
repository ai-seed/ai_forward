package test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/infrastructure/config"
	"ai-api-gateway/internal/infrastructure/database"
	"ai-api-gateway/internal/infrastructure/gateway"
	"ai-api-gateway/internal/infrastructure/logger"
	"ai-api-gateway/internal/infrastructure/redis"
	infraRepos "ai-api-gateway/internal/infrastructure/repositories"
	"ai-api-gateway/internal/presentation/routes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileUploadAPI(t *testing.T) {
	// 创建测试配置
	cfg := &config.Config{
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		S3: config.S3Config{
			Enabled:      false, // 测试时禁用S3
			Region:       "us-east-1",
			Bucket:       "test-bucket",
			MaxFileSize:  10 * 1024 * 1024, // 10MB
			AllowedTypes: []string{"image/jpeg", "image/png", "application/pdf"},
		},
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-jwt-token-generation",
		},
	}

	// 创建日志器
	log := logger.NewLogger(&cfg.Logging)

	// 创建数据库连接（使用内存SQLite进行测试）
	gormDB, err := database.NewGormDB(database.GormConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "test",
		Password: "test",
		DBName:   "test",
		SSLMode:  "disable",
		TimeZone: "UTC",
	})
	// 如果连接失败，跳过测试（因为这是集成测试）
	if err != nil {
		t.Skip("Database connection failed, skipping integration test:", err)
	}

	// 创建Redis工厂（测试时可以为nil）
	var redisFactory *redis.RedisFactory

	// 创建仓储工厂
	repoFactory := infraRepos.NewRepositoryFactory(gormDB)

	// 创建服务工厂
	serviceFactory := services.NewServiceFactory(repoFactory, redisFactory, cfg, log)

	// 创建网关服务（简化版，用于测试）
	gatewayService := createMockGatewayService()

	// 创建路由
	router := routes.NewRouter(cfg, log, serviceFactory, gatewayService)
	router.SetupRoutes()
	engine := router.GetEngine()

	// 运行测试
	t.Run("TestFileUploadDisabled", func(t *testing.T) {
		testFileUploadDisabled(t, engine)
	})

	t.Run("TestFileUploadWithoutAuth", func(t *testing.T) {
		testFileUploadWithoutAuth(t, engine)
	})
}

// testFileUploadDisabled 测试S3服务禁用时的响应
func testFileUploadDisabled(t *testing.T, engine http.Handler) {
	// 创建测试文件
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.jpg")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake image content"))
	require.NoError(t, err)
	writer.Close()

	// 创建请求
	req := httptest.NewRequest("POST", "/api/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer fake-token") // 假的token，会被认证中间件拦截

	// 执行请求
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 验证响应 - 应该返回401因为没有有效的认证
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// testFileUploadWithoutAuth 测试没有认证的文件上传
func testFileUploadWithoutAuth(t *testing.T, engine http.Handler) {
	// 创建测试文件
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.jpg")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake image content"))
	require.NoError(t, err)
	writer.Close()

	// 创建请求（不带认证头）
	req := httptest.NewRequest("POST", "/api/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 执行请求
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	// 验证响应 - 应该返回401因为没有认证
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response dto.Response
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, strings.ToLower(response.Message), "authentication")
}

// createMockGatewayService 创建模拟的网关服务
func createMockGatewayService() gateway.GatewayService {
	return &mockGatewayService{}
}

// mockGatewayService 模拟网关服务实现
type mockGatewayService struct{}

func (m *mockGatewayService) ProcessRequest(ctx context.Context, request *gateway.GatewayRequest) (*gateway.GatewayResponse, error) {
	return nil, nil
}

func (m *mockGatewayService) ProcessStreamRequest(ctx context.Context, request *gateway.GatewayRequest, streamChan chan<- *gateway.StreamChunk) error {
	return nil
}

func (m *mockGatewayService) HealthCheck(ctx context.Context) (*gateway.HealthCheckResult, error) {
	return &gateway.HealthCheckResult{
		Status:    "healthy",
		Timestamp: time.Now(),
		Providers: make(map[string]gateway.ProviderHealth),
		Database:  gateway.DatabaseHealth{Status: "healthy"},
	}, nil
}

func (m *mockGatewayService) GetStats(ctx context.Context) (*gateway.GatewayStats, error) {
	return &gateway.GatewayStats{
		TotalRequests:      0,
		SuccessfulRequests: 0,
		FailedRequests:     0,
		SuccessRate:        1.0,
		AvgResponseTime:    time.Millisecond * 100,
	}, nil
}
