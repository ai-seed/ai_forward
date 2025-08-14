package handlers

import (
	"net/http"
	"strconv"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/services"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

// PaymentHandler 支付处理器
type PaymentHandler struct {
	paymentSvc     services.PaymentService
	transactionSvc services.TransactionService
	giftSvc        services.GiftService
	logger         logger.Logger
}

// NewPaymentHandler 创建支付处理器
func NewPaymentHandler(
	paymentSvc services.PaymentService,
	transactionSvc services.TransactionService,
	giftSvc services.GiftService,
	logger logger.Logger,
) *PaymentHandler {
	return &PaymentHandler{
		paymentSvc:     paymentSvc,
		transactionSvc: transactionSvc,
		giftSvc:        giftSvc,
		logger:         logger,
	}
}

// CreateRechargeOrder 创建充值订单
// @Summary 创建充值订单
// @Description 用户创建充值订单
// @Tags 支付
// @Accept json
// @Produce json
// @Param request body dto.CreateRechargeRequest true "充值请求"
// @Success 200 {object} dto.RechargeResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/payment/recharge [post]
func (h *PaymentHandler) CreateRechargeOrder(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("UNAUTHORIZED", "User not authenticated", nil))
		return
	}

	var req dto.CreateRechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_REQUEST", "Invalid request format", map[string]interface{}{"error": err.Error()}))
		return
	}

	response, err := h.paymentSvc.CreateRechargeOrder(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"user_id":           userID,
			"amount":            req.Amount,
			"payment_method_id": req.PaymentMethodID,
			"error":             err.Error(),
		}).Error("Failed to create recharge order")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("CREATE_ORDER_FAILED", "Failed to create recharge order", map[string]interface{}{"error": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(response, "Recharge order created successfully"))
}

// GetRechargeOrder 获取充值订单详情
// @Summary 获取充值订单详情
// @Description 根据订单ID获取充值订单详情
// @Tags 支付
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} dto.RechargeResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/v1/payment/recharge/{id} [get]
func (h *PaymentHandler) GetRechargeOrder(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("UNAUTHORIZED", "User not authenticated", nil))
		return
	}

	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_ORDER_ID", "Invalid order ID", map[string]interface{}{"error": err.Error()}))
		return
	}

	response, err := h.paymentSvc.GetRechargeOrder(c.Request.Context(), userID, orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse("ORDER_NOT_FOUND", "Recharge order not found", map[string]interface{}{"error": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(response, "Recharge order retrieved successfully"))
}

// CancelRechargeOrder 取消充值订单
// @Summary 取消充值订单
// @Description 用户取消待支付的充值订单
// @Tags 支付
// @Accept json
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/v1/payment/recharge/{id}/cancel [post]
func (h *PaymentHandler) CancelRechargeOrder(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("UNAUTHORIZED", "User not authenticated", nil))
		return
	}

	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_ORDER_ID", "Invalid order ID", map[string]interface{}{"error": err.Error()}))
		return
	}

	err = h.paymentSvc.CancelRechargeOrder(c.Request.Context(), userID, orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("CANCEL_FAILED", "Failed to cancel recharge order", map[string]interface{}{"error": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(nil, "Recharge order cancelled successfully"))
}

// QueryRechargeRecords 查询充值记录
// @Summary 查询充值记录
// @Description 分页查询用户的充值记录
// @Tags 支付
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param status query string false "状态筛选"
// @Param method query string false "支付方式筛选"
// @Success 200 {object} dto.PaginatedRechargeResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /api/v1/payment/recharge [get]
func (h *PaymentHandler) QueryRechargeRecords(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("UNAUTHORIZED", "User not authenticated", nil))
		return
	}

	req := &dto.QueryRechargeRecordsRequest{
		UserID:   &userID,
		Page:     getIntQuery(c, "page", 1),
		PageSize: getIntQuery(c, "page_size", 10),
	}

	// 解析可选参数
	if status := c.Query("status"); status != "" {
		// 这里需要验证状态值的有效性
		req.Status = (*entities.RechargeStatus)(&status)
	}

	if method := c.Query("method"); method != "" {
		req.Method = &method
	}

	response, err := h.paymentSvc.QueryRechargeRecords(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("QUERY_FAILED", "Failed to query recharge records", map[string]interface{}{"error": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(response, "Recharge records retrieved successfully"))
}

// GetPaymentMethods 获取支付方式列表
// @Summary 获取支付方式列表
// @Description 获取可用的支付方式列表（包含服务商信息）
// @Tags 支付
// @Accept json
// @Produce json
// @Param active_only query bool false "是否只获取启用的支付方式" default(true)
// @Success 200 {array} dto.PaymentMethodResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/payment/methods [get]
func (h *PaymentHandler) GetPaymentMethods(c *gin.Context) {
	// 解析查询参数
	activeOnly := true // 默认只获取启用的支付方式
	if activeOnlyStr := c.Query("active_only"); activeOnlyStr != "" {
		if activeOnlyStr == "false" || activeOnlyStr == "0" {
			activeOnly = false
		}
	}

	methods, err := h.paymentSvc.GetPaymentMethods(c.Request.Context(), activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("QUERY_FAILED", "Failed to get payment methods", map[string]interface{}{"error": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(methods, "Payment methods retrieved successfully"))
}

// GetRechargeOptions 获取充值金额选项列表
// @Summary 获取充值金额选项列表
// @Description 获取可用的充值金额选项列表
// @Tags 支付
// @Accept json
// @Produce json
// @Success 200 {array} dto.RechargeOptionResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/payment/recharge-options [get]
func (h *PaymentHandler) GetRechargeOptions(c *gin.Context) {
	options, err := h.paymentSvc.GetRechargeOptions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("QUERY_FAILED", "Failed to get recharge options", map[string]interface{}{"error": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(options, "Recharge options retrieved successfully"))
}

// ProcessPaymentCallback 处理支付回调
// @Summary 处理支付回调
// @Description 处理第三方支付的回调通知
// @Tags 支付
// @Accept json
// @Produce json
// @Param request body dto.PaymentCallbackRequest true "支付回调请求"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/payment/callback [post]
func (h *PaymentHandler) ProcessPaymentCallback(c *gin.Context) {
	var req dto.PaymentCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_REQUEST", "Invalid callback format", map[string]interface{}{"error": err.Error()}))
		return
	}

	err := h.paymentSvc.ProcessPaymentCallback(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"order_no":   req.OrderNo,
			"payment_id": req.PaymentID,
			"status":     req.Status,
			"error":      err.Error(),
		}).Error("Failed to process payment callback")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("CALLBACK_FAILED", "Failed to process payment callback", map[string]interface{}{"error": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(nil, "Payment callback processed successfully"))
}

// GetUserBalance 获取用户余额
// @Summary 获取用户余额
// @Description 获取当前用户的余额信息
// @Tags 支付
// @Accept json
// @Produce json
// @Success 200 {object} dto.BalanceResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/payment/balance [get]
func (h *PaymentHandler) GetUserBalance(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("UNAUTHORIZED", "User not authenticated", nil))
		return
	}

	balance, err := h.transactionSvc.GetUserBalance(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("QUERY_FAILED", "Failed to get user balance", map[string]interface{}{"error": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(balance, "User balance retrieved successfully"))
}

// QueryTransactions 查询交易记录
// @Summary 查询交易记录
// @Description 分页查询用户的交易记录
// @Tags 支付
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param type query string false "交易类型筛选"
// @Success 200 {object} dto.PaginatedTransactionResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /api/v1/payment/transactions [get]
func (h *PaymentHandler) QueryTransactions(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("UNAUTHORIZED", "User not authenticated", nil))
		return
	}

	req := &dto.QueryTransactionsRequest{
		UserID:   &userID,
		Page:     getIntQuery(c, "page", 1),
		PageSize: getIntQuery(c, "page_size", 10),
	}

	// 解析可选参数
	if transactionType := c.Query("type"); transactionType != "" {
		// 这里需要验证交易类型的有效性
		req.Type = (*entities.TransactionType)(&transactionType)
	}

	response, err := h.transactionSvc.QueryTransactions(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("QUERY_FAILED", "Failed to query transactions", map[string]interface{}{"error": err.Error()}))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(response, "Transactions retrieved successfully"))
}

// 辅助方法

// getUserID 从上下文获取用户ID
func getUserID(c *gin.Context) int64 {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(int64); ok {
			return id
		}
	}
	return 0
}

// getIntQuery 获取整数查询参数
func getIntQuery(c *gin.Context, key string, defaultValue int) int {
	if value := c.Query(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
