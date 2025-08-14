package handlers

import (
	"net/http"
	"strings"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService services.AuthService
	logger      logger.Logger
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService services.AuthService, logger logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Login 用户登录
// @Summary 用户登录
// @Description 使用用户名和密码进行登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "登录请求"
// @Success 200 {object} dto.LoginResponse "登录成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "认证失败"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Invalid login request")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// 调用认证服务进行登录
	response, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"username": req.Username,
			"error":    err.Error(),
		}).Warn("Login failed")

		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
			"LOGIN_FAILED",
			"Invalid username or password",
			nil,
		))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"username": req.Username,
		"user_id":  response.User.ID,
	}).Info("User logged in successfully")

	c.JSON(http.StatusOK, dto.SuccessResponse(response, "Login successful"))
}

// Register 用户注册
// @Summary 用户注册
// @Description 注册新用户账户
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "注册请求"
// @Success 201 {object} dto.RegisterResponse "注册成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 409 {object} dto.Response "用户名或邮箱已存在"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Invalid register request")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// 调用认证服务进行注册
	response, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"username": req.Username,
			"email":    req.Email,
			"error":    err.Error(),
		}).Warn("Registration failed")

		// 根据错误类型返回不同的状态码
		statusCode := http.StatusInternalServerError
		errorCode := "REGISTRATION_FAILED"
		if err.Error() == "email already exists" {
			statusCode = http.StatusConflict
			errorCode = "USER_EXISTS"
		}

		c.JSON(statusCode, dto.ErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"username": req.Username,
		"user_id":  response.ID,
	}).Info("User registered successfully")

	c.JSON(http.StatusCreated, dto.SuccessResponse(response, "User registered successfully"))
}

// RefreshToken 刷新令牌
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "刷新令牌请求"
// @Success 200 {object} dto.RefreshTokenResponse "刷新成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "刷新令牌无效"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Invalid refresh token request")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// 调用认证服务刷新令牌
	response, err := h.authService.RefreshToken(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Token refresh failed")

		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
			"TOKEN_REFRESH_FAILED",
			"Invalid refresh token",
			nil,
		))
		return
	}

	h.logger.Debug("Token refreshed successfully")
	c.JSON(http.StatusOK, dto.SuccessResponse(response, "Token refreshed successfully"))
}

// GetProfile 获取用户资料
// @Summary 获取当前用户资料
// @Description 获取当前登录用户的详细信息
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.GetUserProfileResponse "获取成功"
// @Failure 401 {object} dto.Response "未认证"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
			"AUTHENTICATION_REQUIRED",
			"User authentication required",
			nil,
		))
		return
	}

	userIDInt64, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"INTERNAL_ERROR",
			"Invalid user ID format",
			nil,
		))
		return
	}

	// 获取用户资料
	response, err := h.authService.GetUserProfile(c.Request.Context(), userIDInt64)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id": userIDInt64,
			"error":   err.Error(),
		}).Error("Failed to get user profile")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"PROFILE_FETCH_FAILED",
			"Failed to get user profile",
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(response, "User profile retrieved successfully"))
}

// ChangePassword 修改密码
// @Summary 修改用户密码
// @Description 修改当前用户的密码
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.ChangePasswordRequest true "修改密码请求"
// @Success 200 {object} dto.Response "修改成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "未认证或旧密码错误"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
			"AUTHENTICATION_REQUIRED",
			"User authentication required",
			nil,
		))
		return
	}

	userIDInt64, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"INTERNAL_ERROR",
			"Invalid user ID format",
			nil,
		))
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Invalid change password request")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// 调用认证服务修改密码
	err := h.authService.ChangePassword(c.Request.Context(), userIDInt64, &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id": userIDInt64,
			"error":   err.Error(),
		}).Warn("Password change failed")

		statusCode := http.StatusInternalServerError
		if err.Error() == "old password is incorrect" {
			statusCode = http.StatusUnauthorized
		}

		c.JSON(statusCode, dto.ErrorResponse(
			"PASSWORD_CHANGE_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id": userIDInt64,
	}).Info("Password changed successfully")

	c.JSON(http.StatusOK, dto.SuccessResponse(nil, "Password changed successfully"))
}

// SendVerificationCode 发送验证码
// @Summary 发送验证码
// @Description 发送邮箱验证码（注册或密码重置）
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.SendVerificationCodeRequest true "发送验证码请求"
// @Success 200 {object} dto.SendVerificationCodeResponse "验证码发送成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 429 {object} dto.Response "发送过于频繁"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/send-verification-code [post]
func (h *AuthHandler) SendVerificationCode(c *gin.Context) {
	var req dto.SendVerificationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Invalid send verification code request")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// 调用认证服务发送验证码
	response, err := h.authService.SendVerificationCode(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"email": req.Email,
			"type":  req.Type,
			"error": err.Error(),
		}).Warn("Failed to send verification code")

		statusCode := http.StatusInternalServerError
		errorCode := "SEND_CODE_FAILED"
		if err.Error() == "Verification code sent too frequently" {
			statusCode = http.StatusTooManyRequests
			errorCode = "SEND_TOO_FREQUENT"
		}

		c.JSON(statusCode, dto.ErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"email": req.Email,
		"type":  req.Type,
	}).Info("Verification code sent successfully")

	c.JSON(http.StatusOK, dto.SuccessResponse(response, "Verification code sent successfully"))
}

// VerifyCode 验证验证码
// @Summary 验证验证码
// @Description 验证邮箱验证码
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.VerifyCodeRequest true "验证验证码请求"
// @Success 200 {object} dto.VerifyCodeResponse "验证结果"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/verify-code [post]
func (h *AuthHandler) VerifyCode(c *gin.Context) {
	var req dto.VerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Invalid verify code request")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// 调用认证服务验证验证码
	response, err := h.authService.VerifyCode(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"email": req.Email,
			"type":  req.Type,
			"error": err.Error(),
		}).Warn("Failed to verify code")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"VERIFY_CODE_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(response, "Code verification completed"))
}

// RegisterWithCode 带验证码的注册
// @Summary 带验证码的用户注册
// @Description 使用邮箱验证码进行用户注册
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.RegisterWithCodeRequest true "注册请求"
// @Success 201 {object} dto.RegisterResponse "注册成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 409 {object} dto.Response "用户已存在"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/register-with-code [post]
func (h *AuthHandler) RegisterWithCode(c *gin.Context) {
	var req dto.RegisterWithCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Invalid register with code request")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// 调用认证服务进行注册
	response, err := h.authService.RegisterWithCode(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"username": req.Username,
			"email":    req.Email,
			"error":    err.Error(),
		}).Warn("Registration with code failed")

		// 根据错误类型返回不同的状态码
		statusCode := http.StatusInternalServerError
		errorCode := "REGISTRATION_FAILED"
		errorMsg := err.Error()

		if errorMsg == "email already exists" {
			statusCode = http.StatusConflict
			errorCode = "USER_EXISTS"
		} else if strings.Contains(errorMsg, "invalid verification code") {
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_VERIFICATION_CODE"
		}

		c.JSON(statusCode, dto.ErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"username": req.Username,
		"user_id":  response.ID,
	}).Info("User registered successfully with verification code")

	c.JSON(http.StatusCreated, dto.SuccessResponse(response, "User registered successfully"))
}

// ResetPassword 重置密码
// @Summary 重置密码
// @Description 使用邮箱验证码重置密码
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.ResetPasswordRequest true "重置密码请求"
// @Success 200 {object} dto.ResetPasswordResponse "密码重置成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 404 {object} dto.Response "用户不存在"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Invalid reset password request")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// 调用认证服务重置密码
	response, err := h.authService.ResetPassword(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"email": req.Email,
			"error": err.Error(),
		}).Warn("Password reset failed")

		statusCode := http.StatusInternalServerError
		errorCode := "RESET_PASSWORD_FAILED"
		if err.Error() == "user not found" {
			statusCode = http.StatusNotFound
			errorCode = "USER_NOT_FOUND"
		} else if err.Error() == "invalid verification code" {
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_VERIFICATION_CODE"
		}

		c.JSON(statusCode, dto.ErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"email": req.Email,
	}).Info("Password reset successfully")

	c.JSON(http.StatusOK, dto.SuccessResponse(response, "Password reset successfully"))
}
