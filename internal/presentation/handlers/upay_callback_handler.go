package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/domain/services"
	"ai-api-gateway/internal/infrastructure/clients"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

// UPayCallbackHandler UPay回调处理器
type UPayCallbackHandler struct {
	paymentSvc services.PaymentService
	upayClient *clients.UPayClient
	logger     logger.Logger
}

// NewUPayCallbackHandler 创建UPay回调处理器
func NewUPayCallbackHandler(
	paymentSvc services.PaymentService,
	upayClient *clients.UPayClient,
	logger logger.Logger,
) *UPayCallbackHandler {
	return &UPayCallbackHandler{
		paymentSvc: paymentSvc,
		upayClient: upayClient,
		logger:     logger,
	}
}

// HandleCallback 处理UPay支付回调
// @Summary 处理UPay支付回调
// @Description 接收UPay的异步支付通知
// @Tags UPay
// @Accept json
// @Produce json
// @Param request body dto.UPayCallbackRequest true "UPay回调请求"
// @Success 200 {object} dto.UPayCallbackResponse
// @Failure 400 {object} dto.UPayCallbackResponse
// @Failure 500 {object} dto.UPayCallbackResponse
// @Router /api/v1/payment/upay/callback [post]
func (h *UPayCallbackHandler) HandleCallback(c *gin.Context) {
	var req dto.UPayCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"request_ip": c.ClientIP(),
		}).Error("Invalid UPay callback request format")

		c.JSON(http.StatusBadRequest, dto.UPayCallbackResponse{
			Code:    400,
			Message: "Invalid request format",
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"app_id":            req.AppID,
		"order_no":          req.OrderNo,
		"merchant_order_no": req.MerchantOrderNo,
		"status":            req.Status,
		"crypto":            req.Crypto,
		"exchange_rate":     req.ExchangeRate,
		"request_ip":        c.ClientIP(),
	}).Info("Received UPay callback")

	// 签名验证将在统一的回调处理逻辑中进行
	// 这里不再单独验证，因为客户端可能为nil

	// 处理支付回调
	if err := h.processPaymentCallback(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"merchant_order_no": req.MerchantOrderNo,
			"error":             err.Error(),
		}).Error("Failed to process UPay callback")

		c.JSON(http.StatusInternalServerError, dto.UPayCallbackResponse{
			Code:    500,
			Message: "Internal server error",
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"merchant_order_no": req.MerchantOrderNo,
		"status":            req.Status,
	}).Info("UPay callback processed successfully")

	// 返回成功响应
	c.JSON(http.StatusOK, dto.UPayCallbackResponse{
		Code:    200,
		Message: "success",
	})
}

// processPaymentCallback 处理支付回调
func (h *UPayCallbackHandler) processPaymentCallback(req *dto.UPayCallbackRequest) error {
	// 解析金额
	var amount float64
	var err error
	if req.Crypto != "" {
		amount, err = strconv.ParseFloat(req.Crypto, 64)
		if err != nil {
			h.logger.WithFields(map[string]interface{}{
				"crypto": req.Crypto,
				"error":  err.Error(),
			}).Error("Failed to parse crypto amount")
			return err
		}
	}

	// 构建支付回调请求
	now := time.Now()
	callbackReq := &dto.PaymentCallbackRequest{
		OrderNo:   req.MerchantOrderNo, // 使用商户订单号
		PaymentID: req.OrderNo,         // 使用UPay订单号作为支付ID
		Status:    h.mapUPayStatus(req.Status),
		Amount:    amount,
		PaidAt:    &now,
		Signature: req.Signature,
		ExtraData: map[string]interface{}{
			"provider":      "upay",
			"app_id":        req.AppID,
			"upay_order_no": req.OrderNo,
			"exchange_rate": req.ExchangeRate,
			"crypto_amount": req.Crypto,
			"chain_type":    "usdt", // 默认为USDT
		},
	}

	// 调用支付服务处理回调
	ctx := context.Background()
	return h.paymentSvc.ProcessPaymentCallback(ctx, callbackReq)
}

// mapUPayStatus 映射UPay状态到内部状态
func (h *UPayCallbackHandler) mapUPayStatus(upayStatus string) string {
	switch upayStatus {
	case "success", "paid":
		return "success"
	case "failed", "fail":
		return "failed"
	case "cancelled", "cancel":
		return "cancelled"
	case "expired":
		return "expired"
	case "pending":
		return "pending"
	default:
		h.logger.WithField("upay_status", upayStatus).Warn("Unknown UPay status, mapping to failed")
		return "failed"
	}
}

// GetCallbackURL 获取回调URL（用于配置）
func (h *UPayCallbackHandler) GetCallbackURL(baseURL string) string {
	return baseURL + "/api/v1/payment/upay/callback"
}
