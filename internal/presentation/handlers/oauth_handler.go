package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/application/services"
	"ai-api-gateway/internal/infrastructure/config"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

// OAuthHandler OAuth处理器
type OAuthHandler struct {
	oauthService services.OAuthService
	logger       logger.Logger
	config       *config.Config
}

// NewOAuthHandler 创建OAuth处理器
func NewOAuthHandler(oauthService services.OAuthService, logger logger.Logger, config *config.Config) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
		logger:       logger,
		config:       config,
	}
}

// GetAuthURL 获取OAuth认证URL
// @Summary 获取OAuth认证URL
// @Description 获取指定提供商的OAuth认证URL
// @Tags OAuth
// @Accept json
// @Produce json
// @Param provider path string true "OAuth提供商" Enums(google,github)
// @Success 200 {object} dto.Response{data=object{auth_url=string,state=string}} "获取成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/oauth/{provider}/url [get]
func (h *OAuthHandler) GetAuthURL(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		h.logger.Warn("OAuth provider not specified")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"OAuth provider is required",
			nil,
		))
		return
	}

	// 验证提供商
	provider = strings.ToLower(provider)
	if provider != "google" && provider != "github" {
		h.logger.WithFields(map[string]interface{}{
			"provider": provider,
		}).Warn("Unsupported OAuth provider")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"UNSUPPORTED_PROVIDER",
			"Unsupported OAuth provider",
			map[string]interface{}{"provider": provider},
		))
		return
	}

	// 获取认证URL
	authURL, state, err := h.oauthService.GetAuthURL(c.Request.Context(), provider)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"provider": provider,
			"error":    err.Error(),
		}).Error("Failed to get OAuth auth URL")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"OAUTH_URL_FAILED",
			"Failed to generate OAuth URL",
			nil,
		))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"provider": provider,
		"state":    state,
	}).Info("OAuth auth URL generated successfully")

	c.JSON(http.StatusOK, dto.SuccessResponse(map[string]interface{}{
		"auth_url": authURL,
		"state":    state,
	}, "OAuth URL generated successfully"))
}

// HandleCallback 处理OAuth回调
// @Summary 处理OAuth回调
// @Description 处理OAuth提供商的回调，完成用户登录
// @Tags OAuth
// @Accept json
// @Produce json
// @Param provider path string true "OAuth提供商" Enums(google,github)
// @Param request body dto.OAuthLoginRequest true "OAuth回调请求"
// @Success 200 {object} dto.LoginResponse "登录成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "认证失败"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/oauth/{provider}/callback [post]
func (h *OAuthHandler) HandleCallback(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		h.logger.Warn("OAuth provider not specified in callback")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"OAuth provider is required",
			nil,
		))
		return
	}

	// 验证提供商
	provider = strings.ToLower(provider)
	if provider != "google" && provider != "github" {
		h.logger.WithFields(map[string]interface{}{
			"provider": provider,
		}).Warn("Unsupported OAuth provider in callback")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"UNSUPPORTED_PROVIDER",
			"Unsupported OAuth provider",
			map[string]interface{}{"provider": provider},
		))
		return
	}

	// 解析请求体
	var req dto.OAuthLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"provider": provider,
			"error":    err.Error(),
		}).Warn("Invalid OAuth callback request")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			map[string]interface{}{"details": err.Error()},
		))
		return
	}

	// 设置提供商
	req.Provider = provider

	// 处理OAuth回调
	response, err := h.oauthService.HandleCallback(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"provider": provider,
			"code":     req.Code[:10] + "...", // 只记录部分code用于调试
			"state":    req.State,
			"error":    err.Error(),
		}).Error("OAuth callback failed")

		// 根据错误类型返回不同的状态码
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "expired") {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
				"OAUTH_FAILED",
				"OAuth authentication failed",
				nil,
			))
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
				"OAUTH_ERROR",
				"OAuth processing error",
				nil,
			))
		}
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"provider": provider,
		"user_id":  response.User.ID,
		"username": response.User.Username,
	}).Info("OAuth login successful")

	c.JSON(http.StatusOK, dto.SuccessResponse(response, "OAuth login successful"))
}

// GetAuthURLFromQuery 从查询参数获取OAuth认证URL（用于前端重定向）
// @Summary 从查询参数获取OAuth认证URL
// @Description 获取指定提供商的OAuth认证URL并重定向
// @Tags OAuth
// @Param provider path string true "OAuth提供商" Enums(google,github)
// @Success 302 {string} string "重定向到OAuth提供商"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/oauth/{provider}/redirect [get]
func (h *OAuthHandler) GetAuthURLFromQuery(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		h.logger.Warn("OAuth provider not specified for redirect")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"OAuth provider is required",
			nil,
		))
		return
	}

	// 验证提供商
	provider = strings.ToLower(provider)
	if provider != "google" && provider != "github" {
		h.logger.WithFields(map[string]interface{}{
			"provider": provider,
		}).Warn("Unsupported OAuth provider for redirect")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"UNSUPPORTED_PROVIDER",
			"Unsupported OAuth provider",
			map[string]interface{}{"provider": provider},
		))
		return
	}

	// 获取认证URL
	authURL, state, err := h.oauthService.GetAuthURL(c.Request.Context(), provider)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"provider": provider,
			"error":    err.Error(),
		}).Error("Failed to get OAuth auth URL for redirect")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
			"OAUTH_URL_FAILED",
			"Failed to generate OAuth URL",
			nil,
		))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"provider": provider,
		"state":    state,
	}).Info("Redirecting to OAuth provider")

	// 重定向到OAuth提供商
	c.Redirect(http.StatusFound, authURL)
}

// HandleCallbackFromQuery 从查询参数处理OAuth回调
// @Summary 从查询参数处理OAuth回调
// @Description 处理OAuth提供商的回调，从查询参数获取code和state
// @Tags OAuth
// @Param provider path string true "OAuth提供商" Enums(google,github)
// @Param code query string true "授权码"
// @Param state query string true "状态参数"
// @Success 200 {object} dto.LoginResponse "登录成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "认证失败"
// @Failure 500 {object} dto.Response "服务器内部错误"
// @Router /auth/oauth/{provider}/callback [get]
func (h *OAuthHandler) HandleCallbackFromQuery(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	if provider == "" || code == "" || state == "" {
		h.logger.WithFields(map[string]interface{}{
			"provider":  provider,
			"has_code":  code != "",
			"has_state": state != "",
		}).Warn("Missing required OAuth callback parameters")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"INVALID_REQUEST",
			"Missing required parameters: provider, code, and state are required",
			nil,
		))
		return
	}

	// 验证提供商
	provider = strings.ToLower(provider)
	if provider != "google" && provider != "github" {
		h.logger.WithFields(map[string]interface{}{
			"provider": provider,
		}).Warn("Unsupported OAuth provider in query callback")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse(
			"UNSUPPORTED_PROVIDER",
			"Unsupported OAuth provider",
			map[string]interface{}{"provider": provider},
		))
		return
	}

	// 构造请求
	req := &dto.OAuthLoginRequest{
		Provider: provider,
		Code:     code,
		State:    state,
	}

	// 处理OAuth回调
	response, err := h.oauthService.HandleCallback(c.Request.Context(), req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"provider": provider,
			"code":     code[:10] + "...", // 只记录部分code用于调试
			"state":    state,
			"error":    err.Error(),
		}).Error("OAuth query callback failed")

		// 根据错误类型返回不同的状态码
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "expired") {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse(
				"OAUTH_FAILED",
				"OAuth authentication failed",
				nil,
			))
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse(
				"OAUTH_ERROR",
				"OAuth processing error",
				nil,
			))
		}
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"provider": provider,
		"user_id":  response.User.ID,
		"username": response.User.Username,
	}).Info("OAuth query callback login successful")

	// 重定向到前端，携带token信息
	frontendURL := fmt.Sprintf("%s/auth/oauth/callback?access_token=%s&refresh_token=%s",
		h.config.OAuth.FrontendURL, response.AccessToken, response.RefreshToken)

	c.Redirect(http.StatusFound, frontendURL)
}
