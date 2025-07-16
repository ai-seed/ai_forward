package entities

import (
	"encoding/json"
	"time"
)

// MidjourneyJobStatus Midjourney任务状态枚举
type MidjourneyJobStatus string

const (
	MidjourneyJobStatusPendingQueue MidjourneyJobStatus = "PENDING_QUEUE" // 等待队列
	MidjourneyJobStatusOnQueue      MidjourneyJobStatus = "ON_QUEUE"      // 队列中
	MidjourneyJobStatusSuccess      MidjourneyJobStatus = "SUCCESS"       // 成功
	MidjourneyJobStatusFailed       MidjourneyJobStatus = "FAILED"        // 失败
)

// MidjourneyJobAction Midjourney任务动作类型
type MidjourneyJobAction string

const (
	MidjourneyJobActionImagine    MidjourneyJobAction = "imagine"    // 图像生成
	MidjourneyJobActionAction     MidjourneyJobAction = "action"     // 操作按钮
	MidjourneyJobActionBlend      MidjourneyJobAction = "blend"      // 图像混合
	MidjourneyJobActionDescribe   MidjourneyJobAction = "describe"   // 图像描述
	MidjourneyJobActionInpaint    MidjourneyJobAction = "inpaint"    // 图像修复
	MidjourneyJobActionSeed       MidjourneyJobAction = "seed"       // 获取种子
)

// MidjourneyJobMode Midjourney任务模式
type MidjourneyJobMode string

const (
	MidjourneyJobModeFast  MidjourneyJobMode = "fast"  // 快速模式
	MidjourneyJobModeRelax MidjourneyJobMode = "relax" // 放松模式
	MidjourneyJobModeTurbo MidjourneyJobMode = "turbo" // 极速模式
)

// MidjourneyJob Midjourney任务实体
type MidjourneyJob struct {
	ID           int64                   `json:"id" gorm:"primaryKey;autoIncrement"`
	JobID        string                  `json:"job_id" gorm:"uniqueIndex;not null;size:100"` // 任务唯一标识
	UserID       int64                   `json:"user_id" gorm:"not null;index"`               // 用户ID
	APIKeyID     int64                   `json:"api_key_id" gorm:"not null;index"`            // API密钥ID
	Action       MidjourneyJobAction     `json:"action" gorm:"not null;size:20;index"`        // 任务动作
	Status       MidjourneyJobStatus     `json:"status" gorm:"not null;size:20;index"`        // 任务状态
	Mode         MidjourneyJobMode       `json:"mode" gorm:"not null;size:10;default:fast"`   // 任务模式
	Progress     int                     `json:"progress" gorm:"not null;default:0"`          // 进度 0-100
	Prompt       *string                 `json:"prompt,omitempty" gorm:"type:text"`           // 提示词
	HookURL      *string                 `json:"hook_url,omitempty" gorm:"size:500"`          // 回调URL
	Timeout      int                     `json:"timeout" gorm:"not null;default:300"`        // 超时时间（秒）
	GetUImages   bool                    `json:"get_u_images" gorm:"not null;default:false"` // 是否获取四张小图
	ParentJobID  *string                 `json:"parent_job_id,omitempty" gorm:"size:100"`    // 父任务ID（用于action操作）
	
	// 请求参数（JSON格式存储）
	RequestParams *string `json:"request_params,omitempty" gorm:"type:text"` // 请求参数
	
	// 结果数据
	DiscordImage *string `json:"discord_image,omitempty" gorm:"size:1000"`  // Discord图片URL
	CDNImage     *string `json:"cdn_image,omitempty" gorm:"size:1000"`      // CDN图片URL
	Width        *int    `json:"width,omitempty"`                           // 图片宽度
	Height       *int    `json:"height,omitempty"`                          // 图片高度
	Seed         *string `json:"seed,omitempty" gorm:"size:100"`            // 种子值
	Images       *string `json:"images,omitempty" gorm:"type:text"`         // 四张小图URL列表（JSON格式）
	Components   *string `json:"components,omitempty" gorm:"type:text"`     // 可用操作按钮（JSON格式）
	
	// 错误信息
	ErrorMessage *string `json:"error_message,omitempty" gorm:"type:text"` // 错误信息
	
	// 时间戳
	CreatedAt time.Time `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null;autoUpdateTime"`
	StartedAt *time.Time `json:"started_at,omitempty"`   // 开始处理时间
	CompletedAt *time.Time `json:"completed_at,omitempty"` // 完成时间
}

// TableName 指定表名
func (MidjourneyJob) TableName() string {
	return "midjourney_jobs"
}

// GetRequestParams 获取请求参数
func (j *MidjourneyJob) GetRequestParams() (map[string]interface{}, error) {
	if j.RequestParams == nil {
		return make(map[string]interface{}), nil
	}
	
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(*j.RequestParams), &params); err != nil {
		return nil, err
	}
	return params, nil
}

// SetRequestParams 设置请求参数
func (j *MidjourneyJob) SetRequestParams(params map[string]interface{}) error {
	if params == nil {
		j.RequestParams = nil
		return nil
	}
	
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	
	paramsStr := string(data)
	j.RequestParams = &paramsStr
	return nil
}

// GetImages 获取图片URL列表
func (j *MidjourneyJob) GetImages() ([]string, error) {
	if j.Images == nil {
		return []string{}, nil
	}
	
	var images []string
	if err := json.Unmarshal([]byte(*j.Images), &images); err != nil {
		return nil, err
	}
	return images, nil
}

// SetImages 设置图片URL列表
func (j *MidjourneyJob) SetImages(images []string) error {
	if images == nil {
		j.Images = nil
		return nil
	}
	
	data, err := json.Marshal(images)
	if err != nil {
		return err
	}
	
	imagesStr := string(data)
	j.Images = &imagesStr
	return nil
}

// GetComponents 获取可用操作按钮列表
func (j *MidjourneyJob) GetComponents() ([]string, error) {
	if j.Components == nil {
		return []string{}, nil
	}
	
	var components []string
	if err := json.Unmarshal([]byte(*j.Components), &components); err != nil {
		return nil, err
	}
	return components, nil
}

// SetComponents 设置可用操作按钮列表
func (j *MidjourneyJob) SetComponents(components []string) error {
	if components == nil {
		j.Components = nil
		return nil
	}
	
	data, err := json.Marshal(components)
	if err != nil {
		return err
	}
	
	componentsStr := string(data)
	j.Components = &componentsStr
	return nil
}

// IsCompleted 检查任务是否已完成
func (j *MidjourneyJob) IsCompleted() bool {
	return j.Status == MidjourneyJobStatusSuccess || j.Status == MidjourneyJobStatusFailed
}

// IsSuccess 检查任务是否成功
func (j *MidjourneyJob) IsSuccess() bool {
	return j.Status == MidjourneyJobStatusSuccess
}

// IsFailed 检查任务是否失败
func (j *MidjourneyJob) IsFailed() bool {
	return j.Status == MidjourneyJobStatusFailed
}

// IsProcessing 检查任务是否正在处理中
func (j *MidjourneyJob) IsProcessing() bool {
	return j.Status == MidjourneyJobStatusPendingQueue || j.Status == MidjourneyJobStatusOnQueue
}

// GetDefaultTimeout 根据模式获取默认超时时间
func GetDefaultTimeout(mode MidjourneyJobMode) int {
	switch mode {
	case MidjourneyJobModeFast:
		return 300 // 5分钟
	case MidjourneyJobModeTurbo:
		return 300 // 5分钟
	case MidjourneyJobModeRelax:
		return 600 // 10分钟
	default:
		return 300
	}
}

// GetDefaultComponents 获取imagine任务的默认操作按钮
func GetDefaultComponents() []string {
	return []string{
		"upsample1", "upsample2", "upsample3", "upsample4",
		"variation1", "variation2", "variation3", "variation4",
	}
}
