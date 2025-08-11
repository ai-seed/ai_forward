package handlers

import (
	"net/http"
	"strconv"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

// APIKeyHandler API密钥处理器
type APIKeyHandler struct {
	apiKeyService     services.APIKeyService
	usageLogRepo      repositories.UsageLogRepository
	billingRecordRepo repositories.BillingRecordRepository
	modelRepo         repositories.ModelRepository
	logger            logger.Logger
}

// NewAPIKeyHandler 创建API密钥处理器
func NewAPIKeyHandler(apiKeyService services.APIKeyService, usageLogRepo repositories.UsageLogRepository, billingRecordRepo repositories.BillingRecordRepository, modelRepo repositories.ModelRepository, logger logger.Logger) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService:     apiKeyService,
		usageLogRepo:      usageLogRepo,
		billingRecordRepo: billingRecordRepo,
		modelRepo:         modelRepo,
		logger:            logger,
	}
}

// CreateAPIKey 创建API密钥
// @Summary 创建API密钥
// @Description 为指定用户创建一个新的API密钥
// @Tags api-keys
// @Accept json
// @Produce json
// @Param request body dto.CreateAPIKeyRequest true "创建API密钥请求"
// @Success 201 {object} dto.Response{data=dto.APIKeyResponse} "创建成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /admin/api-keys [post]
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req dto.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid create API key request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	apiKey, err := h.apiKeyService.CreateAPIKey(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id": req.UserID,
			"name":    req.Name,
			"error":   err.Error(),
		}).Error("Failed to create API key")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"CREATE_API_KEY_FAILED",
			"Failed to create API key",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(apiKey, "API key created successfully"))
}

// GetAPIKey 获取API密钥
// @Summary 获取API密钥信息
// @Description 根据API密钥ID获取详细信息
// @Tags api-keys
// @Accept json
// @Produce json
// @Param id path int true "API密钥ID"
// @Success 200 {object} dto.Response{data=dto.APIKeyResponse} "获取成功"
// @Failure 400 {object} dto.Response "API密钥ID格式错误"
// @Failure 404 {object} dto.Response "API密钥不存在"
// @Router /admin/api-keys/{id} [get]
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_API_KEY_ID",
			"Invalid API key ID",
			nil,
		))
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKey(c.Request.Context(), id)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"api_key_id": id,
			"error":      err.Error(),
		}).Error("Failed to get API key")

		c.JSON(http.StatusNotFound, dto.ErrorResponse(
			"API_KEY_NOT_FOUND",
			"API key not found",
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(apiKey, "API key retrieved successfully"))
}

// UpdateAPIKey 更新API密钥
// @Summary 更新API密钥
// @Description 根据API密钥ID更新API密钥信息
// @Tags api-keys
// @Accept json
// @Produce json
// @Param id path int true "API密钥ID"
// @Param request body dto.UpdateAPIKeyRequest true "更新API密钥请求"
// @Success 200 {object} dto.Response{data=dto.APIKeyResponse} "更新成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /admin/api-keys/{id} [put]
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_API_KEY_ID",
			"Invalid API key ID",
			nil,
		))
		return
	}

	var req dto.UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid update API key request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	apiKey, err := h.apiKeyService.UpdateAPIKey(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"api_key_id": id,
			"error":      err.Error(),
		}).Error("Failed to update API key")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"UPDATE_API_KEY_FAILED",
			"Failed to update API key",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(apiKey, "API key updated successfully"))
}

// DeleteAPIKey 删除API密钥
// @Summary 删除API密钥
// @Description 根据API密钥ID删除API密钥
// @Tags api-keys
// @Accept json
// @Produce json
// @Param id path int true "API密钥ID"
// @Success 200 {object} dto.Response "删除成功"
// @Failure 400 {object} dto.Response "API密钥ID格式错误"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /admin/api-keys/{id} [delete]
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_API_KEY_ID",
			"Invalid API key ID",
			nil,
		))
		return
	}

	err = h.apiKeyService.DeleteAPIKey(c.Request.Context(), id)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"api_key_id": id,
			"error":      err.Error(),
		}).Error("Failed to delete API key")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"DELETE_API_KEY_FAILED",
			"Failed to delete API key",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(nil, "API key deleted successfully"))
}

// RevokeAPIKey 撤销API密钥
// @Summary 撤销API密钥
// @Description 撤销指定的API密钥，使其失效
// @Tags api-keys
// @Accept json
// @Produce json
// @Param id path int true "API密钥ID"
// @Success 200 {object} dto.Response "撤销成功"
// @Failure 400 {object} dto.Response "API密钥ID格式错误"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /admin/api-keys/{id}/revoke [post]
func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_API_KEY_ID",
			"Invalid API key ID",
			nil,
		))
		return
	}

	err = h.apiKeyService.RevokeAPIKey(c.Request.Context(), id)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"api_key_id": id,
			"error":      err.Error(),
		}).Error("Failed to revoke API key")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"REVOKE_API_KEY_FAILED",
			"Failed to revoke API key",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(nil, "API key revoked successfully"))
}

// ListAPIKeys 获取API密钥列表
// @Summary 获取API密钥列表
// @Description 分页获取所有API密钥列表
// @Tags api-keys
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.APIKeyListResponse} "获取成功"
// @Failure 400 {object} dto.Response "分页参数错误"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /admin/api-keys [get]
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	// 解析分页参数
	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid pagination parameters")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_PAGINATION",
			"Invalid pagination parameters",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	pagination.SetDefaults()

	apiKeys, err := h.apiKeyService.ListAPIKeys(c.Request.Context(), &pagination)
	if err != nil {
		h.logger.WithField("error", err.Error()).Error("Failed to list API keys")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"LIST_API_KEYS_FAILED",
			"Failed to list API keys",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(apiKeys, "API keys retrieved successfully"))
}

// GetUserAPIKeys 获取用户的API密钥列表
// @Summary 获取用户API密钥
// @Description 获取指定用户的所有API密钥
// @Tags api-keys
// @Accept json
// @Produce json
// @Param user_id path int true "用户ID"
// @Success 200 {object} dto.Response{data=[]dto.APIKeyResponse} "获取成功"
// @Failure 400 {object} dto.Response "用户ID格式错误"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /admin/users/{user_id}/api-keys [get]
func (h *APIKeyHandler) GetUserAPIKeys(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_USER_ID",
			"Invalid user ID",
			nil,
		))
		return
	}

	apiKeys, err := h.apiKeyService.GetUserAPIKeys(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to get user API keys")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"GET_USER_API_KEYS_FAILED",
			"Failed to get user API keys",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(apiKeys, "User API keys retrieved successfully"))
}

// GetAPIKeyUsageLogs 获取API密钥使用日志
// @Summary 获取API密钥使用日志
// @Description 分页获取指定API密钥的使用日志
// @Tags api-keys
// @Accept json
// @Produce json
// @Param id path int true "API密钥ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param start_date query string false "开始日期" format(date)
// @Param end_date query string false "结束日期" format(date)
// @Param model query string false "模型名称"
// @Success 200 {object} dto.Response{data=dto.PaginatedResponse} "获取成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /admin/api-keys/{id}/usage-logs [get]
func (h *APIKeyHandler) GetAPIKeyUsageLogs(c *gin.Context) {
	idStr := c.Param("id")
	apiKeyID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_API_KEY_ID",
			"Invalid API key ID",
			nil,
		))
		return
	}

	// 绑定查询参数
	var req dto.UsageLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_QUERY_PARAMS",
			"Invalid query parameters",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	req.APIKeyID = apiKeyID

	// 验证参数
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}

	h.logger.WithFields(map[string]interface{}{
		"api_key_id": apiKeyID,
		"start_date": req.StartDate,
		"end_date":   req.EndDate,
		"page":       req.Page,
		"page_size":  req.PageSize,
	}).Debug("Querying usage logs with date range")

	// 从数据库获取使用日志
	offset := (req.Page - 1) * req.PageSize
	usageLogEntities, err := h.usageLogRepo.GetByAPIKeyIDAndDateRange(c.Request.Context(), apiKeyID, req.StartDate, req.EndDate, offset, req.PageSize)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"api_key_id": apiKeyID,
		}).Error("Failed to get usage logs from database")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"DATABASE_ERROR",
			"Failed to retrieve usage logs",
			nil,
		))
		return
	}

	// 获取总数
	total, err := h.usageLogRepo.CountByAPIKeyIDAndDateRange(c.Request.Context(), apiKeyID, req.StartDate, req.EndDate)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"api_key_id": apiKeyID,
		}).Error("Failed to count usage logs from database")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"DATABASE_ERROR",
			"Failed to count usage logs",
			nil,
		))
		return
	}

	// 转换为响应DTO
	usageLogs := make([]dto.UsageLogResponse, len(usageLogEntities))
	for i, entity := range usageLogEntities {
		// 获取模型名称
		modelName := "unknown"
		if model, err := h.modelRepo.GetByID(c.Request.Context(), entity.ModelID); err == nil {
			modelName = model.GetDisplayName()
		}

		usageLogs[i] = dto.UsageLogResponse{
			ID:          entity.ID,
			APIKeyID:    entity.APIKeyID,
			UserID:      entity.UserID,
			Model:       modelName,
			TokensUsed:  entity.TotalTokens,
			Cost:        entity.Cost,
			RequestType: entity.Endpoint, // 暂时使用endpoint作为request_type
			Status:      getStatusFromCode(entity.StatusCode),
			RequestID:   entity.RequestID,
			IPAddress:   "", // TODO: 如果需要可以添加到entity
			UserAgent:   "", // TODO: 如果需要可以添加到entity
			Timestamp:   entity.CreatedAt,
		}
	}

	response := dto.PaginatedResponse{
		Data:       usageLogs,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: int((total + int64(req.PageSize) - 1) / int64(req.PageSize)),
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(response, "Usage logs retrieved successfully"))
}

// getStatusFromCode 根据HTTP状态码获取状态字符串
func getStatusFromCode(statusCode int) string {
	if statusCode >= 200 && statusCode < 300 {
		return "success"
	}
	return "error"
}

// getStringValue 获取字符串指针的值，如果为nil则返回空字符串
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// GetAPIKeyBillingRecords 获取API密钥扣费记录
// @Summary 获取API密钥扣费记录
// @Description 分页获取指定API密钥的扣费记录
// @Tags api-keys
// @Accept json
// @Produce json
// @Param id path int true "API密钥ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param start_date query string false "开始日期" format(date)
// @Param end_date query string false "结束日期" format(date)
// @Success 200 {object} dto.Response{data=dto.PaginatedResponse} "获取成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /admin/api-keys/{id}/billing-records [get]
func (h *APIKeyHandler) GetAPIKeyBillingRecords(c *gin.Context) {
	idStr := c.Param("id")
	apiKeyID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_API_KEY_ID",
			"Invalid API key ID",
			nil,
		))
		return
	}

	// 绑定查询参数
	var req dto.BillingRecordListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_QUERY_PARAMS",
			"Invalid query parameters",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	req.APIKeyID = apiKeyID

	// 验证参数
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}

	h.logger.WithFields(map[string]interface{}{
		"api_key_id": apiKeyID,
		"start_date": req.StartDate,
		"end_date":   req.EndDate,
		"page":       req.Page,
		"page_size":  req.PageSize,
	}).Debug("Querying billing records with date range")

	// 从数据库获取扣费记录
	offset := (req.Page - 1) * req.PageSize
	billingRecordEntities, err := h.billingRecordRepo.GetByAPIKeyIDAndDateRange(c.Request.Context(), apiKeyID, req.StartDate, req.EndDate, offset, req.PageSize)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"api_key_id": apiKeyID,
		}).Error("Failed to get billing records from database")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"DATABASE_ERROR",
			"Failed to retrieve billing records",
			nil,
		))
		return
	}

	// 获取总数
	total, err := h.billingRecordRepo.CountByAPIKeyIDAndDateRange(c.Request.Context(), apiKeyID, req.StartDate, req.EndDate)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"api_key_id": apiKeyID,
		}).Error("Failed to count billing records from database")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"DATABASE_ERROR",
			"Failed to count billing records",
			nil,
		))
		return
	}

	// 转换为响应DTO
	billingRecords := make([]dto.BillingRecordResponse, len(billingRecordEntities))
	for i, entity := range billingRecordEntities {
		billingRecords[i] = dto.BillingRecordResponse{
			ID:              entity.ID,
			UserID:          entity.UserID,
			Amount:          entity.Amount,
			Description:     getStringValue(entity.Description),
			TransactionType: string(entity.BillingType),
			BalanceBefore:   0.0, // TODO: 需要计算或存储余额变化
			BalanceAfter:    0.0, // TODO: 需要计算或存储余额变化
			Timestamp:       entity.CreatedAt,
		}
	}

	response := dto.PaginatedResponse{
		Data:       billingRecords,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: int((total + int64(req.PageSize) - 1) / int64(req.PageSize)),
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(response, "Billing records retrieved successfully"))
}
