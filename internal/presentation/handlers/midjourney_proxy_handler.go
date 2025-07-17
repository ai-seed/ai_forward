package handlers

import (
	"io"
	"net/http"
	"net/url"

	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

// MidjourneyProxyHandler Midjourney代理处理器 - 直接转发请求
type MidjourneyProxyHandler struct {
	forwardingService *clients.MidjourneyForwardingService
	logger            logger.Logger
}

// NewMidjourneyProxyHandler 创建代理处理器
func NewMidjourneyProxyHandler(forwardingService *clients.MidjourneyForwardingService, logger logger.Logger) *MidjourneyProxyHandler {
	return &MidjourneyProxyHandler{
		forwardingService: forwardingService,
		logger:            logger,
	}
}

// ProxyRequest 通用的请求转发处理器
// @Summary 转发Midjourney请求
// @Description 将请求直接转发到上游Midjourney服务
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 400 {object} map[string]interface{} "请求错误"
// @Failure 401 {object} map[string]interface{} "认证失败"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /mj/* [get,post,put,delete]
func (h *MidjourneyProxyHandler) ProxyRequest(c *gin.Context) {
	// 获取请求方法和路径
	method := c.Request.Method
	path := c.Request.URL.Path
	
	// 获取查询参数
	query := c.Request.URL.Query()
	
	// 获取请求头
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	
	// 读取请求体
	var body []byte
	var err error
	if c.Request.Body != nil {
		body, err = io.ReadAll(c.Request.Body)
		if err != nil {
			h.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
				"path":  path,
			}).Error("Failed to read request body")
			
			c.JSON(http.StatusBadRequest, gin.H{
				"code":        400,
				"description": "Failed to read request body",
				"properties":  map[string]interface{}{},
				"result":      nil,
			})
			return
		}
	}
	
	h.logger.WithFields(map[string]interface{}{
		"method":    method,
		"path":      path,
		"body_size": len(body),
		"query":     query,
	}).Info("Forwarding Midjourney request")
	
	// 转发请求
	response, err := h.forwardingService.ForwardMidjourneyRequest(
		c.Request.Context(),
		method,
		path,
		headers,
		body,
		query,
	)
	
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
			"path":  path,
		}).Error("Failed to forward request")
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":        500,
			"description": "Failed to forward request to upstream service",
			"properties":  map[string]interface{}{},
			"result":      nil,
		})
		return
	}
	
	// 设置响应头
	for key, value := range response.Headers {
		c.Header(key, value)
	}
	
	// 返回响应
	c.Data(response.StatusCode, c.GetHeader("Content-Type"), response.Body)
}

// 为了保持向后兼容，提供具体的端点处理器

// Imagine 图像生成端点
// @Summary 生成图像
// @Description 根据提示词生成图像，类似 /imagine 命令
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param request body map[string]interface{} true "图像生成请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /mj/submit/imagine [post]
func (h *MidjourneyProxyHandler) Imagine(c *gin.Context) {
	h.handleSpecificEndpoint(c, "/mj/submit/imagine")
}

// Action 操作端点
// @Summary 执行操作
// @Description 执行 U1-U4、V1-V4 等操作
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param request body map[string]interface{} true "操作请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /mj/submit/action [post]
func (h *MidjourneyProxyHandler) Action(c *gin.Context) {
	h.handleSpecificEndpoint(c, "/mj/submit/action")
}

// Blend 图像混合端点
// @Summary 混合图像
// @Description 上传2-5张图像并混合成新图像
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param request body map[string]interface{} true "混合请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /mj/submit/blend [post]
func (h *MidjourneyProxyHandler) Blend(c *gin.Context) {
	h.handleSpecificEndpoint(c, "/mj/submit/blend")
}

// Describe 图像描述端点
// @Summary 描述图像
// @Description 上传图像并生成四个提示词
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param request body map[string]interface{} true "描述请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /mj/submit/describe [post]
func (h *MidjourneyProxyHandler) Describe(c *gin.Context) {
	h.handleSpecificEndpoint(c, "/mj/submit/describe")
}

// Modal 局部重绘端点
// @Summary 局部重绘
// @Description 对图像进行局部重绘
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param request body map[string]interface{} true "局部重绘请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /mj/submit/modal [post]
func (h *MidjourneyProxyHandler) Modal(c *gin.Context) {
	h.handleSpecificEndpoint(c, "/mj/submit/modal")
}

// Cancel 取消任务端点
// @Summary 取消任务
// @Description 取消正在进行的任务
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param request body map[string]interface{} true "取消请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /mj/submit/cancel [post]
func (h *MidjourneyProxyHandler) Cancel(c *gin.Context) {
	h.handleSpecificEndpoint(c, "/mj/submit/cancel")
}

// Fetch 获取任务结果端点
// @Summary 获取任务结果
// @Description 获取任务的当前状态和结果
// @Tags Midjourney
// @Accept json
// @Produce json
// @Param mj-api-secret header string true "API密钥"
// @Param id path string true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /mj/task/{id}/fetch [get]
func (h *MidjourneyProxyHandler) Fetch(c *gin.Context) {
	taskID := c.Param("id")
	path := "/mj/task/" + taskID + "/fetch"
	h.handleSpecificEndpoint(c, path)
}

// handleSpecificEndpoint 处理特定端点的通用方法
func (h *MidjourneyProxyHandler) handleSpecificEndpoint(c *gin.Context, targetPath string) {
	// 获取请求头
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	
	// 读取请求体
	var body []byte
	var err error
	if c.Request.Body != nil {
		body, err = io.ReadAll(c.Request.Body)
		if err != nil {
			h.logger.WithFields(map[string]interface{}{
				"error": err.Error(),
				"path":  targetPath,
			}).Error("Failed to read request body")
			
			c.JSON(http.StatusBadRequest, gin.H{
				"code":        400,
				"description": "Failed to read request body",
				"properties":  map[string]interface{}{},
				"result":      nil,
			})
			return
		}
	}
	
	// 获取查询参数
	var query url.Values
	if c.Request.URL.RawQuery != "" {
		query = c.Request.URL.Query()
	}
	
	h.logger.WithFields(map[string]interface{}{
		"method":      c.Request.Method,
		"target_path": targetPath,
		"body_size":   len(body),
	}).Info("Forwarding specific Midjourney request")
	
	// 转发请求
	response, err := h.forwardingService.ForwardMidjourneyRequest(
		c.Request.Context(),
		c.Request.Method,
		targetPath,
		headers,
		body,
		query,
	)
	
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
			"path":  targetPath,
		}).Error("Failed to forward request")
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":        500,
			"description": "Failed to forward request to upstream service",
			"properties":  map[string]interface{}{},
			"result":      nil,
		})
		return
	}
	
	// 设置响应头
	for key, value := range response.Headers {
		c.Header(key, value)
	}
	
	// 返回响应
	c.Data(response.StatusCode, c.GetHeader("Content-Type"), response.Body)
}
