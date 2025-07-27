package services

import (
	"context"
	"time"
)

// VerificationService 验证码服务接口
type VerificationService interface {
	// GenerateCode 生成验证码
	GenerateCode(ctx context.Context, email string, codeType VerificationCodeType) (string, error)
	
	// VerifyCode 验证验证码
	VerifyCode(ctx context.Context, email, code string, codeType VerificationCodeType) error
	
	// CanSendCode 检查是否可以发送验证码（防止频繁发送）
	CanSendCode(ctx context.Context, email string, codeType VerificationCodeType) error
	
	// InvalidateCode 使验证码失效
	InvalidateCode(ctx context.Context, email string, codeType VerificationCodeType) error
}

// VerificationCodeType 验证码类型
type VerificationCodeType string

const (
	// VerificationCodeTypeRegister 注册验证码
	VerificationCodeTypeRegister VerificationCodeType = "register"
	
	// VerificationCodeTypePasswordReset 密码重置验证码
	VerificationCodeTypePasswordReset VerificationCodeType = "password_reset"
)

// VerificationCodeInfo 验证码信息
type VerificationCodeInfo struct {
	Code      string                   `json:"code"`      // 验证码
	Email     string                   `json:"email"`     // 邮箱
	Type      VerificationCodeType     `json:"type"`      // 类型
	CreatedAt time.Time                `json:"createdAt"` // 创建时间
	ExpiresAt time.Time                `json:"expiresAt"` // 过期时间
	Attempts  int                      `json:"attempts"`  // 尝试次数
}

// VerificationError 验证码错误
type VerificationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *VerificationError) Error() string {
	return e.Message
}

// 预定义的验证码错误
var (
	ErrCodeExpired     = &VerificationError{Code: "CODE_EXPIRED", Message: "Verification code has expired"}
	ErrCodeInvalid     = &VerificationError{Code: "CODE_INVALID", Message: "Invalid verification code"}
	ErrCodeNotFound    = &VerificationError{Code: "CODE_NOT_FOUND", Message: "Verification code not found"}
	ErrTooManyAttempts = &VerificationError{Code: "TOO_MANY_ATTEMPTS", Message: "Too many verification attempts"}
	ErrSendTooFrequent = &VerificationError{Code: "SEND_TOO_FREQUENT", Message: "Verification code sent too frequently"}
)
