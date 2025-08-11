package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/config"
	"ai-api-gateway/internal/infrastructure/logger"
	"ai-api-gateway/internal/infrastructure/redis"
)

// MidjourneyQueueService Midjourney任务队列服务
type MidjourneyQueueService interface {
	// StartWorkers 启动工作进程
	StartWorkers(ctx context.Context, workerCount int) error

	// StopWorkers 停止工作进程
	StopWorkers() error

	// EnqueueJob 将任务加入队列
	EnqueueJob(ctx context.Context, job *entities.MidjourneyJob) error

	// GetQueueStats 获取队列统计信息
	GetQueueStats(ctx context.Context) (*QueueStats, error)

	// ProcessExpiredJobs 处理超时任务
	ProcessExpiredJobs(ctx context.Context) error
}

// QueueStats 队列统计信息
type QueueStats struct {
	PendingJobs    int64 `json:"pending_jobs"`
	ProcessingJobs int64 `json:"processing_jobs"`
	CompletedJobs  int64 `json:"completed_jobs"`
	FailedJobs     int64 `json:"failed_jobs"`
	WorkerCount    int   `json:"worker_count"`
	ActiveWorkers  int   `json:"active_workers"`
}

// midjourneyQueueServiceImpl 队列服务实现
type midjourneyQueueServiceImpl struct {
	jobRepo                  repositories.MidjourneyJobRepository
	cache                    *redis.CacheService
	logger                   logger.Logger
	workers                  []*worker
	workerWg                 sync.WaitGroup
	stopCh                   chan struct{}
	jobCh                    chan *entities.MidjourneyJob // 任务队列 channel
	mu                       sync.RWMutex
	isRunning                bool
	webhookService           WebhookService
	imageGenService          ImageGenerationService
	providerModelSupportRepo repositories.ProviderModelSupportRepository // 用于获取模型支持的提供商
	providerRepo             repositories.ProviderRepository             // 用于获取提供商详情
	billingService           BillingService                              // 计费服务
}

// worker 工作进程
type worker struct {
	id      int
	service *midjourneyQueueServiceImpl
	logger  logger.Logger
}

// NewMidjourneyQueueService 创建队列服务（兼容旧接口）
func NewMidjourneyQueueService(
	jobRepo repositories.MidjourneyJobRepository,
	cache *redis.CacheService,
	webhookService WebhookService,
	imageGenService ImageGenerationService,
	providerModelSupportRepo repositories.ProviderModelSupportRepository,
	providerRepo repositories.ProviderRepository,
	billingService BillingService,
	logger logger.Logger,
) MidjourneyQueueService {
	defaultConfig := &config.MidjourneyConfig{
		ChannelSize:  1000,
		WorkerCount:  3,
		MaxRetries:   60,
		PollInterval: 5,
	}
	return NewMidjourneyQueueServiceWithConfig(
		jobRepo, cache, webhookService, imageGenService,
		providerModelSupportRepo, providerRepo, billingService,
		defaultConfig, logger,
	)
}

// NewMidjourneyQueueServiceWithConfig 创建队列服务（带配置）
func NewMidjourneyQueueServiceWithConfig(
	jobRepo repositories.MidjourneyJobRepository,
	cache *redis.CacheService,
	webhookService WebhookService,
	imageGenService ImageGenerationService,
	providerModelSupportRepo repositories.ProviderModelSupportRepository,
	providerRepo repositories.ProviderRepository,
	billingService BillingService,
	mjConfig *config.MidjourneyConfig,
	logger logger.Logger,
) MidjourneyQueueService {
	return &midjourneyQueueServiceImpl{
		jobRepo:                  jobRepo,
		cache:                    cache,
		logger:                   logger,
		stopCh:                   make(chan struct{}),
		jobCh:                    make(chan *entities.MidjourneyJob, mjConfig.ChannelSize),
		webhookService:           webhookService,
		imageGenService:          imageGenService,
		providerModelSupportRepo: providerModelSupportRepo,
		providerRepo:             providerRepo,
		billingService:           billingService,
	}
}

// StartWorkers 启动工作进程
func (s *midjourneyQueueServiceImpl) StartWorkers(ctx context.Context, workerCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("workers already running")
	}

	s.workers = make([]*worker, workerCount)
	s.stopCh = make(chan struct{})

	for i := 0; i < workerCount; i++ {
		worker := &worker{
			id:      i + 1,
			service: s,
			logger:  s.logger.WithField("worker_id", i+1),
		}
		s.workers[i] = worker

		s.workerWg.Add(1)
		go worker.run(ctx)
	}

	// 启动时加载所有待处理任务
	if err := s.loadPendingJobsOnStartup(ctx); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to load pending jobs on startup")
		// 不返回错误，继续启动服务
	}

	s.isRunning = true
	return nil
}

// StopWorkers 停止工作进程
func (s *midjourneyQueueServiceImpl) StopWorkers() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	// 发送停止信号
	close(s.stopCh)

	// 关闭任务队列，通知所有 worker 停止
	close(s.jobCh)

	// 等待所有工作进程结束
	s.workerWg.Wait()

	s.isRunning = false
	s.workers = nil
	s.logger.Info("Midjourney queue workers stopped")

	return nil
}

// EnqueueJob 将任务加入队列
func (s *midjourneyQueueServiceImpl) EnqueueJob(ctx context.Context, job *entities.MidjourneyJob) error {
	// 将任务状态设置为等待队列
	if err := s.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusPendingQueue); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// 直接将任务放入 channel，避免轮询延迟
	select {
	case s.jobCh <- job:
		s.logger.WithFields(map[string]interface{}{
			"job_id": job.JobID,
			"action": job.Action,
		}).Info("Job enqueued")
	default:
		// channel 满了，记录错误
		s.logger.WithFields(map[string]interface{}{
			"job_id":       job.JobID,
			"channel_len":  len(s.jobCh),
			"channel_cap":  cap(s.jobCh),
		}).Error("Job channel is full")
		return fmt.Errorf("job channel is full")
	}

	return nil
}

// GetQueueStats 获取队列统计信息
func (s *midjourneyQueueServiceImpl) GetQueueStats(ctx context.Context) (*QueueStats, error) {
	stats, err := s.jobRepo.GetJobStats(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get job stats: %w", err)
	}

	s.mu.RLock()
	workerCount := len(s.workers)
	activeWorkers := 0
	if s.isRunning {
		activeWorkers = workerCount
	}
	s.mu.RUnlock()

	return &QueueStats{
		PendingJobs:    stats.PendingJobs,
		ProcessingJobs: stats.ProcessingJobs,
		CompletedJobs:  stats.SuccessJobs,
		FailedJobs:     stats.FailedJobs,
		WorkerCount:    workerCount,
		ActiveWorkers:  activeWorkers,
	}, nil
}

// ProcessExpiredJobs 处理超时任务
func (s *midjourneyQueueServiceImpl) ProcessExpiredJobs(ctx context.Context) error {
	jobs, err := s.jobRepo.GetExpiredJobs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get expired jobs: %w", err)
	}

	if len(jobs) == 0 {
		return nil
	}

	s.logger.WithField("count", len(jobs)).Info("Processing expired jobs")

	for _, job := range jobs {
		// 标记为失败
		if err := s.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusFailed); err != nil {
			s.logger.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"job_id": job.JobID,
			}).Error("Failed to mark expired job as failed")
			continue
		}

		// 设置错误信息
		result := &repositories.MidjourneyJobResult{
			ErrorMessage: stringPtr("Job timeout"),
		}
		if err := s.jobRepo.UpdateResult(ctx, job.JobID, result); err != nil {
			s.logger.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"job_id": job.JobID,
			}).Error("Failed to set timeout error message")
		}

		// 发送webhook通知
		if job.HookURL != nil && *job.HookURL != "" {
			s.sendWebhook(ctx, job, "FAILED", "Job timeout")
		}
	}

	return nil
}

// worker.run 工作进程运行逻辑
func (w *worker) run(ctx context.Context) {
	defer w.service.workerWg.Done()

	for {
		// 首先检查是否有job可以处理（非阻塞检查）
		select {
		case job, ok := <-w.service.jobCh:
			if !ok {
				// channel 已关闭，退出
				return
			}

			if job == nil {
				continue
			}

			w.logger.WithFields(map[string]interface{}{
				"worker_id": w.id,
				"job_id":    job.JobID,
				"action":    job.Action,
			}).Info("Processing job")

			// 异步处理任务，避免阻塞 worker
			go w.processJobAsync(ctx, job)
			continue
		default:
			// 没有job可以处理，检查是否需要退出
		}
		
		// 如果没有job，再检查退出信号
		select {
		case <-w.service.stopCh:
			return
		case <-ctx.Done():
			return
		case job, ok := <-w.service.jobCh:
			if !ok {
				// channel 已关闭，退出
				return
			}

			if job == nil {
				continue
			}

			w.logger.WithFields(map[string]interface{}{
				"worker_id": w.id,
				"job_id":    job.JobID,
				"action":    job.Action,
			}).Info("Processing job")

			// 异步处理任务，避免阻塞 worker
			go w.processJobAsync(ctx, job)
		}
	}
}

// processJobAsync 异步处理任务
func (w *worker) processJobAsync(ctx context.Context, job *entities.MidjourneyJob) {
	// 更新状态为处理中 - 这里是防重复处理的关键点
	if err := w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusOnQueue); err != nil {
		w.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": job.JobID,
		}).Error("Failed to update job status to OnQueue")
		return
	}

	// 处理任务
	if err := w.processJob(ctx, job); err != nil {
		w.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": job.JobID,
		}).Error("Job processing failed")

		// 标记为失败
		w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusFailed)

		// 设置错误信息
		result := &repositories.MidjourneyJobResult{
			ErrorMessage: stringPtr(err.Error()),
		}
		w.service.jobRepo.UpdateResult(ctx, job.JobID, result)

		// 发送webhook通知
		if job.HookURL != nil && *job.HookURL != "" {
			w.service.sendWebhook(ctx, job, "FAILED", err.Error())
		}

		// 处理失败任务的计费（不扣费）
		if w.service.billingService != nil {
			if err := w.service.billingService.ProcessMidjourneyBilling(ctx, job.JobID, false); err != nil {
				w.logger.WithFields(map[string]interface{}{
					"job_id": job.JobID,
					"error":  err.Error(),
				}).Error("Failed to process billing for failed job")
			}
		}
	}
}

// processJob 处理单个任务
func (w *worker) processJob(ctx context.Context, job *entities.MidjourneyJob) error {
	// 根据任务类型调用相应的处理逻辑
	switch job.Action {
	case entities.MidjourneyJobActionImagine:
		return w.processImagineJob(ctx, job)
	case entities.MidjourneyJobActionAction:
		return w.processActionJob(ctx, job)
	case entities.MidjourneyJobActionBlend:
		return w.processBlendJob(ctx, job)
	case entities.MidjourneyJobActionDescribe:
		return w.processDescribeJob(ctx, job)
	case entities.MidjourneyJobActionInpaint:
		return w.processInpaintJob(ctx, job)
	default:
		return fmt.Errorf("unsupported job action: %s", job.Action)
	}
}

// processImagineJob 处理图像生成任务
func (w *worker) processImagineJob(ctx context.Context, job *entities.MidjourneyJob) error {
	// 获取 Midjourney 提供商
	provider, err := w.getMidjourneyProvider(ctx, "midjourney")
	if err != nil {
		w.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": job.JobID,
		}).Error("No Midjourney provider available")

		// 标记任务为失败
		w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusFailed)
		result := &repositories.MidjourneyJobResult{
			ErrorMessage: stringPtr(fmt.Sprintf("No available Midjourney provider: %s", err.Error())),
		}
		w.service.jobRepo.UpdateResult(ctx, job.JobID, result)

		// 发送失败webhook
		if job.HookURL != nil && *job.HookURL != "" {
			w.service.sendWebhook(ctx, job, "FAILED", fmt.Sprintf("No available provider: %s", err.Error()))
		}

		return err
	}

	// 从任务参数中获取请求数据
	params, err := job.GetRequestParams()
	if err != nil {
		w.logger.WithFields(map[string]interface{}{
			"job_id": job.JobID,
			"error":  err.Error(),
		}).Error("=== FAILED TO GET REQUEST PARAMS ===")
		return fmt.Errorf("failed to get request params: %w", err)
	}

	w.logger.WithFields(map[string]interface{}{
		"job_id": job.JobID,
		"params": params,
	}).Info("=== GOT REQUEST PARAMS ===")

	// 构造请求体
	requestBody, err := json.Marshal(params)
	if err != nil {
		w.logger.WithFields(map[string]interface{}{
			"job_id": job.JobID,
			"error":  err.Error(),
			"params": params,
		}).Error("=== FAILED TO MARSHAL REQUEST BODY ===")
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	w.logger.WithFields(map[string]interface{}{
		"job_id":           job.JobID,
		"request_body_raw": string(requestBody),
		"body_length":      len(requestBody),
	}).Info("=== CONSTRUCTED REQUEST BODY ===")

	// 构造请求头
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// 设置认证头
	if provider.APIKeyEncrypted != nil {
		headers["mj-api-secret"] = *provider.APIKeyEncrypted
		w.logger.WithFields(map[string]interface{}{
			"job_id":        job.JobID,
			"api_key_set":   true,
			"api_key_first": (*provider.APIKeyEncrypted)[:10] + "...",
		}).Info("=== API KEY SET ===")
	} else {
		w.logger.WithFields(map[string]interface{}{
			"job_id":      job.JobID,
			"api_key_set": false,
		}).Warn("=== NO API KEY FOUND ===")
	}

	w.logger.WithFields(map[string]interface{}{
		"job_id":  job.JobID,
		"headers": headers,
	}).Info("=== FINAL HEADERS ===")

	w.logger.WithFields(map[string]interface{}{
		"job_id":      job.JobID,
		"provider_id": provider.ID,
		"provider":    provider.Name,
		"base_url":    provider.BaseURL,
		"endpoint":    "/mj/submit/imagine",
		"params":      params,
		"headers":     headers,
		"body_size":   len(requestBody),
	}).Info("=== FORWARDING IMAGINE REQUEST TO UPSTREAM ===")

	// 打印请求体内容（用于调试）
	w.logger.WithFields(map[string]interface{}{
		"job_id":       job.JobID,
		"request_body": string(requestBody),
	}).Info("Request body content")

	// 处理 Midjourney API 的 base URL - 去掉 /v1 后缀
	midjourneyBaseURL := provider.BaseURL
	if strings.HasSuffix(midjourneyBaseURL, "/v1") {
		midjourneyBaseURL = strings.TrimSuffix(midjourneyBaseURL, "/v1")
		w.logger.WithFields(map[string]interface{}{
			"job_id":         job.JobID,
			"original_url":   provider.BaseURL,
			"midjourney_url": midjourneyBaseURL,
		}).Info("=== ADJUSTED BASE URL FOR MIDJOURNEY ===")
	}

	// 创建代理客户端
	proxyClient := clients.NewMidjourneyProxyClient(
		midjourneyBaseURL,
		*provider.APIKeyEncrypted,
		w.logger,
	)

	w.logger.WithFields(map[string]interface{}{
		"job_id":  job.JobID,
		"message": "About to call proxyClient.ForwardRequest",
	}).Info("=== CALLING PROXY CLIENT ===")

	// 转发请求到上游
	response, err := proxyClient.ForwardRequest(
		ctx,
		"POST",
		"/mj/submit/imagine",
		headers,
		requestBody,
		nil,
	)

	w.logger.WithFields(map[string]interface{}{
		"job_id": job.JobID,
		"error":  err,
	}).Info("=== PROXY CLIENT CALL COMPLETED ===")

	if err != nil {
		w.logger.WithFields(map[string]interface{}{
			"job_id": job.JobID,
			"error":  err.Error(),
		}).Error("=== FAILED TO FORWARD IMAGINE REQUEST ===")
		return fmt.Errorf("failed to forward imagine request: %w", err)
	}

	w.logger.WithFields(map[string]interface{}{
		"job_id":      job.JobID,
		"status_code": response.StatusCode,
		"body_size":   len(response.Body),
		"body":        string(response.Body),
	}).Info("=== RECEIVED UPSTREAM RESPONSE ===")

	fmt.Printf("=== MID JOURNEY RESPONSE ===\nJob ID: %s\nStatus Code: %d\nResponse Body: %s\n=== END RESPONSE ===\n",
		job.JobID, response.StatusCode, string(response.Body))

	// 处理上游响应
	return w.handleUpstreamResponse(ctx, job, response, proxyClient)
}

// processActionJob 处理操作任务
func (w *worker) processActionJob(ctx context.Context, job *entities.MidjourneyJob) error {
	return w.processGenericJob(ctx, job, "/mj/submit/action")
}

// processBlendJob 处理混合任务
func (w *worker) processBlendJob(ctx context.Context, job *entities.MidjourneyJob) error {
	return w.processGenericJob(ctx, job, "/mj/submit/blend")
}

// processDescribeJob 处理描述任务
func (w *worker) processDescribeJob(ctx context.Context, job *entities.MidjourneyJob) error {
	return w.processGenericJob(ctx, job, "/mj/submit/describe")
}

// processInpaintJob 处理修复任务
func (w *worker) processInpaintJob(ctx context.Context, job *entities.MidjourneyJob) error {
	return w.processGenericJob(ctx, job, "/mj/submit/modal")
}

// processGenericJob 通用任务处理方法
func (w *worker) processGenericJob(ctx context.Context, job *entities.MidjourneyJob, endpoint string) error {
	// 获取 Midjourney 提供商
	provider, err := w.getMidjourneyProvider(ctx, "midjourney")
	if err != nil {
		w.logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"job_id":   job.JobID,
			"endpoint": endpoint,
		}).Error("No Midjourney provider available")

		// 标记任务为失败
		w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusFailed)
		result := &repositories.MidjourneyJobResult{
			ErrorMessage: stringPtr(fmt.Sprintf("No available Midjourney provider: %s", err.Error())),
		}
		w.service.jobRepo.UpdateResult(ctx, job.JobID, result)

		// 发送失败webhook
		if job.HookURL != nil && *job.HookURL != "" {
			w.service.sendWebhook(ctx, job, "FAILED", fmt.Sprintf("No available provider: %s", err.Error()))
		}

		return err
	}

	// 从任务参数中获取请求数据
	params, err := job.GetRequestParams()
	if err != nil {
		return fmt.Errorf("failed to get request params: %w", err)
	}

	// 构造请求体
	requestBody, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 构造请求头
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// 设置认证头
	if provider.APIKeyEncrypted != nil {
		headers["mj-api-secret"] = *provider.APIKeyEncrypted
	}

	w.logger.WithFields(map[string]interface{}{
		"job_id":      job.JobID,
		"provider_id": provider.ID,
		"provider":    provider.Name,
		"endpoint":    endpoint,
		"params":      params,
	}).Info("Forwarding request to upstream")

	// 创建代理客户端
	proxyClient := clients.NewMidjourneyProxyClient(
		provider.BaseURL,
		*provider.APIKeyEncrypted,
		w.logger,
	)

	// 转发请求到上游
	response, err := proxyClient.ForwardRequest(
		ctx,
		"POST",
		endpoint,
		headers,
		requestBody,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to forward request to %s: %w", endpoint, err)
	}

	// 处理上游响应
	return w.handleUpstreamResponse(ctx, job, response, proxyClient)
}

// sendWebhook 发送webhook通知
func (s *midjourneyQueueServiceImpl) sendWebhook(ctx context.Context, job *entities.MidjourneyJob, status, message string) {
	if s.webhookService == nil {
		return
	}

	webhookData := map[string]interface{}{
		"id":      job.JobID,
		"status":  status,
		"message": message,
		"action":  job.Action,
	}

	if err := s.webhookService.SendWebhook(ctx, *job.HookURL, webhookData); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"job_id":   job.JobID,
			"hook_url": *job.HookURL,
		}).Error("Failed to send webhook")
	}
}

// getPriority 获取任务优先级
func getPriority(mode entities.MidjourneyJobMode) int {
	switch mode {
	case entities.MidjourneyJobModeTurbo:
		return 1 // 最高优先级
	case entities.MidjourneyJobModeFast:
		return 2
	case entities.MidjourneyJobModeRelax:
		return 3 // 最低优先级
	default:
		return 2
	}
}

// 辅助函数
func stringPtr(s string) *string {
	return &s
}

// loadPendingJobsOnStartup 启动时加载所有待处理任务到 channel
func (s *midjourneyQueueServiceImpl) loadPendingJobsOnStartup(ctx context.Context) error {
	// TODO: 如果需要多实例部署，可以在这里添加分布式锁逻辑

	s.logger.Info("Loading pending jobs on startup...")

	// 获取所有待处理任务
	jobs, err := s.jobRepo.GetPendingJobs(ctx, 10000) // 一次最多加载10000个任务
	if err != nil {
		return fmt.Errorf("failed to get pending jobs: %w", err)
	}

	if len(jobs) == 0 {
		s.logger.Info("No pending jobs found on startup")
		return nil
	}

	s.logger.WithField("job_count", len(jobs)).Info("Loading pending jobs into channel")

	// 将任务加载到 channel
	loadedCount := 0
	for _, job := range jobs {
		select {
		case s.jobCh <- job:
			loadedCount++
		case <-ctx.Done():
			s.logger.WithField("loaded_count", loadedCount).Info("Context cancelled during job loading")
			return ctx.Err()
		default:
			// channel 满了，记录警告
			s.logger.WithFields(map[string]interface{}{
				"loaded_count": loadedCount,
				"total_jobs":   len(jobs),
			}).Warn("Job channel is full during startup loading")
			break
		}
	}

	s.logger.WithFields(map[string]interface{}{
		"loaded_count": loadedCount,
		"total_jobs":   len(jobs),
	}).Info("Pending jobs loaded successfully on startup")

	return nil
}

func intPtr(i int) *int {
	return &i
}

// handleUpstreamResponse 处理上游响应
func (w *worker) handleUpstreamResponse(ctx context.Context, job *entities.MidjourneyJob, response *clients.ProxyResponse, proxyClient clients.MidjourneyProxyClient) error {
	w.logger.WithFields(map[string]interface{}{
		"job_id":      job.JobID,
		"status_code": response.StatusCode,
		"body_size":   len(response.Body),
	}).Info("Received upstream response")

	// 如果上游返回错误状态码
	if response.StatusCode >= 400 {
		errorMsg := fmt.Sprintf("Upstream error: %d - %s", response.StatusCode, string(response.Body))

		// 更新任务为失败状态
		if err := w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusFailed); err != nil {
			w.logger.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"job_id": job.JobID,
			}).Error("Failed to update job status to failed")
		}

		// 设置错误信息
		result := &repositories.MidjourneyJobResult{
			ErrorMessage: stringPtr(errorMsg),
		}
		if err := w.service.jobRepo.UpdateResult(ctx, job.JobID, result); err != nil {
			w.logger.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"job_id": job.JobID,
			}).Error("Failed to set error message")
		}

		// 发送失败webhook
		if job.HookURL != nil && *job.HookURL != "" {
			w.service.sendWebhook(ctx, job, "FAILED", errorMsg)
		}

		return fmt.Errorf(errorMsg)
	}

	// 解析上游响应
	var upstreamResult map[string]interface{}
	if err := json.Unmarshal(response.Body, &upstreamResult); err != nil {
		w.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": job.JobID,
		}).Error("Failed to parse upstream response")

		// 仍然标记为成功，但记录解析错误
		upstreamResult = map[string]interface{}{
			"raw_response": string(response.Body),
		}
	}

	// 检查上游任务是否成功提交
	if code, ok := upstreamResult["code"].(float64); ok && code == 1 {
		// 上游任务提交成功，获取上游任务ID
		var upstreamTaskID string
		w.logger.WithFields(map[string]interface{}{
			"job_id":         job.JobID,
			"upstreamResult": upstreamResult,
		}).Info("Upstream task submitted successfully")
		if result, ok := upstreamResult["result"].(string); ok {
			upstreamTaskID = result
		}

		w.logger.WithFields(map[string]interface{}{
			"job_id":           job.JobID,
			"upstream_task_id": upstreamTaskID,
		}).Info("Upstream task submitted successfully")

		// 如果有上游任务ID，保存到数据库并开始轮询任务状态
		if upstreamTaskID != "" {
			// 保存上游任务ID到数据库
			if err := w.service.jobRepo.UpdateUpstreamTaskID(ctx, job.JobID, upstreamTaskID); err != nil {
				w.logger.WithFields(map[string]interface{}{
					"error":            err.Error(),
					"job_id":           job.JobID,
					"upstream_task_id": upstreamTaskID,
				}).Error("=== CRITICAL: Failed to save upstream task ID ===")
				
				// upstream_task_id保存失败是严重错误，应该标记任务失败
				w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusFailed)
				result := &repositories.MidjourneyJobResult{
					ErrorMessage: stringPtr(fmt.Sprintf("Failed to save upstream task ID: %s", err.Error())),
				}
				w.service.jobRepo.UpdateResult(ctx, job.JobID, result)
				return fmt.Errorf("critical error: failed to save upstream task ID: %w", err)
			}

			w.logger.WithFields(map[string]interface{}{
				"job_id":           job.JobID,
				"upstream_task_id": upstreamTaskID,
			}).Info("=== UPSTREAM TASK ID SAVED SUCCESSFULLY ===")

			return w.pollUpstreamTask(ctx, job, upstreamTaskID, proxyClient)
		} else {
			// 没有上游任务ID，直接标记为成功
			return w.markJobAsSuccess(ctx, job, upstreamResult)
		}
	} else {
		// 上游任务提交失败
		description := "Unknown error"
		if desc, ok := upstreamResult["description"].(string); ok {
			description = desc
		}

		errorMsg := fmt.Sprintf("Upstream task submission failed: %s", description)

		// 更新任务为失败状态
		if err := w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusFailed); err != nil {
			w.logger.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"job_id": job.JobID,
			}).Error("Failed to update job status to failed")
		}

		// 设置错误信息
		result := &repositories.MidjourneyJobResult{
			ErrorMessage: stringPtr(errorMsg),
		}
		if err := w.service.jobRepo.UpdateResult(ctx, job.JobID, result); err != nil {
			w.logger.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"job_id": job.JobID,
			}).Error("Failed to set error message")
		}

		// 发送失败webhook
		if job.HookURL != nil && *job.HookURL != "" {
			w.service.sendWebhook(ctx, job, "FAILED", errorMsg)
		}

		return fmt.Errorf(errorMsg)
	}
}

// pollUpstreamTask 轮询上游任务状态
func (w *worker) pollUpstreamTask(ctx context.Context, job *entities.MidjourneyJob, upstreamTaskID string, proxyClient clients.MidjourneyProxyClient) error {
	maxRetries := 60 // 最多轮询5分钟（每5秒一次）

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 获取上游任务状态
		response, err := proxyClient.ForwardRequest(
			ctx,
			"GET",
			fmt.Sprintf("/mj/task/%s/fetch", upstreamTaskID),
			map[string]string{},
			nil,
			nil,
		)

		if err != nil {
			w.logger.WithFields(map[string]interface{}{
				"error":            err.Error(),
				"job_id":           job.JobID,
				"upstream_task_id": upstreamTaskID,
				"retry":            i + 1,
			}).Warn("Failed to fetch upstream task status, retrying...")

			time.Sleep(5 * time.Second)
			continue
		}

		// 解析上游任务状态
		var taskResult map[string]interface{}
		if err := json.Unmarshal(response.Body, &taskResult); err != nil {
			w.logger.WithFields(map[string]interface{}{
				"error":            err.Error(),
				"job_id":           job.JobID,
				"upstream_task_id": upstreamTaskID,
			}).Error("Failed to parse upstream task response")

			time.Sleep(5 * time.Second)
			continue
		}

		// 检查任务状态
		status, _ := taskResult["status"].(string)
		progress, _ := taskResult["progress"].(string)

		w.logger.WithFields(map[string]interface{}{
			"job_id":           job.JobID,
			"upstream_task_id": upstreamTaskID,
			"status":           status,
			"progress_raw":     progress,
		}).Info("=== POLLING UPSTREAM TASK STATUS ===")

		// 更新本地任务进度
		if progressInt := parseProgress(progress); progressInt >= 0 {
			w.logger.WithFields(map[string]interface{}{
				"job_id":       job.JobID,
				"progress_int": progressInt,
			}).Info("=== UPDATING PROGRESS ===")
			if err := w.service.jobRepo.UpdateProgress(ctx, job.JobID, progressInt); err != nil {
				w.logger.WithFields(map[string]interface{}{
					"job_id": job.JobID,
					"error":  err.Error(),
				}).Error("=== FAILED TO UPDATE PROGRESS ===")
			}
		}

		switch status {
		case "SUCCESS":
			// 任务成功完成
			return w.markJobAsSuccess(ctx, job, taskResult)

		case "FAILED", "FAILURE":
			// 任务失败
			failReason, _ := taskResult["failReason"].(string)
			if failReason == "" {
				failReason = "Unknown failure"
			}

			w.logger.WithFields(map[string]interface{}{
				"job_id":           job.JobID,
				"upstream_task_id": upstreamTaskID,
				"fail_reason":      failReason,
			}).Error("=== UPSTREAM TASK FAILED ===")

			// 更新任务为失败状态
			w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusFailed)

			// 设置错误信息
			result := &repositories.MidjourneyJobResult{
				ErrorMessage: stringPtr(failReason),
			}
			w.service.jobRepo.UpdateResult(ctx, job.JobID, result)

			// 发送失败webhook
			if job.HookURL != nil && *job.HookURL != "" {
				w.service.sendWebhook(ctx, job, "FAILED", failReason)
			}

			return fmt.Errorf("upstream task failed: %s", failReason)

		case "IN_PROGRESS", "PENDING", "PROCESSING":
			// 任务进行中，继续等待

			// 发送进度webhook
			if job.HookURL != nil && *job.HookURL != "" {
				w.service.sendWebhook(ctx, job, "IN_PROGRESS", fmt.Sprintf("Progress: %s", progress))
			}

			time.Sleep(5 * time.Second)
			continue

		default:
			// 未知状态，继续等待
			w.logger.WithFields(map[string]interface{}{
				"job_id":           job.JobID,
				"upstream_task_id": upstreamTaskID,
				"unknown_status":   status,
			}).Warn("=== UNKNOWN UPSTREAM STATUS, CONTINUING ===")
			time.Sleep(5 * time.Second)
			continue
		}
	}

	// 超时
	errorMsg := fmt.Sprintf("Upstream task timeout after %d retries", maxRetries)

	w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusFailed)
	result := &repositories.MidjourneyJobResult{
		ErrorMessage: stringPtr(errorMsg),
	}
	w.service.jobRepo.UpdateResult(ctx, job.JobID, result)

	if job.HookURL != nil && *job.HookURL != "" {
		w.service.sendWebhook(ctx, job, "FAILED", errorMsg)
	}

	return fmt.Errorf(errorMsg)
}

// markJobAsSuccess 标记任务为成功
func (w *worker) markJobAsSuccess(ctx context.Context, job *entities.MidjourneyJob, taskResult map[string]interface{}) error {

	// 构造结果
	result := &repositories.MidjourneyJobResult{}

	// 处理主图片URL
	if imageURL, ok := taskResult["imageUrl"].(string); ok && imageURL != "" {
		result.DiscordImage = stringPtr(imageURL)
		result.CDNImage = stringPtr(imageURL)
	}

	// 处理四张小图URLs
	if imageUrls, ok := taskResult["imageUrls"].([]interface{}); ok && len(imageUrls) > 0 {
		var urls []string
		for _, urlObj := range imageUrls {
			if urlMap, ok := urlObj.(map[string]interface{}); ok {
				if url, ok := urlMap["url"].(string); ok && url != "" {
					urls = append(urls, url)
				}
			}
		}
		if len(urls) > 0 {
			result.Images = urls
		}
	}

	// 设置图片尺寸
	if width, ok := taskResult["imageWidth"].(float64); ok {
		result.Width = intPtr(int(width))
	} else {
		result.Width = intPtr(1024) // 默认值
	}

	if height, ok := taskResult["imageHeight"].(float64); ok {
		result.Height = intPtr(int(height))
	} else {
		result.Height = intPtr(1024) // 默认值
	}

	// 处理操作按钮
	if buttons, ok := taskResult["buttons"].([]interface{}); ok && len(buttons) > 0 {
		// 从按钮数据中提取 customId 作为组件标识
		var components []string
		for _, btnObj := range buttons {
			if btnMap, ok := btnObj.(map[string]interface{}); ok {
				if customId, ok := btnMap["customId"].(string); ok && customId != "" {
					// 提取按钮类型，如 U1, V1 等
					if label, ok := btnMap["label"].(string); ok && label != "" {
						components = append(components, label)
					} else {
						// 如果没有标签，使用自定义ID的一部分
						components = append(components, customId)
					}
				}
			}
		}

		if len(components) > 0 {
			result.Components = components
		}
	}

	// 如果没有提取到按钮数据，使用默认按钮
	if len(result.Components) == 0 && job.Action == entities.MidjourneyJobActionImagine {
		result.Components = entities.GetDefaultComponents()
	}

	// 更新结果
	if err := w.service.jobRepo.UpdateResult(ctx, job.JobID, result); err != nil {
		w.logger.WithFields(map[string]interface{}{
			"job_id": job.JobID,
			"error":  err.Error(),
		}).Error("=== FAILED TO UPDATE RESULT ===")
		return fmt.Errorf("failed to update job result: %w", err)
	}

	// 更新进度为100%
	if err := w.service.jobRepo.UpdateProgress(ctx, job.JobID, 100); err != nil {
		w.logger.WithFields(map[string]interface{}{
			"job_id": job.JobID,
			"error":  err.Error(),
		}).Error("=== FAILED TO UPDATE PROGRESS ===")
		return fmt.Errorf("failed to update progress: %w", err)
	}

	// 更新状态为成功
	if err := w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusSuccess); err != nil {
		w.logger.WithFields(map[string]interface{}{
			"job_id": job.JobID,
			"error":  err.Error(),
		}).Error("=== FAILED TO UPDATE STATUS ===")
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// 发送成功webhook
	if job.HookURL != nil && *job.HookURL != "" {
		w.service.sendWebhook(ctx, job, "SUCCESS", "Job completed successfully")
	}

	// 处理成功任务的计费
	if w.service.billingService != nil {
		if err := w.service.billingService.ProcessMidjourneyBilling(ctx, job.JobID, true); err != nil {
			w.logger.WithFields(map[string]interface{}{
				"job_id": job.JobID,
				"error":  err.Error(),
			}).Error("Failed to process billing for successful job")
			// 不返回错误，避免影响任务完成状态
		}
	}

	w.logger.WithFields(map[string]interface{}{
		"job_id": job.JobID,
	}).Info("=== JOB COMPLETED SUCCESSFULLY ===")

	return nil
}

// parseProgress 解析进度字符串
func parseProgress(progress string) int {
	if progress == "" {
		return 0
	}

	// 尝试解析百分比格式，如 "50%"
	if strings.HasSuffix(progress, "%") {
		if val, err := strconv.Atoi(strings.TrimSuffix(progress, "%")); err == nil {
			return val
		}
	}

	// 尝试直接解析数字
	if val, err := strconv.Atoi(progress); err == nil {
		return val
	}

	return 0
}

// getMidjourneyProvider 获取 Midjourney 提供商
func (w *worker) getMidjourneyProvider(ctx context.Context, modelSlug string) (*entities.Provider, error) {
	// 获取支持该模型的提供商列表
	supportInfos, err := w.service.providerModelSupportRepo.GetSupportingProviders(ctx, modelSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get supporting providers for model %s: %w", modelSlug, err)
	}

	if len(supportInfos) == 0 {
		return nil, fmt.Errorf("no providers configured for model %s. Please add providers to the database using the SQL script in docs/MIDJOURNEY_DATABASE_SETUP.sql", modelSlug)
	}

	var unavailableReasons []string

	// 遍历支持信息，获取可用的提供商
	for _, supportInfo := range supportInfos {
		if !supportInfo.Enabled {
			unavailableReasons = append(unavailableReasons, fmt.Sprintf("provider support disabled for model %s", modelSlug))
			continue
		}

		// 检查是否已经包含提供商信息
		if supportInfo.Provider != nil {
			// 检查提供商是否可用
			if !supportInfo.Provider.IsAvailable() {
				unavailableReasons = append(unavailableReasons, fmt.Sprintf("provider %s is not available (status: %s, health: %s)",
					supportInfo.Provider.Name, supportInfo.Provider.Status, supportInfo.Provider.HealthStatus))
				continue
			}
			if supportInfo.Provider.APIKeyEncrypted == nil {
				unavailableReasons = append(unavailableReasons, fmt.Sprintf("provider %s has no API key configured", supportInfo.Provider.Name))
				continue
			}
			return supportInfo.Provider, nil
		} else {
			// 如果没有提供商信息，通过ID获取
			provider, err := w.service.providerRepo.GetByID(ctx, supportInfo.Support.ProviderID)
			if err != nil {
				unavailableReasons = append(unavailableReasons, fmt.Sprintf("failed to get provider details for ID %d: %s", supportInfo.Support.ProviderID, err.Error()))
				w.logger.WithFields(map[string]interface{}{
					"provider_id": supportInfo.Support.ProviderID,
					"error":       err.Error(),
				}).Warn("Failed to get provider details")
				continue
			}

			// 检查提供商是否可用
			if !provider.IsAvailable() {
				unavailableReasons = append(unavailableReasons, fmt.Sprintf("provider %s is not available (status: %s, health: %s)",
					provider.Name, provider.Status, provider.HealthStatus))
				continue
			}
			if provider.APIKeyEncrypted == nil {
				unavailableReasons = append(unavailableReasons, fmt.Sprintf("provider %s has no API key configured", provider.Name))
				continue
			}
			return provider, nil
		}
	}

	// 构造详细的错误信息
	errorMsg := fmt.Sprintf("no available providers for model %s. Reasons: %s", modelSlug, strings.Join(unavailableReasons, "; "))
	return nil, fmt.Errorf(errorMsg)
}

