package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
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
	jobRepo         repositories.MidjourneyJobRepository
	cache           *redis.CacheService
	logger          logger.Logger
	workers         []*worker
	workerWg        sync.WaitGroup
	stopCh          chan struct{}
	mu              sync.RWMutex
	isRunning       bool
	webhookService  WebhookService
	imageGenService ImageGenerationService
}

// worker 工作进程
type worker struct {
	id      int
	service *midjourneyQueueServiceImpl
	stopCh  chan struct{}
	logger  logger.Logger
}

// NewMidjourneyQueueService 创建队列服务
func NewMidjourneyQueueService(
	jobRepo repositories.MidjourneyJobRepository,
	cache *redis.CacheService,
	webhookService WebhookService,
	imageGenService ImageGenerationService,
	logger logger.Logger,
) MidjourneyQueueService {
	return &midjourneyQueueServiceImpl{
		jobRepo:         jobRepo,
		cache:           cache,
		logger:          logger,
		stopCh:          make(chan struct{}),
		webhookService:  webhookService,
		imageGenService: imageGenService,
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
			stopCh:  make(chan struct{}),
			logger:  s.logger.WithField("worker_id", i+1),
		}
		s.workers[i] = worker

		s.workerWg.Add(1)
		go worker.run(ctx)
	}

	s.isRunning = true
	s.logger.WithField("worker_count", workerCount).Info("Midjourney queue workers started")

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

	// 停止所有工作进程
	for _, worker := range s.workers {
		close(worker.stopCh)
	}

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

	// 如果使用Redis，可以将任务ID加入队列
	// 暂时跳过Redis队列实现，直接使用数据库轮询
	_ = s.cache // 避免未使用变量警告

	s.logger.WithFields(map[string]interface{}{
		"job_id": job.JobID,
		"action": job.Action,
		"mode":   job.Mode,
	}).Info("Job enqueued successfully")

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

	w.logger.Info("Worker started")
	defer w.logger.Info("Worker stopped")

	ticker := time.NewTicker(5 * time.Second) // 每5秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-w.service.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processNextJob(ctx); err != nil {
				w.logger.WithFields(map[string]interface{}{
					"error": err.Error(),
				}).Error("Failed to process job")
			}
		}
	}
}

// processNextJob 处理下一个任务
func (w *worker) processNextJob(ctx context.Context) error {
	// 获取待处理任务
	jobs, err := w.service.jobRepo.GetPendingJobs(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to get pending jobs: %w", err)
	}

	if len(jobs) == 0 {
		return nil // 没有待处理任务
	}

	job := jobs[0]

	w.logger.WithFields(map[string]interface{}{
		"job_id": job.JobID,
		"action": job.Action,
	}).Info("Processing job")

	// 更新状态为处理中
	if err := w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusOnQueue); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
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

		return nil // 不返回错误，继续处理下一个任务
	}

	return nil
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
	// 这里应该调用实际的图像生成服务
	// 目前先模拟处理

	// 更新进度
	for progress := 10; progress <= 100; progress += 10 {
		if err := w.service.jobRepo.UpdateProgress(ctx, job.JobID, progress); err != nil {
			w.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("Failed to update progress")
		}

		// 发送进度webhook
		if job.HookURL != nil && *job.HookURL != "" && progress < 100 {
			w.service.sendWebhook(ctx, job, "IN_PROGRESS", fmt.Sprintf("Progress: %d%%", progress))
		}

		time.Sleep(2 * time.Second) // 模拟处理时间
	}

	// 模拟成功结果
	result := &repositories.MidjourneyJobResult{
		DiscordImage: stringPtr("https://cdn.discordapp.com/attachments/example.png"),
		CDNImage:     stringPtr("https://cdn.example.com/example.png"),
		Width:        intPtr(1024),
		Height:       intPtr(1024),
		Images: []string{
			"https://cdn.example.com/image1.png",
			"https://cdn.example.com/image2.png",
			"https://cdn.example.com/image3.png",
			"https://cdn.example.com/image4.png",
		},
		Components: entities.GetDefaultComponents(),
	}

	// 更新结果
	if err := w.service.jobRepo.UpdateResult(ctx, job.JobID, result); err != nil {
		return fmt.Errorf("failed to update job result: %w", err)
	}

	// 更新状态为成功
	if err := w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusSuccess); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// 发送成功webhook
	if job.HookURL != nil && *job.HookURL != "" {
		w.service.sendWebhook(ctx, job, "SUCCESS", "Job completed successfully")
	}

	w.logger.WithField("job_id", job.JobID).Info("Imagine job completed successfully")
	return nil
}

// processActionJob 处理操作任务
func (w *worker) processActionJob(ctx context.Context, job *entities.MidjourneyJob) error {
	// 模拟操作处理
	time.Sleep(3 * time.Second)

	// 更新进度为100%
	if err := w.service.jobRepo.UpdateProgress(ctx, job.JobID, 100); err != nil {
		return fmt.Errorf("failed to update progress: %w", err)
	}

	// 模拟结果
	result := &repositories.MidjourneyJobResult{
		DiscordImage: stringPtr("https://cdn.discordapp.com/attachments/action_result.png"),
		CDNImage:     stringPtr("https://cdn.example.com/action_result.png"),
		Width:        intPtr(2048),
		Height:       intPtr(2048),
	}

	// 更新结果和状态
	if err := w.service.jobRepo.UpdateResult(ctx, job.JobID, result); err != nil {
		return fmt.Errorf("failed to update job result: %w", err)
	}

	if err := w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusSuccess); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// 发送webhook
	if job.HookURL != nil && *job.HookURL != "" {
		w.service.sendWebhook(ctx, job, "SUCCESS", "Action completed successfully")
	}

	w.logger.WithField("job_id", job.JobID).Info("Action job completed successfully")
	return nil
}

// processBlendJob 处理混合任务
func (w *worker) processBlendJob(ctx context.Context, job *entities.MidjourneyJob) error {
	// 模拟混合处理
	time.Sleep(5 * time.Second)

	if err := w.service.jobRepo.UpdateProgress(ctx, job.JobID, 100); err != nil {
		return fmt.Errorf("failed to update progress: %w", err)
	}

	result := &repositories.MidjourneyJobResult{
		DiscordImage: stringPtr("https://cdn.discordapp.com/attachments/blend_result.png"),
		CDNImage:     stringPtr("https://cdn.example.com/blend_result.png"),
		Width:        intPtr(1024),
		Height:       intPtr(1024),
	}

	if err := w.service.jobRepo.UpdateResult(ctx, job.JobID, result); err != nil {
		return fmt.Errorf("failed to update job result: %w", err)
	}

	if err := w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusSuccess); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	if job.HookURL != nil && *job.HookURL != "" {
		w.service.sendWebhook(ctx, job, "SUCCESS", "Blend completed successfully")
	}

	w.logger.WithField("job_id", job.JobID).Info("Blend job completed successfully")
	return nil
}

// processDescribeJob 处理描述任务
func (w *worker) processDescribeJob(ctx context.Context, job *entities.MidjourneyJob) error {
	// 模拟描述处理
	time.Sleep(3 * time.Second)

	if err := w.service.jobRepo.UpdateProgress(ctx, job.JobID, 100); err != nil {
		return fmt.Errorf("failed to update progress: %w", err)
	}

	// 模拟描述结果
	description := "A beautiful landscape with mountains and trees"
	result := &repositories.MidjourneyJobResult{
		ErrorMessage: &description, // 临时使用这个字段存储描述结果
	}

	if err := w.service.jobRepo.UpdateResult(ctx, job.JobID, result); err != nil {
		return fmt.Errorf("failed to update job result: %w", err)
	}

	if err := w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusSuccess); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	if job.HookURL != nil && *job.HookURL != "" {
		w.service.sendWebhook(ctx, job, "SUCCESS", "Describe completed successfully")
	}

	w.logger.WithField("job_id", job.JobID).Info("Describe job completed successfully")
	return nil
}

// processInpaintJob 处理修复任务
func (w *worker) processInpaintJob(ctx context.Context, job *entities.MidjourneyJob) error {
	// 模拟修复处理
	time.Sleep(8 * time.Second)

	if err := w.service.jobRepo.UpdateProgress(ctx, job.JobID, 100); err != nil {
		return fmt.Errorf("failed to update progress: %w", err)
	}

	result := &repositories.MidjourneyJobResult{
		DiscordImage: stringPtr("https://cdn.discordapp.com/attachments/inpaint_result.png"),
		CDNImage:     stringPtr("https://cdn.example.com/inpaint_result.png"),
		Width:        intPtr(1024),
		Height:       intPtr(1024),
	}

	if err := w.service.jobRepo.UpdateResult(ctx, job.JobID, result); err != nil {
		return fmt.Errorf("failed to update job result: %w", err)
	}

	if err := w.service.jobRepo.UpdateStatus(ctx, job.JobID, entities.MidjourneyJobStatusSuccess); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	if job.HookURL != nil && *job.HookURL != "" {
		w.service.sendWebhook(ctx, job, "SUCCESS", "Inpaint completed successfully")
	}

	w.logger.WithField("job_id", job.JobID).Info("Inpaint job completed successfully")
	return nil
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

func intPtr(i int) *int {
	return &i
}
