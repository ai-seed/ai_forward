package services

import (
	"context"
	"fmt"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/logger"
)

// MidjourneyService Midjourney服务接口
type MidjourneyService interface {
	// SubmitJob 提交任务
	SubmitJob(ctx context.Context, job *entities.MidjourneyJob) error

	// GetJob 获取任务
	GetJob(ctx context.Context, jobID string) (*entities.MidjourneyJob, error)

	// GetUserJobs 获取用户任务列表
	GetUserJobs(ctx context.Context, userID int64, limit, offset int) ([]*entities.MidjourneyJob, error)

	// ProcessPendingJobs 处理待处理任务
	ProcessPendingJobs(ctx context.Context) error

	// UpdateJobStatus 更新任务状态
	UpdateJobStatus(ctx context.Context, jobID string, status entities.MidjourneyJobStatus) error

	// UpdateJobProgress 更新任务进度
	UpdateJobProgress(ctx context.Context, jobID string, progress int) error

	// UpdateJobResult 更新任务结果
	UpdateJobResult(ctx context.Context, jobID string, result *repositories.MidjourneyJobResult) error

	// GetJobStats 获取任务统计
	GetJobStats(ctx context.Context, userID *int64) (*repositories.MidjourneyJobStats, error)

	// CleanupExpiredJobs 清理超时任务
	CleanupExpiredJobs(ctx context.Context) error
}

// midjourneyServiceImpl Midjourney服务实现
type midjourneyServiceImpl struct {
	jobRepo      repositories.MidjourneyJobRepository
	queueService MidjourneyQueueService
	logger       logger.Logger
}

// NewMidjourneyService 创建Midjourney服务
func NewMidjourneyService(
	jobRepo repositories.MidjourneyJobRepository,
	queueService MidjourneyQueueService,
	logger logger.Logger,
) MidjourneyService {
	return &midjourneyServiceImpl{
		jobRepo:      jobRepo,
		queueService: queueService,
		logger:       logger,
	}
}

// SubmitJob 提交任务
func (s *midjourneyServiceImpl) SubmitJob(ctx context.Context, job *entities.MidjourneyJob) error {
	// 验证任务参数
	if err := s.validateJob(job); err != nil {
		return fmt.Errorf("invalid job: %w", err)
	}

	// 设置创建时间
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	// 创建任务
	if err := s.jobRepo.Create(ctx, job); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": job.JobID,
		}).Error("Failed to create job")
		return fmt.Errorf("failed to create job: %w", err)
	}

	// 将任务加入队列
	if s.queueService != nil {
		if err := s.queueService.EnqueueJob(ctx, job); err != nil {
			s.logger.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"job_id": job.JobID,
			}).Error("Failed to enqueue job")
			// 不返回错误，任务已经创建，可以通过其他方式处理
		}
	}

	s.logger.WithFields(map[string]interface{}{
		"job_id":  job.JobID,
		"user_id": job.UserID,
		"action":  job.Action,
	}).Info("Job submitted successfully")

	return nil
}

// GetJob 获取任务
func (s *midjourneyServiceImpl) GetJob(ctx context.Context, jobID string) (*entities.MidjourneyJob, error) {
	job, err := s.jobRepo.GetByJobID(ctx, jobID)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": jobID,
		}).Error("Failed to get job")
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	return job, nil
}

// GetUserJobs 获取用户任务列表
func (s *midjourneyServiceImpl) GetUserJobs(ctx context.Context, userID int64, limit, offset int) ([]*entities.MidjourneyJob, error) {
	jobs, err := s.jobRepo.GetUserJobs(ctx, userID, limit, offset)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		}).Error("Failed to get user jobs")
		return nil, fmt.Errorf("failed to get user jobs: %w", err)
	}
	return jobs, nil
}

// ProcessPendingJobs 处理待处理任务
func (s *midjourneyServiceImpl) ProcessPendingJobs(ctx context.Context) error {
	// 如果有队列服务，委托给队列服务处理
	if s.queueService != nil {
		return nil // 队列服务会自动处理
	}

	// 获取待处理任务
	jobs, err := s.jobRepo.GetPendingJobs(ctx, 10) // 一次处理10个任务
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to get pending jobs")
		return fmt.Errorf("failed to get pending jobs: %w", err)
	}

	if len(jobs) == 0 {
		return nil
	}

	s.logger.WithFields(map[string]interface{}{
		"count": len(jobs),
	}).Info("Processing pending jobs")

	for _, job := range jobs {
		if err := s.processJob(ctx, job); err != nil {
			s.logger.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"job_id": job.JobID,
			}).Error("Failed to process job")
			// 标记任务为失败
			s.UpdateJobStatus(ctx, job.JobID, entities.MidjourneyJobStatusFailed)
		}
	}

	return nil
}

// UpdateJobStatus 更新任务状态
func (s *midjourneyServiceImpl) UpdateJobStatus(ctx context.Context, jobID string, status entities.MidjourneyJobStatus) error {
	if err := s.jobRepo.UpdateStatus(ctx, jobID, status); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": jobID,
			"status": status,
		}).Error("Failed to update job status")
		return fmt.Errorf("failed to update job status: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"job_id": jobID,
		"status": status,
	}).Info("Job status updated")

	return nil
}

// UpdateJobProgress 更新任务进度
func (s *midjourneyServiceImpl) UpdateJobProgress(ctx context.Context, jobID string, progress int) error {
	if err := s.jobRepo.UpdateProgress(ctx, jobID, progress); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":    err.Error(),
			"job_id":   jobID,
			"progress": progress,
		}).Error("Failed to update job progress")
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}

// UpdateJobResult 更新任务结果
func (s *midjourneyServiceImpl) UpdateJobResult(ctx context.Context, jobID string, result *repositories.MidjourneyJobResult) error {
	if err := s.jobRepo.UpdateResult(ctx, jobID, result); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": jobID,
		}).Error("Failed to update job result")
		return fmt.Errorf("failed to update job result: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"job_id": jobID,
	}).Info("Job result updated")
	return nil
}

// GetJobStats 获取任务统计
func (s *midjourneyServiceImpl) GetJobStats(ctx context.Context, userID *int64) (*repositories.MidjourneyJobStats, error) {
	stats, err := s.jobRepo.GetJobStats(ctx, userID)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to get job stats")
		return nil, fmt.Errorf("failed to get job stats: %w", err)
	}
	return stats, nil
}

// CleanupExpiredJobs 清理超时任务
func (s *midjourneyServiceImpl) CleanupExpiredJobs(ctx context.Context) error {
	// 如果有队列服务，委托给队列服务处理
	if s.queueService != nil {
		return s.queueService.ProcessExpiredJobs(ctx)
	}

	jobs, err := s.jobRepo.GetExpiredJobs(ctx)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to get expired jobs")
		return fmt.Errorf("failed to get expired jobs: %w", err)
	}

	if len(jobs) == 0 {
		return nil
	}

	s.logger.WithFields(map[string]interface{}{
		"count": len(jobs),
	}).Info("Cleaning up expired jobs")

	for _, job := range jobs {
		// 标记为失败
		if err := s.UpdateJobStatus(ctx, job.JobID, entities.MidjourneyJobStatusFailed); err != nil {
			s.logger.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"job_id": job.JobID,
			}).Error("Failed to mark expired job as failed")
		}

		// 设置错误信息
		result := &repositories.MidjourneyJobResult{
			ErrorMessage: stringPtr("Job timeout"),
		}
		if err := s.UpdateJobResult(ctx, job.JobID, result); err != nil {
			s.logger.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"job_id": job.JobID,
			}).Error("Failed to set timeout error message")
		}
	}

	return nil
}

// validateJob 验证任务参数
func (s *midjourneyServiceImpl) validateJob(job *entities.MidjourneyJob) error {
	if job.JobID == "" {
		return fmt.Errorf("job_id is required")
	}

	if job.UserID == 0 {
		return fmt.Errorf("user_id is required")
	}

	if job.APIKeyID == 0 {
		return fmt.Errorf("api_key_id is required")
	}

	if job.Action == "" {
		return fmt.Errorf("action is required")
	}

	// 验证特定动作的参数
	switch job.Action {
	case entities.MidjourneyJobActionImagine:
		if job.Prompt == nil || *job.Prompt == "" {
			return fmt.Errorf("prompt is required for imagine action")
		}
	case entities.MidjourneyJobActionAction:
		if job.ParentJobID == nil || *job.ParentJobID == "" {
			return fmt.Errorf("parent_job_id is required for action")
		}
	}

	return nil
}

// processJob 处理单个任务（简化版本）
func (s *midjourneyServiceImpl) processJob(ctx context.Context, job *entities.MidjourneyJob) error {
	// 更新状态为处理中
	if err := s.UpdateJobStatus(ctx, job.JobID, entities.MidjourneyJobStatusOnQueue); err != nil {
		return err
	}

	// 这里应该调用实际的图像生成服务
	// 目前先模拟处理过程
	s.logger.WithFields(map[string]interface{}{
		"job_id": job.JobID,
	}).Info("Processing job (simulated)")

	// 模拟处理时间
	time.Sleep(2 * time.Second)

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
	if err := s.UpdateJobResult(ctx, job.JobID, result); err != nil {
		return err
	}

	// 更新进度为100%
	if err := s.UpdateJobProgress(ctx, job.JobID, 100); err != nil {
		return err
	}

	// 更新状态为成功
	if err := s.UpdateJobStatus(ctx, job.JobID, entities.MidjourneyJobStatusSuccess); err != nil {
		return err
	}

	s.logger.WithFields(map[string]interface{}{
		"job_id": job.JobID,
	}).Info("Job processed successfully")
	return nil
}
