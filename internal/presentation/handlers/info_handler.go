package handlers

import (
	"net/http"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/infrastructure/logger"
	"ai-api-gateway/internal/presentation/middleware"

	"github.com/gin-gonic/gin"
)

// InfoHandler 信息查询处理器
type InfoHandler struct {
	modelService    services.ModelService
	usageLogService services.UsageLogService
	logger          logger.Logger
}

// NewInfoHandler 创建信息查询处理器
func NewInfoHandler(
	modelService services.ModelService,
	usageLogService services.UsageLogService,
	logger logger.Logger,
) *InfoHandler {
	return &InfoHandler{
		modelService:    modelService,
		usageLogService: usageLogService,
		logger:          logger,
	}
}

// Models 获取可用模型列表
// @Summary 列出模型
// @Description 获取可用的AI模型列表，包含多语言显示名称和描述
// @Tags AI接口
// @Produce json
// @Security BearerAuth
// @Success 200 {object} clients.ModelsResponse "模型列表"
// @Failure 401 {object} dto.Response "认证失败"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /v1/models [get]
func (h *InfoHandler) Models(c *gin.Context) {
	// 获取可用模型列表
	models, err := h.modelService.GetAvailableModels(c.Request.Context(), 0) // 0 表示获取所有提供商的模型
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to get available models")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to get models",
				"type":    "internal_error",
				"code":    "models_fetch_failed",
			},
		})
		return
	}

	// 转换为 OpenAI API 格式，包含多语言字段
	var modelList []map[string]interface{}
	for _, model := range models {
		modelData := map[string]interface{}{
			"id":       model.Slug,
			"object":   "model",
			"created":  model.CreatedAt.Unix(),
			"owned_by": "system",
		}

		// 添加默认显示名称（向后兼容）
		if model.DisplayName != nil && *model.DisplayName != "" {
			modelData["display_name"] = *model.DisplayName
		} else {
			modelData["display_name"] = model.Name
		}

		// 添加默认描述（向后兼容）
		if model.Description != nil && *model.Description != "" {
			modelData["description"] = *model.Description
		}

		// 添加多语言描述字段
		if model.DescriptionEN != nil && *model.DescriptionEN != "" {
			modelData["description_en"] = *model.DescriptionEN
		}
		if model.DescriptionZH != nil && *model.DescriptionZH != "" {
			modelData["description_zh"] = *model.DescriptionZH
		}
		if model.DescriptionJP != nil && *model.DescriptionJP != "" {
			modelData["description_jp"] = *model.DescriptionJP
		}

		// 添加多语言模型类型字段
		modelTypes := model.GetAllModelTypes()
		if len(modelTypes) > 0 {
			modelData["model_type"] = modelTypes
		}

		// 添加上下文长度
		modelData["context_length"] = model.GetContextLength()

		modelList = append(modelList, modelData)
	}

	// 返回 OpenAI 兼容格式
	response := gin.H{
		"object": "list",
		"data":   modelList,
	}

	c.JSON(http.StatusOK, response)
}

// Usage 获取使用情况
// @Summary 使用统计
// @Description 获取当前用户的API使用统计
// @Tags AI接口
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.UsageResponse "使用统计信息"
// @Failure 401 {object} dto.Response "认证失败"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /v1/usage [get]
func (h *InfoHandler) Usage(c *gin.Context) {
	// 获取认证信息
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
			"AUTHENTICATION_REQUIRED",
			"Authentication required",
			nil,
		))
		return
	}

	// 获取使用统计
	stats, err := h.usageLogService.GetUsageStats(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to get usage stats")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"USAGE_STATS_ERROR",
			"Failed to get usage statistics",
			nil,
		))
		return
	}

	// 构造响应数据
	usageResponse := dto.UsageResponse{
		TotalRequests: int(stats.TotalRequests),
		TotalTokens:   int(stats.TotalTokens),
		TotalCost:     stats.TotalCost,
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(usageResponse, "Usage statistics retrieved successfully"))
}
