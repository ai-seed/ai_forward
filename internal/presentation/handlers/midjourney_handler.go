package handlers

import (
	"net/http"
	"strconv"

	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/infrastructure/logger"
	"ai-api-gateway/internal/presentation/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MidjourneyHandler Midjourney API处理器（302AI格式）
type MidjourneyHandler struct {
	midjourneyService services.MidjourneyService
	logger            logger.Logger
}

// NewMidjourneyHandler 创建Midjourney处理器
func NewMidjourneyHandler(midjourneyService services.MidjourneyService, logger logger.Logger) *MidjourneyHandler {
	return &MidjourneyHandler{
		midjourneyService: midjourneyService,
		logger:            logger,
	}
}

// MJResponse 302AI标准响应格式
type MJResponse struct {
	Code        int                    `json:"code"`
	Description string                 `json:"description"`
	Properties  map[string]interface{} `json:"properties"`
	Result      interface{}            `json:"result"`
}

// MJTaskResponse 任务详情响应
type MJTaskResponse struct {
	Action      string                 `json:"action"`
	Buttons     []MJButton             `json:"buttons,omitempty"`
	Description string                 `json:"description,omitempty"`
	FailReason  string                 `json:"failReason,omitempty"`
	FinishTime  *int64                 `json:"finishTime,omitempty"`
	ID          string                 `json:"id"`
	ImageURL    string                 `json:"imageUrl,omitempty"`
	Progress    string                 `json:"progress"`
	Prompt      string                 `json:"prompt,omitempty"`
	PromptEn    string                 `json:"promptEn,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	StartTime   *int64                 `json:"startTime,omitempty"`
	State       string                 `json:"state,omitempty"`
	Status      string                 `json:"status"`
	SubmitTime  int64                  `json:"submitTime"`
}

// MJButton 可执行按钮
type MJButton struct {
	CustomID string `json:"customId"`
	Emoji    string `json:"emoji,omitempty"`
	Label    string `json:"label"`
	Style    int    `json:"style,omitempty"`
	Type     int    `json:"type"`
}

// ImagineRequest 图像生成请求
type ImagineRequest struct {
	Base64Array []string `json:"base64Array,omitempty"`
	BotType     string   `json:"botType,omitempty"`
	NotifyHook  string   `json:"notifyHook,omitempty"`
	Prompt      string   `json:"prompt" binding:"required"`
	State       string   `json:"state,omitempty"`
}

// ActionRequest 操作请求
type ActionRequest struct {
	CustomID   string `json:"customId" binding:"required"`
	NotifyHook string `json:"notifyHook,omitempty"`
	State      string `json:"state,omitempty"`
	TaskID     string `json:"taskId" binding:"required"`
}

// BlendRequest 混合请求
type BlendRequest struct {
	Base64Array []string `json:"base64Array" binding:"required,min=2,max=5"`
	BotType     string   `json:"botType,omitempty"`
	Dimensions  string   `json:"dimensions,omitempty"`
	NotifyHook  string   `json:"notifyHook,omitempty"`
	State       string   `json:"state,omitempty"`
}

// DescribeRequest 描述请求
type DescribeRequest struct {
	Base64     string `json:"base64" binding:"required"`
	BotType    string `json:"botType,omitempty"`
	NotifyHook string `json:"notifyHook,omitempty"`
	State      string `json:"state,omitempty"`
}

// ModalRequest 局部重绘请求
type ModalRequest struct {
	Base64     string `json:"base64" binding:"required"`
	BotType    string `json:"botType,omitempty"`
	MaskBase64 string `json:"maskBase64" binding:"required"`
	NotifyHook string `json:"notifyHook,omitempty"`
	Prompt     string `json:"prompt" binding:"required"`
	State      string `json:"state,omitempty"`
}

// CancelRequest 取消请求
type CancelRequest struct {
	TaskID string `json:"taskId" binding:"required"`
}

// Imagine 图像生成端点
// @Summary 生成图像
// @Description 根据提示词生成图像，类似 /imagine 命令
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param request body ImagineRequest true "图像生成请求"
// @Success 200 {object} MJResponse
// @Failure 400 {object} MJResponse
// @Failure 401 {object} MJResponse
// @Failure 500 {object} MJResponse
// @Router /mj/submit/imagine [post]
func (h *MidjourneyHandler) Imagine(c *gin.Context) {
	var req ImagineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Invalid imagine request")
		c.JSON(http.StatusBadRequest, MJResponse{
			Code:        400,
			Description: "Invalid request parameters: " + err.Error(),
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 获取用户信息
	userID, hasUserID := utils.GetUserIDFromContext(c)
	apiKeyID, hasAPIKeyID := utils.GetAPIKeyIDFromContext(c)
	if !hasUserID || !hasAPIKeyID {
		c.JSON(http.StatusUnauthorized, MJResponse{
			Code:        401,
			Description: "Unauthorized",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 生成任务ID
	jobID := uuid.New().String()

	// 设置默认值
	if req.BotType == "" {
		req.BotType = "MID_JOURNEY"
	}

	// 创建任务
	job := &entities.MidjourneyJob{
		JobID:      jobID,
		UserID:     userID,
		APIKeyID:   apiKeyID,
		Action:     entities.MidjourneyJobActionImagine,
		Status:     entities.MidjourneyJobStatusPendingQueue,
		Mode:       entities.MidjourneyJobModeFast,
		Progress:   0,
		Prompt:     &req.Prompt,
		HookURL:    &req.NotifyHook,
		Timeout:    300,
		GetUImages: false,
	}

	// 设置请求参数
	params := map[string]interface{}{
		"prompt":      req.Prompt,
		"botType":     req.BotType,
		"base64Array": req.Base64Array,
		"state":       req.State,
	}
	if err := job.SetRequestParams(params); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to set request params")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Internal server error",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 提交任务
	if err := h.midjourneyService.SubmitJob(c.Request.Context(), job); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to submit imagine job")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Failed to submit job",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"job_id": jobID,
	}).Info("Imagine job submitted successfully")

	c.JSON(http.StatusOK, MJResponse{
		Code:        1,
		Description: "提交成功",
		Properties:  map[string]interface{}{},
		Result:      jobID,
	})
}

// Action 操作端点
// @Summary 执行操作
// @Description 执行 U1-U4、V1-V4 等操作
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param request body ActionRequest true "操作请求"
// @Success 200 {object} MJResponse
// @Failure 400 {object} MJResponse
// @Failure 401 {object} MJResponse
// @Failure 500 {object} MJResponse
// @Router /mj/submit/action [post]
func (h *MidjourneyHandler) Action(c *gin.Context) {
	var req ActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Invalid action request")
		c.JSON(http.StatusBadRequest, MJResponse{
			Code:        400,
			Description: "Invalid request parameters: " + err.Error(),
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 获取用户信息
	userID, hasUserID := utils.GetUserIDFromContext(c)
	apiKeyID, hasAPIKeyID := utils.GetAPIKeyIDFromContext(c)
	if !hasUserID || !hasAPIKeyID {
		c.JSON(http.StatusUnauthorized, MJResponse{
			Code:        401,
			Description: "Unauthorized",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 生成新的任务ID
	newJobID := uuid.New().String()

	// 创建操作任务
	job := &entities.MidjourneyJob{
		JobID:       newJobID,
		UserID:      userID,
		APIKeyID:    apiKeyID,
		Action:      entities.MidjourneyJobActionAction,
		Status:      entities.MidjourneyJobStatusPendingQueue,
		Mode:        entities.MidjourneyJobModeFast,
		Progress:    0,
		HookURL:     &req.NotifyHook,
		Timeout:     300,
		GetUImages:  false,
		ParentJobID: &req.TaskID,
	}

	// 设置请求参数
	params := map[string]interface{}{
		"taskId":   req.TaskID,
		"customId": req.CustomID,
		"state":    req.State,
	}
	if err := job.SetRequestParams(params); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to set request params")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Internal server error",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 提交任务
	if err := h.midjourneyService.SubmitJob(c.Request.Context(), job); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to submit action job")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Failed to submit job",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"job_id":        newJobID,
		"parent_job_id": req.TaskID,
		"custom_id":     req.CustomID,
	}).Info("Action job submitted successfully")

	c.JSON(http.StatusOK, MJResponse{
		Code:        1,
		Description: "提交成功",
		Properties:  map[string]interface{}{},
		Result:      newJobID,
	})
}

// Fetch 获取任务结果端点
// @Summary 获取任务结果
// @Description 获取任务的当前状态和结果
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param id path string true "任务ID"
// @Success 200 {object} MJTaskResponse
// @Failure 400 {object} MJResponse
// @Failure 401 {object} MJResponse
// @Failure 404 {object} MJResponse
// @Failure 500 {object} MJResponse
// @Router /mj/task/{id}/fetch [get]
func (h *MidjourneyHandler) Fetch(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, MJResponse{
			Code:        400,
			Description: "Missing task ID",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 获取用户信息
	userID, hasUserID := utils.GetUserIDFromContext(c)
	if !hasUserID {
		c.JSON(http.StatusUnauthorized, MJResponse{
			Code:        401,
			Description: "Unauthorized",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 获取任务详情
	job, err := h.midjourneyService.GetJob(c.Request.Context(), jobID)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": jobID,
		}).Error("Failed to get job")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Failed to get job",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	if job == nil {
		c.JSON(http.StatusNotFound, MJResponse{
			Code:        404,
			Description: "Task not found",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 检查权限
	if job.UserID != userID {
		c.JSON(http.StatusForbidden, MJResponse{
			Code:        403,
			Description: "Access denied",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 构造响应
	response := h.buildTaskResponse(job)
	c.JSON(http.StatusOK, response)
}

// Blend 图像混合端点
// @Summary 混合图像
// @Description 上传2-5张图像并混合成新图像
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param request body BlendRequest true "混合请求"
// @Success 200 {object} MJResponse
// @Failure 400 {object} MJResponse
// @Failure 401 {object} MJResponse
// @Failure 500 {object} MJResponse
// @Router /mj/submit/blend [post]
func (h *MidjourneyHandler) Blend(c *gin.Context) {
	var req BlendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Invalid blend request")
		c.JSON(http.StatusBadRequest, MJResponse{
			Code:        400,
			Description: "Invalid request parameters: " + err.Error(),
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 获取用户信息
	userID, hasUserID := utils.GetUserIDFromContext(c)
	apiKeyID, hasAPIKeyID := utils.GetAPIKeyIDFromContext(c)
	if !hasUserID || !hasAPIKeyID {
		c.JSON(http.StatusUnauthorized, MJResponse{
			Code:        401,
			Description: "Unauthorized",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 生成任务ID
	jobID := uuid.New().String()

	// 设置默认值
	if req.BotType == "" {
		req.BotType = "MID_JOURNEY"
	}
	if req.Dimensions == "" {
		req.Dimensions = "SQUARE"
	}

	// 创建任务
	job := &entities.MidjourneyJob{
		JobID:      jobID,
		UserID:     userID,
		APIKeyID:   apiKeyID,
		Action:     entities.MidjourneyJobActionBlend,
		Status:     entities.MidjourneyJobStatusPendingQueue,
		Mode:       entities.MidjourneyJobModeFast,
		Progress:   0,
		HookURL:    &req.NotifyHook,
		Timeout:    300,
		GetUImages: false,
	}

	// 设置请求参数
	params := map[string]interface{}{
		"base64Array": req.Base64Array,
		"botType":     req.BotType,
		"dimensions":  req.Dimensions,
		"state":       req.State,
	}
	if err := job.SetRequestParams(params); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to set request params")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Internal server error",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 提交任务
	if err := h.midjourneyService.SubmitJob(c.Request.Context(), job); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to submit blend job")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Failed to submit job",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"job_id": jobID,
	}).Info("Blend job submitted successfully")

	c.JSON(http.StatusOK, MJResponse{
		Code:        1,
		Description: "提交成功",
		Properties:  map[string]interface{}{},
		Result:      jobID,
	})
}

// Describe 图像描述端点
// @Summary 描述图像
// @Description 上传图像并生成四个提示词
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param request body DescribeRequest true "描述请求"
// @Success 200 {object} MJResponse
// @Failure 400 {object} MJResponse
// @Failure 401 {object} MJResponse
// @Failure 500 {object} MJResponse
// @Router /mj/submit/describe [post]
func (h *MidjourneyHandler) Describe(c *gin.Context) {
	var req DescribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Invalid describe request")
		c.JSON(http.StatusBadRequest, MJResponse{
			Code:        400,
			Description: "Invalid request parameters: " + err.Error(),
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 获取用户信息
	userID, hasUserID := utils.GetUserIDFromContext(c)
	apiKeyID, hasAPIKeyID := utils.GetAPIKeyIDFromContext(c)
	if !hasUserID || !hasAPIKeyID {
		c.JSON(http.StatusUnauthorized, MJResponse{
			Code:        401,
			Description: "Unauthorized",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 生成任务ID
	jobID := uuid.New().String()

	// 设置默认值
	if req.BotType == "" {
		req.BotType = "MID_JOURNEY"
	}

	// 创建任务
	job := &entities.MidjourneyJob{
		JobID:      jobID,
		UserID:     userID,
		APIKeyID:   apiKeyID,
		Action:     entities.MidjourneyJobActionDescribe,
		Status:     entities.MidjourneyJobStatusPendingQueue,
		Mode:       entities.MidjourneyJobModeFast,
		Progress:   0,
		HookURL:    &req.NotifyHook,
		Timeout:    300,
		GetUImages: false,
	}

	// 设置请求参数
	params := map[string]interface{}{
		"base64":  req.Base64,
		"botType": req.BotType,
		"state":   req.State,
	}
	if err := job.SetRequestParams(params); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to set request params")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Internal server error",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 提交任务
	if err := h.midjourneyService.SubmitJob(c.Request.Context(), job); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to submit describe job")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Failed to submit job",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"job_id": jobID,
	}).Info("Describe job submitted successfully")

	c.JSON(http.StatusOK, MJResponse{
		Code:        1,
		Description: "提交成功",
		Properties:  map[string]interface{}{},
		Result:      jobID,
	})
}

// Modal 局部重绘端点
// @Summary 局部重绘
// @Description 对图像进行局部重绘
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param request body ModalRequest true "局部重绘请求"
// @Success 200 {object} MJResponse
// @Failure 400 {object} MJResponse
// @Failure 401 {object} MJResponse
// @Failure 500 {object} MJResponse
// @Router /mj/submit/modal [post]
func (h *MidjourneyHandler) Modal(c *gin.Context) {
	var req ModalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Invalid modal request")
		c.JSON(http.StatusBadRequest, MJResponse{
			Code:        400,
			Description: "Invalid request parameters: " + err.Error(),
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 获取用户信息
	userID, hasUserID := utils.GetUserIDFromContext(c)
	apiKeyID, hasAPIKeyID := utils.GetAPIKeyIDFromContext(c)
	if !hasUserID || !hasAPIKeyID {
		c.JSON(http.StatusUnauthorized, MJResponse{
			Code:        401,
			Description: "Unauthorized",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 生成任务ID
	jobID := uuid.New().String()

	// 设置默认值
	if req.BotType == "" {
		req.BotType = "MID_JOURNEY"
	}

	// 创建任务
	job := &entities.MidjourneyJob{
		JobID:      jobID,
		UserID:     userID,
		APIKeyID:   apiKeyID,
		Action:     entities.MidjourneyJobActionInpaint,
		Status:     entities.MidjourneyJobStatusPendingQueue,
		Mode:       entities.MidjourneyJobModeFast,
		Progress:   0,
		Prompt:     &req.Prompt,
		HookURL:    &req.NotifyHook,
		Timeout:    300,
		GetUImages: false,
	}

	// 设置请求参数
	params := map[string]interface{}{
		"base64":     req.Base64,
		"maskBase64": req.MaskBase64,
		"prompt":     req.Prompt,
		"botType":    req.BotType,
		"state":      req.State,
	}
	if err := job.SetRequestParams(params); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to set request params")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Internal server error",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 提交任务
	if err := h.midjourneyService.SubmitJob(c.Request.Context(), job); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to submit modal job")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Failed to submit job",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"job_id": jobID,
	}).Info("Modal job submitted successfully")

	c.JSON(http.StatusOK, MJResponse{
		Code:        1,
		Description: "提交成功",
		Properties:  map[string]interface{}{},
		Result:      jobID,
	})
}

// Cancel 取消任务端点
// @Summary 取消任务
// @Description 取消正在进行的任务
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param request body CancelRequest true "取消请求"
// @Success 200 {object} MJResponse
// @Failure 400 {object} MJResponse
// @Failure 401 {object} MJResponse
// @Failure 500 {object} MJResponse
// @Router /mj/submit/cancel [post]
func (h *MidjourneyHandler) Cancel(c *gin.Context) {
	var req CancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Invalid cancel request")
		c.JSON(http.StatusBadRequest, MJResponse{
			Code:        400,
			Description: "Invalid request parameters: " + err.Error(),
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 获取用户信息
	userID, hasUserID := utils.GetUserIDFromContext(c)
	if !hasUserID {
		c.JSON(http.StatusUnauthorized, MJResponse{
			Code:        401,
			Description: "Unauthorized",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 获取任务详情
	job, err := h.midjourneyService.GetJob(c.Request.Context(), req.TaskID)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": req.TaskID,
		}).Error("Failed to get job")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Failed to get job",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	if job == nil {
		c.JSON(http.StatusNotFound, MJResponse{
			Code:        404,
			Description: "Task not found",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 检查权限
	if job.UserID != userID {
		c.JSON(http.StatusForbidden, MJResponse{
			Code:        403,
			Description: "Access denied",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 检查任务状态
	if job.IsCompleted() {
		c.JSON(http.StatusBadRequest, MJResponse{
			Code:        400,
			Description: "Task already completed",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	// 取消任务
	if err := h.midjourneyService.UpdateJobStatus(c.Request.Context(), req.TaskID, entities.MidjourneyJobStatusFailed); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"job_id": req.TaskID,
		}).Error("Failed to cancel job")
		c.JSON(http.StatusInternalServerError, MJResponse{
			Code:        500,
			Description: "Failed to cancel job",
			Properties:  map[string]interface{}{},
			Result:      nil,
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"job_id": req.TaskID,
	}).Info("Job cancelled successfully")

	c.JSON(http.StatusOK, MJResponse{
		Code:        1,
		Description: "取消成功",
		Properties:  map[string]interface{}{},
		Result:      req.TaskID,
	})
}

// buildTaskResponse 构造任务响应
func (h *MidjourneyHandler) buildTaskResponse(job *entities.MidjourneyJob) MJTaskResponse {
	response := MJTaskResponse{
		Action:     string(job.Action),
		ID:         job.JobID,
		Progress:   strconv.Itoa(job.Progress),
		Status:     string(job.Status),
		SubmitTime: job.CreatedAt.Unix() * 1000, // 转换为毫秒
		Properties: map[string]interface{}{},
	}

	if job.Prompt != nil {
		response.Prompt = *job.Prompt
		response.PromptEn = *job.Prompt // 简化处理，实际应该翻译
	}

	if job.StartedAt != nil {
		startTime := job.StartedAt.Unix() * 1000
		response.StartTime = &startTime
	}

	if job.CompletedAt != nil {
		finishTime := job.CompletedAt.Unix() * 1000
		response.FinishTime = &finishTime
	}

	if job.ErrorMessage != nil {
		response.FailReason = *job.ErrorMessage
	}

	if job.CDNImage != nil {
		response.ImageURL = *job.CDNImage
	}

	// 如果任务成功完成，添加操作按钮
	if job.IsSuccess() && job.Action == entities.MidjourneyJobActionImagine {
		response.Buttons = []MJButton{
			{CustomID: "upsample1", Label: "U1", Type: 2, Style: 2},
			{CustomID: "upsample2", Label: "U2", Type: 2, Style: 2},
			{CustomID: "upsample3", Label: "U3", Type: 2, Style: 2},
			{CustomID: "upsample4", Label: "U4", Type: 2, Style: 2},
			{CustomID: "variation1", Label: "V1", Type: 2, Style: 3},
			{CustomID: "variation2", Label: "V2", Type: 2, Style: 3},
			{CustomID: "variation3", Label: "V3", Type: 2, Style: 3},
			{CustomID: "variation4", Label: "V4", Type: 2, Style: 3},
		}
	}

	return response
}
