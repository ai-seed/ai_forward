package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ai-api-gateway/internal/domain/services"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// BrevoService Brevo邮件服务实现
type BrevoService struct {
	apiKey    string
	apiURL    string
	sender    BrevoSender
	client    *http.Client
	template  services.EmailTemplate
	logger    *logrus.Logger
}

// BrevoSender 发送者信息
type BrevoSender struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// BrevoEmailRequest Brevo发送邮件请求
type BrevoEmailRequest struct {
	Sender      BrevoSender       `json:"sender"`
	To          []BrevoRecipient  `json:"to"`
	Subject     string            `json:"subject"`
	HTMLContent string            `json:"htmlContent,omitempty"`
	TextContent string            `json:"textContent,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// BrevoRecipient 收件人信息
type BrevoRecipient struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// BrevoResponse Brevo API响应
type BrevoResponse struct {
	MessageID string `json:"messageId"`
}

// BrevoError Brevo API错误响应
type BrevoError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// NewBrevoService 创建Brevo邮件服务
func NewBrevoService(template services.EmailTemplate, logger *logrus.Logger) *BrevoService {
	timeout := viper.GetDuration("email.brevo.timeout")
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &BrevoService{
		apiKey: viper.GetString("email.brevo.api_key"),
		apiURL: viper.GetString("email.brevo.api_url"),
		sender: BrevoSender{
			Email: viper.GetString("email.brevo.sender.email"),
			Name:  viper.GetString("email.brevo.sender.name"),
		},
		client: &http.Client{
			Timeout: timeout,
		},
		template: template,
		logger:   logger,
	}
}

// SendVerificationCode 发送验证码邮件
func (s *BrevoService) SendVerificationCode(ctx context.Context, to, code string) error {
	subject, htmlBody, textBody := s.template.RenderVerificationCode(code)
	
	req := &services.SendEmailRequest{
		To:       to,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
	
	return s.SendEmail(ctx, req)
}

// SendPasswordResetCode 发送密码重置验证码邮件
func (s *BrevoService) SendPasswordResetCode(ctx context.Context, to, code string) error {
	subject, htmlBody, textBody := s.template.RenderPasswordResetCode(code)
	
	req := &services.SendEmailRequest{
		To:       to,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	}
	
	return s.SendEmail(ctx, req)
}

// SendEmail 发送通用邮件
func (s *BrevoService) SendEmail(ctx context.Context, req *services.SendEmailRequest) error {
	brevoReq := &BrevoEmailRequest{
		Sender: s.sender,
		To: []BrevoRecipient{
			{
				Email: req.To,
			},
		},
		Subject:     req.Subject,
		HTMLContent: req.HTMLBody,
		TextContent: req.TextBody,
		Headers:     req.Headers,
	}

	return s.sendWithRetry(ctx, brevoReq)
}

// sendWithRetry 带重试的发送邮件
func (s *BrevoService) sendWithRetry(ctx context.Context, req *BrevoEmailRequest) error {
	maxAttempts := viper.GetInt("email.brevo.retry.max_attempts")
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	
	retryDelay := viper.GetDuration("email.brevo.retry.delay")
	if retryDelay == 0 {
		retryDelay = time.Second
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := s.sendEmail(ctx, req)
		if err == nil {
			return nil
		}
		
		lastErr = err
		s.logger.WithFields(logrus.Fields{
			"attempt": attempt,
			"error":   err.Error(),
			"to":      req.To[0].Email,
		}).Warn("Failed to send email, retrying...")
		
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				// 继续重试
			}
		}
	}
	
	return fmt.Errorf("failed to send email after %d attempts: %w", maxAttempts, lastErr)
}

// sendEmail 发送邮件到Brevo API
func (s *BrevoService) sendEmail(ctx context.Context, req *BrevoEmailRequest) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.apiURL+"/smtp/email", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", s.apiKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var brevoResp BrevoResponse
		if err := json.Unmarshal(body, &brevoResp); err != nil {
			s.logger.WithFields(logrus.Fields{
				"to":       req.To[0].Email,
				"response": string(body),
			}).Warn("Email sent but failed to parse response")
		} else {
			s.logger.WithFields(logrus.Fields{
				"to":         req.To[0].Email,
				"message_id": brevoResp.MessageID,
			}).Info("Email sent successfully")
		}
		return nil
	}

	var brevoErr BrevoError
	if err := json.Unmarshal(body, &brevoErr); err != nil {
		return fmt.Errorf("email send failed with status %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Errorf("email send failed: %s (code: %s)", brevoErr.Message, brevoErr.Code)
}
