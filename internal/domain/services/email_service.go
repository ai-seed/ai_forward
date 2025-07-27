package services

import (
	"context"
)

// EmailService 邮件服务接口
type EmailService interface {
	// SendVerificationCode 发送验证码邮件
	SendVerificationCode(ctx context.Context, to, code string) error
	
	// SendPasswordResetCode 发送密码重置验证码邮件
	SendPasswordResetCode(ctx context.Context, to, code string) error
	
	// SendEmail 发送通用邮件
	SendEmail(ctx context.Context, req *SendEmailRequest) error
}

// SendEmailRequest 发送邮件请求
type SendEmailRequest struct {
	To       string            `json:"to"`       // 收件人邮箱
	Subject  string            `json:"subject"`  // 邮件主题
	HTMLBody string            `json:"htmlBody"` // HTML邮件内容
	TextBody string            `json:"textBody"` // 纯文本邮件内容（可选）
	Headers  map[string]string `json:"headers"`  // 自定义邮件头（可选）
}

// EmailTemplate 邮件模板接口
type EmailTemplate interface {
	// RenderVerificationCode 渲染验证码邮件模板
	RenderVerificationCode(code string) (subject, htmlBody, textBody string)
	
	// RenderPasswordResetCode 渲染密码重置邮件模板
	RenderPasswordResetCode(code string) (subject, htmlBody, textBody string)
}
