package repositories

import (
	"ai-api-gateway/internal/domain/entities"
	"context"
)

// MidjourneyJobRepository Midjourney任务仓储接口
type MidjourneyJobRepository interface {
	// Create 创建任务
	Create(ctx context.Context, job *entities.MidjourneyJob) error

	// GetByJobID 根据任务ID获取任务
	GetByJobID(ctx context.Context, jobID string) (*entities.MidjourneyJob, error)

	// GetByID 根据主键ID获取任务
	GetByID(ctx context.Context, id int64) (*entities.MidjourneyJob, error)

	// Update 更新任务
	Update(ctx context.Context, job *entities.MidjourneyJob) error

	// UpdateStatus 更新任务状态
	UpdateStatus(ctx context.Context, jobID string, status entities.MidjourneyJobStatus) error

	// UpdateProgress 更新任务进度
	UpdateProgress(ctx context.Context, jobID string, progress int) error

	// UpdateUpstreamTaskID 更新上游任务ID
	UpdateUpstreamTaskID(ctx context.Context, jobID string, upstreamTaskID string) error

	// UpdateResult 更新任务结果
	UpdateResult(ctx context.Context, jobID string, result *MidjourneyJobResult) error

	// GetUserJobs 获取用户的任务列表
	GetUserJobs(ctx context.Context, userID int64, limit, offset int) ([]*entities.MidjourneyJob, error)

	// GetPendingJobs 获取待处理的任务列表
	GetPendingJobs(ctx context.Context, limit int) ([]*entities.MidjourneyJob, error)

	// GetProcessingJobs 获取正在处理的任务列表
	GetProcessingJobs(ctx context.Context) ([]*entities.MidjourneyJob, error)

	// GetExpiredJobs 获取超时的任务列表
	GetExpiredJobs(ctx context.Context) ([]*entities.MidjourneyJob, error)

	// Delete 删除任务
	Delete(ctx context.Context, jobID string) error

	// GetJobStats 获取任务统计信息
	GetJobStats(ctx context.Context, userID *int64) (*MidjourneyJobStats, error)
}

// MidjourneyJobResult 任务结果结构
type MidjourneyJobResult struct {
	DiscordImage *string  `json:"discord_image,omitempty"`
	CDNImage     *string  `json:"cdn_image,omitempty"`
	Width        *int     `json:"width,omitempty"`
	Height       *int     `json:"height,omitempty"`
	Seed         *string  `json:"seed,omitempty"`
	Images       []string `json:"images,omitempty"`
	Components   []string `json:"components,omitempty"`
	ErrorMessage *string  `json:"error_message,omitempty"`
}

// MidjourneyJobStats 任务统计信息
type MidjourneyJobStats struct {
	TotalJobs      int64   `json:"total_jobs"`
	PendingJobs    int64   `json:"pending_jobs"`
	ProcessingJobs int64   `json:"processing_jobs"`
	SuccessJobs    int64   `json:"success_jobs"`
	FailedJobs     int64   `json:"failed_jobs"`
	SuccessRate    float64 `json:"success_rate"`
}
