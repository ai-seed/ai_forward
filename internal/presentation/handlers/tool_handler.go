package handlers

import (
	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/infrastructure/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ToolHandler 工具处理器
type ToolHandler struct {
	toolService *services.ToolService
	logger      logger.Logger
}

// NewToolHandler 创建工具处理器
func NewToolHandler(toolService *services.ToolService, logger logger.Logger) *ToolHandler {
	return &ToolHandler{
		toolService: toolService,
		logger:      logger,
	}
}

// GetTools 获取工具模板列表
// @Summary 获取工具模板列表
// @Description 获取所有可用的工具模板
// @Tags tools
// @Accept json
// @Produce json
// @Success 200 {object} object "获取成功"
// @Failure 500 {object} object "服务器内部错误"
// @Router /tools/types [get]
func (h *ToolHandler) GetTools(c *gin.Context) {
	tools, err := h.toolService.GetTools(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get tools",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tools,
	})
}

// GetPublicTools 获取公开工具列表
// @Summary 获取公开工具列表
// @Description 分页获取公开可用的工具实例
// @Tags tools
// @Accept json
// @Produce json
// @Param limit query int false "限制数量" default(20)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} object "获取成功"
// @Failure 500 {object} object "服务器内部错误"
// @Router /tools/public [get]
func (h *ToolHandler) GetPublicTools(c *gin.Context) {
	// 解析分页参数
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// 这里需要修改service方法来支持分页
	tools, err := h.toolService.GetUserToolInstances(c.Request.Context(), 0, "public")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get public tools",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tools,
	})
}

// GetUserToolInstances 获取用户工具实例列表
// @Summary 获取用户工具实例
// @Description 获取指定用户的所有工具实例
// @Tags tools
// @Accept json
// @Produce json
// @Param category query string false "工具类型" default(all)
// @Success 200 {object} object "获取成功"
// @Failure 401 {object} object "未认证"
// @Failure 500 {object} object "服务器内部错误"
// @Security BearerAuth
// @Router /admin/tools [get]
func (h *ToolHandler) GetUserToolInstances(c *gin.Context) {
	userID := c.GetInt64("user_id")
	category := c.DefaultQuery("category", "all")

	tools, err := h.toolService.GetUserToolInstances(c.Request.Context(), userID, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get user tools",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tools,
	})
}

// CreateUserToolInstance 创建用户工具实例
// @Summary 创建用户工具实例
// @Description 为用户创建新的工具实例
// @Tags tools
// @Accept json
// @Produce json
// @Param request body entities.UserToolInstance true "创建工具实例请求"
// @Success 201 {object} object "创建成功"
// @Failure 400 {object} object "请求参数错误"
// @Failure 401 {object} object "未认证"
// @Failure 500 {object} object "服务器内部错误"
// @Security BearerAuth
// @Router /admin/tools [post]
func (h *ToolHandler) CreateUserToolInstance(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req entities.CreateUserToolInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	tool, err := h.toolService.CreateUserToolInstance(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to create tool",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    tool,
		"message": "Tool created successfully",
	})
}

// GetUserToolInstance 获取用户工具实例详情
// @Summary 获取工具实例详情
// @Description 根据ID获取用户工具实例的详细信息
// @Tags tools
// @Accept json
// @Produce json
// @Param id path int true "工具实例ID"
// @Success 200 {object} object "获取成功"
// @Failure 400 {object} object "ID格式错误"
// @Failure 401 {object} object "未认证"
// @Failure 403 {object} object "访问被拒绝"
// @Failure 404 {object} object "工具实例不存在"
// @Failure 500 {object} object "服务器内部错误"
// @Security BearerAuth
// @Router /admin/tools/{id} [get]
func (h *ToolHandler) GetUserToolInstance(c *gin.Context) {
	userID := c.GetInt64("user_id")
	toolID := c.Param("id")

	tool, err := h.toolService.GetUserToolInstanceByID(c.Request.Context(), toolID, userID)
	if err != nil {
		if err.Error() == "tool not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Tool not found",
			})
			return
		}
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Access denied",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get tool",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tool,
	})
}

// UpdateUserToolInstance 更新用户工具实例
// @Summary 更新工具实例
// @Description 更新用户工具实例的信息
// @Tags tools
// @Accept json
// @Produce json
// @Param id path int true "工具实例ID"
// @Param request body entities.UserToolInstance true "更新工具实例请求"
// @Success 200 {object} object "更新成功"
// @Failure 400 {object} object "请求参数错误"
// @Failure 401 {object} object "未认证"
// @Failure 403 {object} object "访问被拒绝"
// @Failure 404 {object} object "工具实例不存在"
// @Failure 500 {object} object "服务器内部错误"
// @Security BearerAuth
// @Router /admin/tools/{id} [put]
func (h *ToolHandler) UpdateUserToolInstance(c *gin.Context) {
	userID := c.GetInt64("user_id")
	toolID := c.Param("id")

	var req entities.UpdateUserToolInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	tool, err := h.toolService.UpdateUserToolInstance(c.Request.Context(), toolID, userID, &req)
	if err != nil {
		if err.Error() == "tool not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Tool not found",
			})
			return
		}
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Access denied",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to update tool",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tool,
		"message": "Tool updated successfully",
	})
}

// DeleteUserToolInstance 删除用户工具实例
// @Summary 删除工具实例
// @Description 删除指定的用户工具实例
// @Tags tools
// @Accept json
// @Produce json
// @Param id path int true "工具实例ID"
// @Success 200 {object} object "删除成功"
// @Failure 400 {object} object "ID格式错误"
// @Failure 401 {object} object "未认证"
// @Failure 404 {object} object "工具实例不存在"
// @Failure 500 {object} object "服务器内部错误"
// @Security BearerAuth
// @Router /admin/tools/{id} [delete]
func (h *ToolHandler) DeleteUserToolInstance(c *gin.Context) {
	userID := c.GetInt64("user_id")
	toolID := c.Param("id")

	err := h.toolService.DeleteUserToolInstance(c.Request.Context(), toolID, userID)
	if err != nil {
		if err.Error() == "tool not found or not owned by user" {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Tool not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to delete tool",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tool deleted successfully",
	})
}

// GetSharedToolInstance 获取分享的工具实例
// @Summary 获取共享工具实例
// @Description 通过共享ID获取工具实例信息
// @Tags tools
// @Accept json
// @Produce json
// @Param id path string true "共享ID"
// @Success 200 {object} object{success=bool,data=entities.UserToolInstance} "获取成功"
// @Failure 404 {object} object{success=bool,message=string} "工具实例不存在"
// @Failure 500 {object} object{success=bool,message=string,error=string} "服务器内部错误"
// @Router /tools/share/{token} [get]
func (h *ToolHandler) GetSharedToolInstance(c *gin.Context) {
	shareToken := c.Param("token")

	tool, err := h.toolService.GetSharedToolInstance(c.Request.Context(), shareToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get shared tool",
			"error":   err.Error(),
		})
		return
	}

	if tool == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Shared tool not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tool,
	})
}

// IncrementUsage 增加工具使用次数
// @Summary 增加工具使用次数
// @Description 增加指定工具实例的使用次数
// @Tags tools
// @Accept json
// @Produce json
// @Param id path string true "工具ID"
// @Success 200 {object} object{success=bool,message=string} "更新成功"
// @Failure 401 {object} object "未认证"
// @Failure 500 {object} object{success=bool,message=string,error=string} "服务器内部错误"
// @Security BearerAuth
// @Router /admin/tools/{id}/usage [post]
func (h *ToolHandler) IncrementUsage(c *gin.Context) {
	toolID := c.Param("id")

	err := h.toolService.IncrementUsageCount(c.Request.Context(), toolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to increment usage count",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Usage count incremented",
	})
}

// GetToolInstanceByCode 通过code获取工具实例信息（用于第三方鉴权）
// @Summary 通过Code获取工具实例
// @Description 通过授权代码获取工具实例信息，用于第三方鉴权
// @Tags tools
// @Accept json
// @Produce json
// @Param code path string true "授权代码"
// @Success 200 {object} object{success=bool,data=entities.UserToolInstance} "获取成功"
// @Failure 404 {object} object{success=bool,message=string} "工具实例不存在"
// @Failure 500 {object} object{success=bool,message=string,error=string} "服务器内部错误"
// @Router /tools/by-code/{code} [get]
func (h *ToolHandler) GetToolInstanceByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Code parameter is required",
		})
		return
	}

	toolInfo, err := h.toolService.GetToolInstanceByCode(c.Request.Context(), code)
	if err != nil {
		if err.Error() == "tool instance not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Tool instance not found",
			})
			return
		}

		h.logger.WithFields(map[string]interface{}{
			"code":  code,
			"error": err.Error(),
		}).Error("Failed to get tool instance by code")

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get tool instance",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    toolInfo,
		"message": "Tool instance retrieved successfully",
	})
}

// GetModels 获取可用模型列表
// @Summary 获取AI模型列表
// @Description 获取可用的AI模型列表，支持分页查询
// @Tags models
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param provider query string false "厂商筛选"
// @Param type query string false "类型筛选"
// @Success 200 {object} object{success=bool,data=[]entities.Model,total=int64,page=int,page_size=int,total_pages=int} "获取成功"
// @Failure 400 {object} object{success=bool,message=string,error=string} "请求参数错误"
// @Failure 500 {object} object{success=bool,message=string,error=string} "服务器内部错误"
// @Router /tools/models [get]
func (h *ToolHandler) GetModels(c *gin.Context) {
	// 解析分页参数
	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid pagination parameters",
			"error":   err.Error(),
		})
		return
	}

	// 设置默认值
	pagination.SetDefaults()
	if pagination.PageSize > 50 { // 限制每页最大数量为50
		pagination.PageSize = 50
	}

	// 获取筛选参数
	provider := c.Query("provider")
	modelType := c.Query("type")

	// 构建筛选条件
	filters := map[string]interface{}{}
	if provider != "" && provider != "All" {
		filters["provider"] = provider
	}
	if modelType != "" && modelType != "All" {
		filters["type"] = modelType
	}

	// 获取分页的模型列表
	result, err := h.toolService.GetAvailableModelsWithPaginationAndFilters(c.Request.Context(), &pagination, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get models",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"data":        result.Data,
		"total":       result.Total,
		"page":        result.Page,
		"page_size":   result.PageSize,
		"total_pages": result.TotalPages,
	})
}

// GetModelCategories 获取模型分类列表
// @Summary 获取模型分类列表
// @Description 获取所有可用的模型厂商分类和类型分类
// @Tags models
// @Accept json
// @Produce json
// @Success 200 {object} object{success=bool,data=object{providers=[]object,types=[]string}} "获取成功"
// @Failure 500 {object} object{success=bool,message=string,error=string} "服务器内部错误"
// @Router /tools/models/categories [get]
func (h *ToolHandler) GetModelCategories(c *gin.Context) {
	// 获取模型分类信息
	categories, err := h.toolService.GetModelCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get model categories",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    categories,
	})
}

// GetUserAPIKeys 获取用户API密钥列表
// @Summary 获取用户API密钥列表
// @Description 获取当前用户的所有API密钥
// @Tags api-keys
// @Accept json
// @Produce json
// @Success 200 {object} object{success=bool,data=[]entities.APIKey} "获取成功"
// @Failure 401 {object} object "未认证"
// @Failure 500 {object} object{success=bool,message=string,error=string} "服务器内部错误"
// @Security BearerAuth
// @Router /admin/tools/api-keys [get]
func (h *ToolHandler) GetUserAPIKeys(c *gin.Context) {
	userID := c.GetInt64("user_id")

	apiKeys, err := h.toolService.GetUserAPIKeys(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get API keys",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    apiKeys,
	})
}
