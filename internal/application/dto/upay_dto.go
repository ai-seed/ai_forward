package dto

import "time"

// UPayCreateOrderRequest UPay创建订单请求
type UPayCreateOrderRequest struct {
	Amount       float64 `json:"amount" binding:"required,gt=0"`
	Currency     string  `json:"currency" binding:"required"`
	ChainType    string  `json:"chain_type" binding:"required"`
	ProductName  string  `json:"product_name,omitempty"`
	RedirectURL  string  `json:"redirect_url,omitempty"`
	Attach       string  `json:"attach,omitempty"`
}

// UPayCreateOrderResponse UPay创建订单响应
type UPayCreateOrderResponse struct {
	OrderNo         string  `json:"order_no"`
	MerchantOrderNo string  `json:"merchant_order_no"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	ChainType       string  `json:"chain_type"`
	ExchangeRate    string  `json:"exchange_rate"`
	CryptoAmount    string  `json:"crypto_amount"`
	PayURL          string  `json:"pay_url"`
	Status          string  `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	ExpiredAt       *time.Time `json:"expired_at,omitempty"`
}

// UPayCallbackRequest UPay回调请求
type UPayCallbackRequest struct {
	AppID            string `json:"appId" binding:"required"`
	OrderNo          string `json:"orderNo" binding:"required"`
	MerchantOrderNo  string `json:"merchantOrderNo" binding:"required"`
	ExchangeRate     string `json:"exchangeRate"`
	Crypto           string `json:"crypto"`
	Status           string `json:"status" binding:"required"`
	Signature        string `json:"signature" binding:"required"`
}

// UPayCallbackResponse UPay回调响应
type UPayCallbackResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// UPayOrderStatus UPay订单状态
type UPayOrderStatus string

const (
	UPayOrderStatusPending   UPayOrderStatus = "pending"   // 待支付
	UPayOrderStatusSuccess   UPayOrderStatus = "success"   // 支付成功
	UPayOrderStatusFailed    UPayOrderStatus = "failed"    // 支付失败
	UPayOrderStatusCancelled UPayOrderStatus = "cancelled" // 已取消
	UPayOrderStatusExpired   UPayOrderStatus = "expired"   // 已过期
)

// UPayChainType UPay链路类型
type UPayChainType string

const (
	UPayChainTypeTRC20 UPayChainType = "1" // 波场(TRC20)
	UPayChainTypeERC20 UPayChainType = "2" // 以太坊(ERC20)
	UPayChainTypePYUSD UPayChainType = "3" // PayPal(PYUSD)
)

// UPayFiatCurrency UPay支持的法币
type UPayFiatCurrency string

const (
	UPayFiatCurrencyUSD UPayFiatCurrency = "USD" // 美元
	UPayFiatCurrencyCNY UPayFiatCurrency = "CNY" // 人民币
	UPayFiatCurrencyJPY UPayFiatCurrency = "JPY" // 日元
	UPayFiatCurrencyKRW UPayFiatCurrency = "KRW" // 韩元
	UPayFiatCurrencyPHP UPayFiatCurrency = "PHP" // 比索
	UPayFiatCurrencyEUR UPayFiatCurrency = "EUR" // 欧元
	UPayFiatCurrencyGBP UPayFiatCurrency = "GBP" // 英镑
	UPayFiatCurrencyCHF UPayFiatCurrency = "CHF" // 瑞士法郎
	UPayFiatCurrencyTWD UPayFiatCurrency = "TWD" // 新台币
	UPayFiatCurrencyHKD UPayFiatCurrency = "HKD" // 港币
	UPayFiatCurrencyMOP UPayFiatCurrency = "MOP" // 澳门元
	UPayFiatCurrencySGD UPayFiatCurrency = "SGD" // 新加坡币
	UPayFiatCurrencyNZD UPayFiatCurrency = "NZD" // 新西兰元
	UPayFiatCurrencyTHB UPayFiatCurrency = "THB" // 泰铢
	UPayFiatCurrencyCAD UPayFiatCurrency = "CAD" // 加拿大元
	UPayFiatCurrencyZAR UPayFiatCurrency = "ZAR" // 南非兰特
	UPayFiatCurrencyBRL UPayFiatCurrency = "BRL" // 巴西雷亚尔
	UPayFiatCurrencyINR UPayFiatCurrency = "INR" // 卢比
)

// GetUPayChainTypeName 获取链路类型名称
func GetUPayChainTypeName(chainType UPayChainType) string {
	switch chainType {
	case UPayChainTypeTRC20:
		return "TRC20 (波场)"
	case UPayChainTypeERC20:
		return "ERC20 (以太坊)"
	case UPayChainTypePYUSD:
		return "PYUSD (PayPal)"
	default:
		return "未知链路"
	}
}

// GetUPayFiatCurrencyName 获取法币名称
func GetUPayFiatCurrencyName(currency UPayFiatCurrency) string {
	switch currency {
	case UPayFiatCurrencyUSD:
		return "美元"
	case UPayFiatCurrencyCNY:
		return "人民币"
	case UPayFiatCurrencyJPY:
		return "日元"
	case UPayFiatCurrencyKRW:
		return "韩元"
	case UPayFiatCurrencyPHP:
		return "比索"
	case UPayFiatCurrencyEUR:
		return "欧元"
	case UPayFiatCurrencyGBP:
		return "英镑"
	case UPayFiatCurrencyCHF:
		return "瑞士法郎"
	case UPayFiatCurrencyTWD:
		return "新台币"
	case UPayFiatCurrencyHKD:
		return "港币"
	case UPayFiatCurrencyMOP:
		return "澳门元"
	case UPayFiatCurrencySGD:
		return "新加坡币"
	case UPayFiatCurrencyNZD:
		return "新西兰元"
	case UPayFiatCurrencyTHB:
		return "泰铢"
	case UPayFiatCurrencyCAD:
		return "加拿大元"
	case UPayFiatCurrencyZAR:
		return "南非兰特"
	case UPayFiatCurrencyBRL:
		return "巴西雷亚尔"
	case UPayFiatCurrencyINR:
		return "卢比"
	default:
		return string(currency)
	}
}

// IsValidUPayChainType 验证链路类型是否有效
func IsValidUPayChainType(chainType string) bool {
	switch UPayChainType(chainType) {
	case UPayChainTypeTRC20, UPayChainTypeERC20, UPayChainTypePYUSD:
		return true
	default:
		return false
	}
}

// IsValidUPayFiatCurrency 验证法币类型是否有效
func IsValidUPayFiatCurrency(currency string) bool {
	switch UPayFiatCurrency(currency) {
	case UPayFiatCurrencyUSD, UPayFiatCurrencyCNY, UPayFiatCurrencyJPY,
		 UPayFiatCurrencyKRW, UPayFiatCurrencyPHP, UPayFiatCurrencyEUR,
		 UPayFiatCurrencyGBP, UPayFiatCurrencyCHF, UPayFiatCurrencyTWD,
		 UPayFiatCurrencyHKD, UPayFiatCurrencyMOP, UPayFiatCurrencySGD,
		 UPayFiatCurrencyNZD, UPayFiatCurrencyTHB, UPayFiatCurrencyCAD,
		 UPayFiatCurrencyZAR, UPayFiatCurrencyBRL, UPayFiatCurrencyINR:
		return true
	default:
		return false
	}
}
