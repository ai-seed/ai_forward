package repositories

import (
	"context"
	"fmt"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/redis"

	"gorm.io/gorm"
)

// midjourneyJobRepositoryGorm GORM Midjourney任务仓储实现
type midjourneyJobRepositoryGorm struct {
	db    *gorm.DB
	cache *redis.CacheService
}

// NewMidjourneyJobRepositoryGorm 创建GORM Midjourney任务仓储
func NewMidjourneyJobRepositoryGorm(db *gorm.DB, cache *redis.CacheService) repositories.MidjourneyJobRepository {
	return &midjourneyJobRepositoryGorm{
		db:    db,
		cache: cache,
	}
}

// Create 创建任务
func (r *midjourneyJobRepositoryGorm) Create(ctx context.Context, job *entities.MidjourneyJob) error {
	if err := r.db.WithContext(ctx).Create(job).Error; err != nil {
		return fmt.Errorf("failed to create midjourney job: %w", err)
	}

	// 清除相关缓存
	if r.cache != nil {
		r.clearJobCache(ctx, job.JobID, job.UserID)
	}

	return nil
}

// GetByJobID 根据任务ID获取任务
func (r *midjourneyJobRepositoryGorm) GetByJobID(ctx context.Context, jobID string) (*entities.MidjourneyJob, error) {
	// 尝试从缓存获取
	if r.cache != nil {
		cacheKey := fmt.Sprintf("midjourney_job:%s", jobID)
		var cachedJob entities.MidjourneyJob
		if err := r.cache.Get(ctx, cacheKey, &cachedJob); err == nil {
			return &cachedJob, nil
		}
	}

	var job entities.MidjourneyJob
	if err := r.db.WithContext(ctx).Where("job_id = ?", jobID).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get midjourney job by job_id: %w", err)
	}

	// 缓存结果
	if r.cache != nil {
		cacheKey := fmt.Sprintf("midjourney_job:%s", jobID)
		ttl := 10 * time.Minute
		r.cache.Set(ctx, cacheKey, &job, ttl)
	}

	return &job, nil
}

// GetByID 根据主键ID获取任务
func (r *midjourneyJobRepositoryGorm) GetByID(ctx context.Context, id int64) (*entities.MidjourneyJob, error) {
	var job entities.MidjourneyJob
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get midjourney job by id: %w", err)
	}
	return &job, nil
}

// Update 更新任务
func (r *midjourneyJobRepositoryGorm) Update(ctx context.Context, job *entities.MidjourneyJob) error {
	if err := r.db.WithContext(ctx).Save(job).Error; err != nil {
		return fmt.Errorf("failed to update midjourney job: %w", err)
	}

	// 清除缓存
	if r.cache != nil {
		r.clearJobCache(ctx, job.JobID, job.UserID)
	}

	return nil
}

// UpdateStatus 更新任务状态
func (r *midjourneyJobRepositoryGorm) UpdateStatus(ctx context.Context, jobID string, status entities.MidjourneyJobStatus) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	// 如果状态变为处理中，设置开始时间
	if status == entities.MidjourneyJobStatusOnQueue {
		updates["started_at"] = time.Now()
	}

	// 如果状态变为完成或失败，设置完成时间
	if status == entities.MidjourneyJobStatusSuccess || status == entities.MidjourneyJobStatusFailed {
		updates["completed_at"] = time.Now()
	}

	if err := r.db.WithContext(ctx).Model(&entities.MidjourneyJob{}).
		Where("job_id = ?", jobID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update midjourney job status: %w", err)
	}

	// 清除缓存
	if r.cache != nil {
		cacheKey := fmt.Sprintf("midjourney_job:%s", jobID)
		r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// UpdateProgress 更新任务进度
func (r *midjourneyJobRepositoryGorm) UpdateProgress(ctx context.Context, jobID string, progress int) error {
	if err := r.db.WithContext(ctx).Model(&entities.MidjourneyJob{}).
		Where("job_id = ?", jobID).
		Updates(map[string]interface{}{
			"progress":   progress,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("failed to update midjourney job progress: %w", err)
	}

	// 清除缓存
	if r.cache != nil {
		cacheKey := fmt.Sprintf("midjourney_job:%s", jobID)
		r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// UpdateUpstreamTaskID 更新上游任务ID
func (r *midjourneyJobRepositoryGorm) UpdateUpstreamTaskID(ctx context.Context, jobID string, upstreamTaskID string) error {
	if err := r.db.WithContext(ctx).Model(&entities.MidjourneyJob{}).
		Where("job_id = ?", jobID).
		Updates(map[string]interface{}{
			"upstream_task_id": upstreamTaskID,
			"updated_at":       time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("failed to update midjourney job upstream task ID: %w", err)
	}

	// 清除缓存
	if r.cache != nil {
		cacheKey := fmt.Sprintf("midjourney_job:%s", jobID)
		r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// UpdateResult 更新任务结果
func (r *midjourneyJobRepositoryGorm) UpdateResult(ctx context.Context, jobID string, result *repositories.MidjourneyJobResult) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if result.DiscordImage != nil {
		updates["discord_image"] = *result.DiscordImage
	}
	if result.CDNImage != nil {
		updates["cdn_image"] = *result.CDNImage
	}
	if result.Width != nil {
		updates["width"] = *result.Width
	}
	if result.Height != nil {
		updates["height"] = *result.Height
	}
	if result.Seed != nil {
		updates["seed"] = *result.Seed
	}
	if result.ErrorMessage != nil {
		updates["error_message"] = *result.ErrorMessage
	}

	// 处理图片列表
	if result.Images != nil {
		job := &entities.MidjourneyJob{}
		if err := job.SetImages(result.Images); err != nil {
			return fmt.Errorf("failed to set images: %w", err)
		}
		updates["images"] = job.Images
	}

	// 处理操作按钮列表
	if result.Components != nil {
		job := &entities.MidjourneyJob{}
		if err := job.SetComponents(result.Components); err != nil {
			return fmt.Errorf("failed to set components: %w", err)
		}
		updates["components"] = job.Components
	}

	if err := r.db.WithContext(ctx).Model(&entities.MidjourneyJob{}).
		Where("job_id = ?", jobID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update midjourney job result: %w", err)
	}

	// 清除缓存
	if r.cache != nil {
		cacheKey := fmt.Sprintf("midjourney_job:%s", jobID)
		r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// GetUserJobs 获取用户的任务列表
func (r *midjourneyJobRepositoryGorm) GetUserJobs(ctx context.Context, userID int64, limit, offset int) ([]*entities.MidjourneyJob, error) {
	var jobs []*entities.MidjourneyJob
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to get user midjourney jobs: %w", err)
	}
	return jobs, nil
}

// GetPendingJobs 获取待处理的任务列表
func (r *midjourneyJobRepositoryGorm) GetPendingJobs(ctx context.Context, limit int) ([]*entities.MidjourneyJob, error) {
	var jobs []*entities.MidjourneyJob
	if err := r.db.WithContext(ctx).
		Where("status = ?", entities.MidjourneyJobStatusPendingQueue).
		Order("created_at ASC").
		Limit(limit).
		Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending midjourney jobs: %w", err)
	}
	return jobs, nil
}

// GetProcessingJobs 获取正在处理的任务列表
func (r *midjourneyJobRepositoryGorm) GetProcessingJobs(ctx context.Context) ([]*entities.MidjourneyJob, error) {
	var jobs []*entities.MidjourneyJob
	if err := r.db.WithContext(ctx).
		Where("status = ?", entities.MidjourneyJobStatusOnQueue).
		Order("started_at ASC").
		Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to get processing midjourney jobs: %w", err)
	}
	return jobs, nil
}

// GetExpiredJobs 获取超时的任务列表
func (r *midjourneyJobRepositoryGorm) GetExpiredJobs(ctx context.Context) ([]*entities.MidjourneyJob, error) {
	var jobs []*entities.MidjourneyJob

	// 查找超时的任务：状态为处理中且开始时间+超时时间 < 当前时间
	if err := r.db.WithContext(ctx).
		Where("status IN (?, ?) AND created_at + INTERVAL timeout SECOND < ?",
			entities.MidjourneyJobStatusPendingQueue,
			entities.MidjourneyJobStatusOnQueue,
			time.Now()).
		Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to get expired midjourney jobs: %w", err)
	}
	return jobs, nil
}

// Delete 删除任务
func (r *midjourneyJobRepositoryGorm) Delete(ctx context.Context, jobID string) error {
	if err := r.db.WithContext(ctx).Where("job_id = ?", jobID).Delete(&entities.MidjourneyJob{}).Error; err != nil {
		return fmt.Errorf("failed to delete midjourney job: %w", err)
	}

	// 清除缓存
	if r.cache != nil {
		cacheKey := fmt.Sprintf("midjourney_job:%s", jobID)
		r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// GetJobStats 获取任务统计信息
func (r *midjourneyJobRepositoryGorm) GetJobStats(ctx context.Context, userID *int64) (*repositories.MidjourneyJobStats, error) {
	var stats repositories.MidjourneyJobStats

	query := r.db.WithContext(ctx).Model(&entities.MidjourneyJob{})
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	// 获取总任务数
	if err := query.Count(&stats.TotalJobs).Error; err != nil {
		return nil, fmt.Errorf("failed to count total jobs: %w", err)
	}

	// 获取各状态任务数
	statusCounts := make(map[entities.MidjourneyJobStatus]int64)
	rows, err := query.Select("status, COUNT(*) as count").Group("status").Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status entities.MidjourneyJobStatus
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}
		statusCounts[status] = count
	}

	stats.PendingJobs = statusCounts[entities.MidjourneyJobStatusPendingQueue]
	stats.ProcessingJobs = statusCounts[entities.MidjourneyJobStatusOnQueue]
	stats.SuccessJobs = statusCounts[entities.MidjourneyJobStatusSuccess]
	stats.FailedJobs = statusCounts[entities.MidjourneyJobStatusFailed]

	// 计算成功率
	if stats.TotalJobs > 0 {
		completedJobs := stats.SuccessJobs + stats.FailedJobs
		if completedJobs > 0 {
			stats.SuccessRate = float64(stats.SuccessJobs) / float64(completedJobs) * 100
		}
	}

	return &stats, nil
}

// clearJobCache 清除任务相关缓存
func (r *midjourneyJobRepositoryGorm) clearJobCache(ctx context.Context, jobID string, userID int64) {
	if r.cache == nil {
		return
	}

	// 清除任务缓存
	cacheKey := fmt.Sprintf("midjourney_job:%s", jobID)
	r.cache.Delete(ctx, cacheKey)

	// 清除用户任务列表缓存（如果有的话）
	userCacheKey := fmt.Sprintf("user_midjourney_jobs:%d", userID)
	r.cache.Delete(ctx, userCacheKey)
}
