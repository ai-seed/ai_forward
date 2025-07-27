package dto

import (
	"time"
)

// LoginRequest 用户登录请求
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6,max=100"`
}

// LoginResponse 用户登录响应
type LoginResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	TokenType    string   `json:"token_type"`
	ExpiresIn    int64    `json:"expires_in"`
	User         UserInfo `json:"user"`
}

// RefreshTokenRequest 刷新Token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshTokenResponse 刷新Token响应
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// UserInfo 用户信息（用于认证响应）
type UserInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"full_name,omitempty"`
}

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Username string `json:"username" validate:"omitempty,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=100"`
}

// RegisterResponse 用户注册响应
type RegisterResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name,omitempty"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6,max=100"`
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

// GetUserProfileResponse 获取用户资料响应
type GetUserProfileResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name,omitempty"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SendVerificationCodeRequest 发送验证码请求
type SendVerificationCodeRequest struct {
	Email string `json:"email" validate:"required,email"`
	Type  string `json:"type" validate:"required,oneof=register password_reset"`
}

// SendVerificationCodeResponse 发送验证码响应
type SendVerificationCodeResponse struct {
	Message   string `json:"message"`
	ExpiresIn int    `json:"expires_in"` // 过期时间（秒）
}

// VerifyCodeRequest 验证验证码请求
type VerifyCodeRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,len=6"`
	Type  string `json:"type" validate:"required,oneof=register password_reset"`
}

// VerifyCodeResponse 验证验证码响应
type VerifyCodeResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

// RegisterWithCodeRequest 带验证码的注册请求
type RegisterWithCodeRequest struct {
	Username         string `json:"username" validate:"omitempty,min=3,max=50"`
	Email            string `json:"email" validate:"required,email"`
	Password         string `json:"password" validate:"required,min=6,max=100"`
	VerificationCode string `json:"verification_code" validate:"required,len=6"`
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Email            string `json:"email" validate:"required,email"`
	NewPassword      string `json:"new_password" validate:"required,min=6,max=100"`
	VerificationCode string `json:"verification_code" validate:"required,len=6"`
}

// ResetPasswordResponse 重置密码响应
type ResetPasswordResponse struct {
	Message string `json:"message"`
}
