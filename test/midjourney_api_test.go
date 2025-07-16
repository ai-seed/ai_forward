package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/infrastructure/config"
	"ai-api-gateway/internal/infrastructure/database"
	"ai-api-gateway/internal/infrastructure/logger"
	"ai-api-gateway/internal/infrastructure/redis"
	infraRepos "ai-api-gateway/internal/infrastructure/repositories"
	"ai-api-gateway/internal/presentation/handlers"
	"ai-api-gateway/internal/presentation/routes"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMidjourneyAPI 测试Midjourney API
func TestMidjourneyAPI(t *testing.T) {
	// 设置测试环境
	gin.SetMode(gin.TestMode)
	
	// 创建测试配置
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    ":memory:",
		},
		Redis: config.RedisConfig{
			Enabled: false, // 测试时禁用Redis
		},
	}
	
	// 创建日志器
	log := logger.NewLogger()
	
	// 创建数据库连接
	gormDB, err := database.NewGormDB(cfg, log)
	require.NoError(t, err)
	
	// 创建Redis工厂（测试时可以为nil）
	var redisFactory *redis.RedisFactory
	
	// 创建仓储工厂
	repoFactory := infraRepos.NewRepositoryFactory(gormDB, nil)
	
	// 创建服务工厂
	serviceFactory := services.NewServiceFactory(repoFactory, redisFactory, cfg, log)
	
	// 创建路由
	router := routes.NewRouter(serviceFactory, log)
	engine := router.SetupRoutes()
	
	// 运行测试
	t.Run("TestImagineEndpoint", func(t *testing.T) {
		testImagineEndpoint(t, engine)
	})
	
	t.Run("TestFetchEndpoint", func(t *testing.T) {
		testFetchEndpoint(t, engine)
	})
	
	t.Run("TestActionEndpoint", func(t *testing.T) {
		testActionEndpoint(t, engine)
	})
}

// testImagineEndpoint 测试图像生成端点
func testImagineEndpoint(t *testing.T, engine *gin.Engine) {
	// 准备请求数据
	requestData := map[string]interface{}{
		"prompt":  "A beautiful cat sitting in a garden",
		"botType": "MID_JOURNEY",
		"state":   "test",
	}
	
	jsonData, err := json.Marshal(requestData)
	require.NoError(t, err)
	
	// 创建请求
	req, err := http.NewRequest("POST", "/mj/submit/imagine", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("mj-api-secret", "test-api-key")
	
	// 执行请求
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response handlers.MJResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, 1, response.Code)
	assert.Equal(t, "提交成功", response.Description)
	assert.NotNil(t, response.Result)
	
	// 验证返回的任务ID
	taskID, ok := response.Result.(string)
	assert.True(t, ok)
	assert.NotEmpty(t, taskID)
	
	t.Logf("Created task ID: %s", taskID)
}

// testFetchEndpoint 测试获取任务端点
func testFetchEndpoint(t *testing.T, engine *gin.Engine) {
	// 首先创建一个任务
	taskID := createTestTask(t, engine)
	
	// 等待一段时间让任务处理
	time.Sleep(100 * time.Millisecond)
	
	// 获取任务状态
	req, err := http.NewRequest("GET", "/mj/task/"+taskID+"/fetch", nil)
	require.NoError(t, err)
	
	req.Header.Set("mj-api-secret", "test-api-key")
	
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response handlers.MJTaskResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, taskID, response.ID)
	assert.Equal(t, "imagine", response.Action)
	assert.NotEmpty(t, response.Status)
	
	t.Logf("Task status: %s, Progress: %s", response.Status, response.Progress)
}

// testActionEndpoint 测试操作端点
func testActionEndpoint(t *testing.T, engine *gin.Engine) {
	// 首先创建一个任务
	parentTaskID := createTestTask(t, engine)
	
	// 准备操作请求数据
	requestData := map[string]interface{}{
		"taskId":   parentTaskID,
		"customId": "upsample1",
		"state":    "test",
	}
	
	jsonData, err := json.Marshal(requestData)
	require.NoError(t, err)
	
	// 创建请求
	req, err := http.NewRequest("POST", "/mj/submit/action", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("mj-api-secret", "test-api-key")
	
	// 执行请求
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response handlers.MJResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, 1, response.Code)
	assert.Equal(t, "提交成功", response.Description)
	assert.NotNil(t, response.Result)
	
	// 验证返回的任务ID
	taskID, ok := response.Result.(string)
	assert.True(t, ok)
	assert.NotEmpty(t, taskID)
	assert.NotEqual(t, parentTaskID, taskID) // 应该是新的任务ID
	
	t.Logf("Created action task ID: %s for parent: %s", taskID, parentTaskID)
}

// createTestTask 创建测试任务
func createTestTask(t *testing.T, engine *gin.Engine) string {
	requestData := map[string]interface{}{
		"prompt":  "Test prompt for task creation",
		"botType": "MID_JOURNEY",
	}
	
	jsonData, err := json.Marshal(requestData)
	require.NoError(t, err)
	
	req, err := http.NewRequest("POST", "/mj/submit/imagine", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("mj-api-secret", "test-api-key")
	
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	
	var response handlers.MJResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	taskID, ok := response.Result.(string)
	require.True(t, ok)
	require.NotEmpty(t, taskID)
	
	return taskID
}

// TestMidjourneyJobEntity 测试Midjourney任务实体
func TestMidjourneyJobEntity(t *testing.T) {
	job := &entities.MidjourneyJob{
		JobID:    "test-job-123",
		UserID:   1,
		APIKeyID: 1,
		Action:   entities.MidjourneyJobActionImagine,
		Status:   entities.MidjourneyJobStatusPendingQueue,
		Mode:     entities.MidjourneyJobModeFast,
		Progress: 0,
	}
	
	// 测试请求参数设置和获取
	params := map[string]interface{}{
		"prompt":  "test prompt",
		"botType": "MID_JOURNEY",
	}
	
	err := job.SetRequestParams(params)
	assert.NoError(t, err)
	
	retrievedParams, err := job.GetRequestParams()
	assert.NoError(t, err)
	assert.Equal(t, "test prompt", retrievedParams["prompt"])
	assert.Equal(t, "MID_JOURNEY", retrievedParams["botType"])
	
	// 测试图片列表设置和获取
	images := []string{
		"https://example.com/image1.png",
		"https://example.com/image2.png",
	}
	
	err = job.SetImages(images)
	assert.NoError(t, err)
	
	retrievedImages, err := job.GetImages()
	assert.NoError(t, err)
	assert.Equal(t, images, retrievedImages)
	
	// 测试操作按钮设置和获取
	components := []string{"upsample1", "upsample2", "variation1", "variation2"}
	
	err = job.SetComponents(components)
	assert.NoError(t, err)
	
	retrievedComponents, err := job.GetComponents()
	assert.NoError(t, err)
	assert.Equal(t, components, retrievedComponents)
	
	// 测试状态检查方法
	assert.True(t, job.IsProcessing())
	assert.False(t, job.IsCompleted())
	assert.False(t, job.IsSuccess())
	assert.False(t, job.IsFailed())
	
	// 更改状态并重新测试
	job.Status = entities.MidjourneyJobStatusSuccess
	assert.False(t, job.IsProcessing())
	assert.True(t, job.IsCompleted())
	assert.True(t, job.IsSuccess())
	assert.False(t, job.IsFailed())
}

// TestDefaultTimeout 测试默认超时时间
func TestDefaultTimeout(t *testing.T) {
	fastTimeout := entities.GetDefaultTimeout(entities.MidjourneyJobModeFast)
	assert.Equal(t, 300, fastTimeout)
	
	turboTimeout := entities.GetDefaultTimeout(entities.MidjourneyJobModeTurbo)
	assert.Equal(t, 300, turboTimeout)
	
	relaxTimeout := entities.GetDefaultTimeout(entities.MidjourneyJobModeRelax)
	assert.Equal(t, 600, relaxTimeout)
}

// TestDefaultComponents 测试默认操作按钮
func TestDefaultComponents(t *testing.T) {
	components := entities.GetDefaultComponents()
	
	expectedComponents := []string{
		"upsample1", "upsample2", "upsample3", "upsample4",
		"variation1", "variation2", "variation3", "variation4",
	}
	
	assert.Equal(t, expectedComponents, components)
}
