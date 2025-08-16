package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	if orderNo := c.Query("order_no"); orderNo != "" {
		req.OrderNo = &orderNo
	}

	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse("2006-01-02", startTime); err == nil {
			req.StartTime = &t
		}
	}

	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse("2006-01-02", endTime); err == nil {
			// 设置为当天的23:59:59
			endOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
			req.EndTime = &endOfDay
		}
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

// ProcessUnifiedPaymentCallback 处理统一格式的支付回调
// @Summary 处理统一格式的支付回调
// @Description 处理第三方支付的回调通知，使用统一的URL格式 /callback/:orderNo
// @Tags 支付
// @Accept json
// @Produce json
// @Param orderNo path string true "商户订单号"
// @Param request body dto.PaymentCallbackRequest true "支付回调请求"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/payment/callback/{orderNo} [post]
func (h *PaymentHandler) ProcessUnifiedPaymentCallback(c *gin.Context) {
	// 从URL路径中获取订单号
	orderNo := c.Param("orderNo")
	if orderNo == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_REQUEST", "Order number is required in URL path", nil))
		return
	}

	var req dto.PaymentCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_REQUEST", "Invalid callback format", map[string]interface{}{"error": err.Error()}))
		return
	}

	// 如果请求体中没有订单号，使用URL中的订单号
	if req.OrderNo == "" {
		req.OrderNo = orderNo
	}

	// 验证URL中的订单号与请求体中的订单号是否一致（如果请求体中有的话）
	if req.OrderNo != orderNo {
		h.logger.WithFields(map[string]interface{}{
			"url_order_no":  orderNo,
			"body_order_no": req.OrderNo,
		}).Warn("Order number mismatch between URL and request body")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_REQUEST", "Order number mismatch", nil))
		return
	}

	err := h.paymentSvc.ProcessPaymentCallback(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"order_no":   req.OrderNo,
			"payment_id": req.PaymentID,
			"status":     req.Status,
			"error":      err.Error(),
		}).Error("Failed to process unified payment callback")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("CALLBACK_FAILED", "Failed to process payment callback", nil))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"order_no":   req.OrderNo,
		"payment_id": req.PaymentID,
		"status":     req.Status,
	}).Info("Unified payment callback processed successfully")

	c.JSON(http.StatusOK, dto.SuccessResponse(nil, "Payment callback processed successfully"))
}

// ProcessPaymentCallback 处理支付回调（兼容旧格式）
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

// GetPaymentPage 获取支付页面信息
// @Summary 获取支付页面信息
// @Description 获取支付页面的订单信息，用于展示支付页面
// @Tags 支付
// @Accept json
// @Produce json
// @Param order_no query string true "订单号"
// @Success 200 {object} dto.SuccessResponse{data=dto.PaymentPageResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/payment/pay [get]
func (h *PaymentHandler) GetPaymentPage(c *gin.Context) {
	orderNo := c.Query("order_no")
	if orderNo == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_REQUEST", "Order number is required", nil))
		return
	}

	pageInfo, err := h.paymentSvc.GetPaymentPage(c.Request.Context(), orderNo)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"order_no": orderNo,
			"error":    err.Error(),
		}).Error("Failed to get payment page")

		if err.Error() == "recharge order not found: record not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("ORDER_NOT_FOUND", "Payment order not found", nil))
		} else if err.Error() == "payment order has expired" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("ORDER_EXPIRED", "Payment order has expired", nil))
		} else if strings.Contains(err.Error(), "not pending") {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("ORDER_NOT_PENDING", "Payment order is not pending", nil))
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("GET_PAYMENT_PAGE_FAILED", "Failed to get payment page", nil))
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(pageInfo, "Payment page info retrieved successfully"))
}

// SimulateQRCodePayment 模拟扫码支付
// @Summary 模拟扫码支付
// @Description 模拟用户扫码支付，调用支付回调接口完成支付
// @Tags 支付
// @Accept json
// @Produce json
// @Param order_no query string true "订单号"
// @Success 200 {object} dto.SuccessResponse{data=dto.PaymentResultResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/payment/qr-pay [post]
func (h *PaymentHandler) SimulateQRCodePayment(c *gin.Context) {
	orderNo := c.Query("order_no")
	if orderNo == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_REQUEST", "Order number is required", nil))
		return
	}

	// 获取订单信息
	pageInfo, err := h.paymentSvc.GetPaymentPage(c.Request.Context(), orderNo)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"order_no": orderNo,
			"error":    err.Error(),
		}).Error("Failed to get payment page for QR payment")

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("ORDER_NOT_FOUND", "Payment order not found", nil))
		} else if strings.Contains(err.Error(), "expired") {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("ORDER_EXPIRED", "Payment order has expired", nil))
		} else if strings.Contains(err.Error(), "not pending") {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("ORDER_NOT_PENDING", "Payment order is not pending", nil))
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("GET_ORDER_FAILED", "Failed to get order info", nil))
		}
		return
	}

	// 生成模拟支付数据
	paymentID := fmt.Sprintf("QR_PAY_%d_%d", time.Now().Unix(), time.Now().Nanosecond()%10000)
	now := time.Now()

	// 构建支付回调请求
	callbackReq := dto.PaymentCallbackRequest{
		OrderNo:   orderNo,
		PaymentID: paymentID,
		Status:    "success",
		Amount:    pageInfo.Amount,
		PaidAt:    &now,
		Signature: fmt.Sprintf("qr_payment_%s_%s", orderNo, paymentID),
		ExtraData: map[string]interface{}{
			"provider":     "qr_simulator",
			"payment_type": "qr_code",
			"simulated":    true,
		},
	}

	// 处理支付回调
	err = h.paymentSvc.ProcessPaymentCallback(c.Request.Context(), &callbackReq)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"order_no":   orderNo,
			"payment_id": paymentID,
			"error":      err.Error(),
		}).Error("Failed to process QR payment callback")

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse("QR_PAYMENT_FAILED", "Failed to process QR payment", nil))
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"order_no":   orderNo,
		"payment_id": paymentID,
		"amount":     pageInfo.Amount,
	}).Info("QR payment simulated successfully")

	// 返回支付结果
	result := dto.PaymentResultResponse{
		OrderNo:      orderNo,
		Status:       "success",
		Amount:       pageInfo.Amount,
		ActualAmount: pageInfo.ActualAmount,
		PaymentID:    paymentID,
		PaidAt:       now,
		Message:      "QR payment completed successfully",
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(result, "QR payment completed successfully"))
}

// ProcessPayment 处理支付确认
// @Summary 处理支付确认
// @Description 确认支付并更新订单状态，充值用户余额
// @Tags 支付
// @Accept json
// @Produce json
// @Param order_no query string true "订单号"
// @Success 200 {object} dto.SuccessResponse{data=dto.PaymentResultResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/payment/pay [post]
func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	orderNo := c.Query("order_no")
	if orderNo == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_REQUEST", "Order number is required", nil))
		return
	}

	result, err := h.paymentSvc.ProcessPayment(c.Request.Context(), orderNo)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"order_no": orderNo,
			"error":    err.Error(),
		}).Error("Failed to process payment")

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("ORDER_NOT_FOUND", "Payment order not found", nil))
		} else if strings.Contains(err.Error(), "expired") {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("ORDER_EXPIRED", "Payment order has expired", nil))
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("PROCESS_PAYMENT_FAILED", "Failed to process payment", nil))
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(result, "Payment processed successfully"))
}

// SimulatePaymentSuccess 模拟支付成功
// @Summary 模拟支付成功
// @Description 模拟第三方支付成功，用于测试环境
// @Tags 支付
// @Accept json
// @Produce json
// @Param order_no query string true "订单号"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/payment/simulate-success [post]
func (h *PaymentHandler) SimulatePaymentSuccess(c *gin.Context) {
	orderNo := c.Query("order_no")
	if orderNo == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("INVALID_REQUEST", "Order number is required", nil))
		return
	}

	err := h.paymentSvc.SimulatePaymentSuccess(c.Request.Context(), orderNo)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"order_no": orderNo,
			"error":    err.Error(),
		}).Error("Failed to simulate payment success")

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, dto.ErrorResponse("ORDER_NOT_FOUND", "Payment order not found", nil))
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse("SIMULATE_PAYMENT_FAILED", "Failed to simulate payment success", nil))
		}
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(nil, "Payment simulated successfully"))
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
