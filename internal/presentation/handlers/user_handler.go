package handlers

import (
	"net/http"
	"strconv"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService services.UserService
	logger      logger.Logger
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService services.UserService, logger logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建一个新用户
// @Tags users
// @Accept json
// @Produce json
// @Param request body dto.CreateUserRequest true "创建用户请求"
// @Success 201 {object} dto.Response{data=dto.UserResponse} "创建成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "未认证"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Security BearerAuth
// @Router /admin/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid create user request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"username": req.Username,
			"email":    req.Email,
			"error":    err.Error(),
		}).Error("Failed to create user")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"CREATE_USER_FAILED",
			"Failed to create user",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(user, "User created successfully"))
}

// GetUser 获取用户
// @Summary 获取用户信息
// @Description 根据用户ID获取用户详细信息
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} dto.Response{data=dto.UserResponse} "获取成功"
// @Failure 400 {object} dto.Response "用户ID格式错误"
// @Failure 401 {object} dto.Response "未认证"
// @Failure 404 {object} dto.Response "用户不存在"
// @Security BearerAuth
// @Router /admin/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_USER_ID",
			"Invalid user ID",
			nil,
		))
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id": id,
			"error":   err.Error(),
		}).Error("Failed to get user")

		c.JSON(http.StatusNotFound, dto.ErrorResponse(
			"USER_NOT_FOUND",
			"User not found",
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(user, "User retrieved successfully"))
}

// UpdateUser 更新用户
// @Summary 更新用户信息
// @Description 根据用户ID更新用户信息
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body dto.UpdateUserRequest true "更新用户请求"
// @Success 200 {object} dto.Response{data=dto.UserResponse} "更新成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "未认证"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Security BearerAuth
// @Router /admin/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_USER_ID",
			"Invalid user ID",
			nil,
		))
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid update user request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id": id,
			"error":   err.Error(),
		}).Error("Failed to update user")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"UPDATE_USER_FAILED",
			"Failed to update user",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(user, "User updated successfully"))
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 根据用户ID删除用户
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} dto.Response "删除成功"
// @Failure 400 {object} dto.Response "用户ID格式错误"
// @Failure 401 {object} dto.Response "未认证"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Security BearerAuth
// @Router /admin/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_USER_ID",
			"Invalid user ID",
			nil,
		))
		return
	}

	err = h.userService.DeleteUser(c.Request.Context(), id)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id": id,
			"error":   err.Error(),
		}).Error("Failed to delete user")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"DELETE_USER_FAILED",
			"Failed to delete user",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(nil, "User deleted successfully"))
}

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 分页获取用户列表
// @Tags users
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.UserListResponse} "获取成功"
// @Failure 400 {object} dto.Response "分页参数错误"
// @Failure 401 {object} dto.Response "未认证"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Security BearerAuth
// @Router /admin/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
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

	users, err := h.userService.ListUsers(c.Request.Context(), &pagination)
	if err != nil {
		h.logger.WithField("error", err.Error()).Error("Failed to list users")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"LIST_USERS_FAILED",
			"Failed to list users",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(users, "Users retrieved successfully"))
}

// UpdateBalance 更新用户余额
// @Summary 更新用户余额
// @Description 根据用户ID更新用户余额（增加或扣减）
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body dto.BalanceUpdateRequest true "余额更新请求"
// @Success 200 {object} dto.Response{data=dto.UserResponse} "更新成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "未认证"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Security BearerAuth
// @Router /admin/users/{id}/balance [post]
func (h *UserHandler) UpdateBalance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_USER_ID",
			"Invalid user ID",
			nil,
		))
		return
	}

	var req dto.BalanceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid balance update request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	user, err := h.userService.UpdateBalance(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":   id,
			"operation": req.Operation,
			"amount":    req.Amount,
			"error":     err.Error(),
		}).Error("Failed to update user balance")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"UPDATE_BALANCE_FAILED",
			"Failed to update balance",
			map[string]interface{}{
				"details": err.Error(),
			},
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(user, "Balance updated successfully"))
}
