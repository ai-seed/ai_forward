package clients

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"ai-api-gateway/internal/infrastructure/logger"
)

// UPayConfig UPay配置
type UPayConfig struct {
	AppID      string `yaml:"app_id"`
	AppSecret  string `yaml:"app_secret"`
	APIBaseURL string `yaml:"api_base_url"` // 从数据库获取，这里保留用于兼容
	NotifyURL  string `yaml:"notify_url"`
}

// UPayClient UPay客户端
type UPayClient struct {
	config     UPayConfig
	httpClient *http.Client
	logger     logger.Logger
}

// NewUPayClient 创建UPay客户端
func NewUPayClient(config UPayConfig, logger logger.Logger) *UPayClient {
	return &UPayClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// ChainType 链路类型
type ChainType string

const (
	ChainTypeTRC20 ChainType = "1" // 波场(TRC20)
	ChainTypeERC20 ChainType = "2" // 以太坊(ERC20)
	ChainTypePYUSD ChainType = "3" // PayPal(PYUSD)
)

// FiatCurrency 法币类型
type FiatCurrency string

const (
	FiatCurrencyUSD FiatCurrency = "USD" // 美元
	FiatCurrencyCNY FiatCurrency = "CNY" // 人民币
	FiatCurrencyJPY FiatCurrency = "JPY" // 日元
	FiatCurrencyKRW FiatCurrency = "KRW" // 韩元
	FiatCurrencyEUR FiatCurrency = "EUR" // 欧元
	FiatCurrencyGBP FiatCurrency = "GBP" // 英镑
)

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	AppID           string       `json:"appId"`
	MerchantOrderNo string       `json:"merchantOrderNo"`
	ChainType       ChainType    `json:"chainType"`
	FiatAmount      string       `json:"fiatAmount"`
	FiatCurrency    FiatCurrency `json:"fiatCurrency"`
	Attach          string       `json:"attach,omitempty"`
	ProductName     string       `json:"productName,omitempty"`
	NotifyURL       string       `json:"notifyUrl"`
	RedirectURL     string       `json:"redirectUrl,omitempty"`
	Signature       string       `json:"signature"`
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	Code    string `json:"code"` // UPay返回的是字符串，不是数字
	Message string `json:"message"`
	Data    struct {
		AppID           string `json:"appId"`
		OrderNo         string `json:"orderNo"`
		MerchantOrderNo string `json:"merchantOrderNo"`
		ExchangeRate    string `json:"exchangeRate"`
		Crypto          string `json:"crypto"`
		Status          string `json:"status"`
		PayURL          string `json:"payUrl"`
	} `json:"data"`
}

// CallbackRequest UPay回调请求
type CallbackRequest struct {
	AppID           string `json:"appId"`
	OrderNo         string `json:"orderNo"`
	MerchantOrderNo string `json:"merchantOrderNo"`
	ExchangeRate    string `json:"exchangeRate"`
	Crypto          string `json:"crypto"`
	Status          string `json:"status"`
	Signature       string `json:"signature"`
}

// CreateOrder 创建支付订单
func (c *UPayClient) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	c.logger.WithFields(map[string]interface{}{
		"merchant_order_no": req.MerchantOrderNo,
		"chain_type":        req.ChainType,
		"fiat_amount":       req.FiatAmount,
		"fiat_currency":     req.FiatCurrency,
		"product_name":      req.ProductName,
	}).Error("UPay CreateOrder called with request")

	// 设置基础参数
	req.AppID = c.config.AppID
	req.NotifyURL = c.config.NotifyURL

	c.logger.WithFields(map[string]interface{}{
		"app_id":     req.AppID,
		"notify_url": req.NotifyURL,
	}).Error("UPay request after setting AppID and NotifyURL")

	// 生成签名
	signature, err := c.generateSignature(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signature: %w", err)
	}
	req.Signature = signature

	c.logger.WithFields(map[string]interface{}{
		"signature": signature,
	}).Error("UPay signature generated")

	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送HTTP请求
	apiURL := c.config.APIBaseURL + "/v1/api/open/order/apply"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	c.logger.WithFields(map[string]interface{}{
		"url":     apiURL,
		"request": string(reqBody),
	}).Info("Sending UPay create order request")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"status":   resp.Status,
		"response": string(respBody),
	}).Info("Received UPay create order response")

	// 解析响应
	var orderResp CreateOrderResponse
	if err := json.Unmarshal(respBody, &orderResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 检查响应状态
	// UPay成功响应的code为"1"
	if orderResp.Code != "1" {
		return nil, fmt.Errorf("UPay API error: code=%s, message=%s", orderResp.Code, orderResp.Message)
	}

	return &orderResp, nil
}

// VerifyCallback 验证回调签名
func (c *UPayClient) VerifyCallback(callback *CallbackRequest) bool {
	// 构建签名参数
	params := map[string]string{
		"appId":           callback.AppID,
		"orderNo":         callback.OrderNo,
		"merchantOrderNo": callback.MerchantOrderNo,
		"exchangeRate":    callback.ExchangeRate,
		"crypto":          callback.Crypto,
		"status":          callback.Status,
	}

	// 生成签名
	expectedSignature := c.generateCallbackSignature(params)

	c.logger.WithFields(map[string]interface{}{
		"expected_signature": expectedSignature,
		"received_signature": callback.Signature,
	}).Debug("Verifying UPay callback signature")

	return expectedSignature == callback.Signature
}

// generateSignature 生成请求签名
func (c *UPayClient) generateSignature(req *CreateOrderRequest) (string, error) {
	// 构建签名参数（只包含需要签名的字段，根据UPay文档）
	// 根据UPay文档，只有以下字段的"是否签名"列为"是"：
	// appId, merchantOrderNo, chainType, fiatAmount, fiatCurrency, notifyUrl
	params := map[string]string{
		"appId":           req.AppID,
		"merchantOrderNo": req.MerchantOrderNo,
		"chainType":       string(req.ChainType),
		"fiatAmount":      req.FiatAmount,
		"fiatCurrency":    string(req.FiatCurrency),
		"notifyUrl":       req.NotifyURL,
	}

	// 注意：根据UPay文档，attach、productName、redirectUrl 都不参与签名
	// 它们的"是否签名"列都标记为"否"

	return c.generateSignatureFromParams(params), nil
}

// generateCallbackSignature 生成回调签名
func (c *UPayClient) generateCallbackSignature(params map[string]string) string {
	return c.generateSignatureFromParams(params)
}

// generateSignatureFromParams 从参数生成签名
func (c *UPayClient) generateSignatureFromParams(params map[string]string) string {
	// 1. 参数排序
	var keys []string
	for k := range params {
		if params[k] != "" { // 排除空值
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// 2. 构建查询字符串
	var parts []string
	for _, k := range keys {
		if params[k] != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
		}
	}
	queryString := strings.Join(parts, "&")

	// 3. 添加密钥
	signString := queryString + "&appSecret=" + c.config.AppSecret

	// 4. MD5加密并转大写
	hash := md5.Sum([]byte(signString))
	signature := fmt.Sprintf("%X", hash)

	c.logger.WithFields(map[string]interface{}{
		"params":       params,
		"sorted_keys":  keys,
		"query_string": queryString,
		"sign_string":  signString,
		"signature":    signature,
	}).Error("Generated UPay signature debug info")

	return signature
}
