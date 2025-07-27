package verification

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"ai-api-gateway/internal/domain/services"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// RedisVerificationService Redis验证码服务实现
type RedisVerificationService struct {
	client *redis.Client
	logger *logrus.Logger
}

// NewRedisVerificationService 创建Redis验证码服务
func NewRedisVerificationService(client *redis.Client, logger *logrus.Logger) *RedisVerificationService {
	return &RedisVerificationService{
		client: client,
		logger: logger,
	}
}

// GenerateCode 生成验证码
func (s *RedisVerificationService) GenerateCode(ctx context.Context, email string, codeType services.VerificationCodeType) (string, error) {
	// 检查是否可以发送验证码
	if err := s.CanSendCode(ctx, email, codeType); err != nil {
		return "", err
	}

	// 生成验证码
	code, err := s.generateRandomCode()
	if err != nil {
		return "", fmt.Errorf("failed to generate code: %w", err)
	}

	// 获取过期时间
	expireTime := viper.GetDuration("verification.expire_time")
	if expireTime == 0 {
		expireTime = 10 * time.Minute
	}

	// 创建验证码信息
	now := time.Now()
	codeInfo := &services.VerificationCodeInfo{
		Code:      code,
		Email:     email,
		Type:      codeType,
		CreatedAt: now,
		ExpiresAt: now.Add(expireTime),
		Attempts:  0,
	}

	// 存储到Redis
	key := s.getCodeKey(email, codeType)
	data, err := json.Marshal(codeInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal code info: %w", err)
	}

	if err := s.client.Set(ctx, key, data, expireTime).Err(); err != nil {
		return "", fmt.Errorf("failed to store code: %w", err)
	}

	// 设置发送间隔限制
	sendIntervalKey := s.getSendIntervalKey(email, codeType)
	sendInterval := viper.GetDuration("verification.send_interval")
	if sendInterval == 0 {
		sendInterval = time.Minute
	}
	
	if err := s.client.Set(ctx, sendIntervalKey, "1", sendInterval).Err(); err != nil {
		s.logger.WithFields(logrus.Fields{
			"email": email,
			"type":  codeType,
			"error": err.Error(),
		}).Warn("Failed to set send interval limit")
	}

	s.logger.WithFields(logrus.Fields{
		"email": email,
		"type":  codeType,
	}).Info("Verification code generated")

	return code, nil
}

// VerifyCode 验证验证码
func (s *RedisVerificationService) VerifyCode(ctx context.Context, email, code string, codeType services.VerificationCodeType) error {
	key := s.getCodeKey(email, codeType)
	
	// 获取验证码信息
	data, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return services.ErrCodeNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to get code: %w", err)
	}

	var codeInfo services.VerificationCodeInfo
	if err := json.Unmarshal([]byte(data), &codeInfo); err != nil {
		return fmt.Errorf("failed to unmarshal code info: %w", err)
	}

	// 检查是否过期
	if time.Now().After(codeInfo.ExpiresAt) {
		// 删除过期的验证码
		s.client.Del(ctx, key)
		return services.ErrCodeExpired
	}

	// 检查尝试次数
	maxAttempts := viper.GetInt("verification.max_attempts")
	if maxAttempts == 0 {
		maxAttempts = 5
	}
	
	if codeInfo.Attempts >= maxAttempts {
		// 删除超过尝试次数的验证码
		s.client.Del(ctx, key)
		return services.ErrTooManyAttempts
	}

	// 验证验证码
	if codeInfo.Code != code {
		// 增加尝试次数
		codeInfo.Attempts++
		data, _ := json.Marshal(codeInfo)
		s.client.Set(ctx, key, data, time.Until(codeInfo.ExpiresAt))
		return services.ErrCodeInvalid
	}

	// 验证成功，删除验证码
	s.client.Del(ctx, key)
	
	s.logger.WithFields(logrus.Fields{
		"email": email,
		"type":  codeType,
	}).Info("Verification code verified successfully")

	return nil
}

// CanSendCode 检查是否可以发送验证码
func (s *RedisVerificationService) CanSendCode(ctx context.Context, email string, codeType services.VerificationCodeType) error {
	sendIntervalKey := s.getSendIntervalKey(email, codeType)
	
	exists, err := s.client.Exists(ctx, sendIntervalKey).Result()
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"email": email,
			"type":  codeType,
			"error": err.Error(),
		}).Warn("Failed to check send interval")
		// 如果检查失败，允许发送
		return nil
	}
	
	if exists > 0 {
		return services.ErrSendTooFrequent
	}
	
	return nil
}

// InvalidateCode 使验证码失效
func (s *RedisVerificationService) InvalidateCode(ctx context.Context, email string, codeType services.VerificationCodeType) error {
	key := s.getCodeKey(email, codeType)
	return s.client.Del(ctx, key).Err()
}

// generateRandomCode 生成随机验证码
func (s *RedisVerificationService) generateRandomCode() (string, error) {
	codeLength := viper.GetInt("verification.code_length")
	if codeLength == 0 {
		codeLength = 6
	}
	
	codeType := viper.GetString("verification.code_type")
	if codeType == "" {
		codeType = "numeric"
	}

	var charset string
	switch codeType {
	case "numeric":
		charset = "0123456789"
	case "alphanumeric":
		charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	default:
		charset = "0123456789"
	}

	code := make([]byte, codeLength)
	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[num.Int64()]
	}

	return string(code), nil
}

// getCodeKey 获取验证码存储键
func (s *RedisVerificationService) getCodeKey(email string, codeType services.VerificationCodeType) string {
	return fmt.Sprintf("verification:code:%s:%s", codeType, strings.ToLower(email))
}

// getSendIntervalKey 获取发送间隔限制键
func (s *RedisVerificationService) getSendIntervalKey(email string, codeType services.VerificationCodeType) string {
	return fmt.Sprintf("verification:interval:%s:%s", codeType, strings.ToLower(email))
}
