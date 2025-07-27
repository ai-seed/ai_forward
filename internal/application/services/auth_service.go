package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/domain/services"

	"golang.org/x/crypto/bcrypt"
)

// 类型别名
type EmailService = services.EmailService
type VerificationService = services.VerificationService

// AuthService 认证服务接口
type AuthService interface {
	// Login 用户登录
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error)

	// Register 用户注册
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error)

	// RefreshToken 刷新令牌
	RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error)

	// ChangePassword 修改密码
	ChangePassword(ctx context.Context, userID int64, req *dto.ChangePasswordRequest) error

	// GetUserProfile 获取用户资料
	GetUserProfile(ctx context.Context, userID int64) (*dto.GetUserProfileResponse, error)

	// Recharge 用户充值
	Recharge(ctx context.Context, userID int64, req *dto.UserRechargeRequest) (*dto.GetUserProfileResponse, error)

	// ValidateUser 验证用户凭据
	ValidateUser(ctx context.Context, username, password string) (*entities.User, error)

	// SendVerificationCode 发送验证码
	SendVerificationCode(ctx context.Context, req *dto.SendVerificationCodeRequest) (*dto.SendVerificationCodeResponse, error)

	// VerifyCode 验证验证码
	VerifyCode(ctx context.Context, req *dto.VerifyCodeRequest) (*dto.VerifyCodeResponse, error)

	// RegisterWithCode 带验证码的注册
	RegisterWithCode(ctx context.Context, req *dto.RegisterWithCodeRequest) (*dto.RegisterResponse, error)

	// ResetPassword 重置密码
	ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) (*dto.ResetPasswordResponse, error)
}

// authServiceImpl 认证服务实现
type authServiceImpl struct {
	userRepo            repositories.UserRepository
	jwtService          JWTService
	emailService        EmailService
	verificationService VerificationService
}

// NewAuthService 创建认证服务
func NewAuthService(userRepo repositories.UserRepository, jwtService JWTService, emailService EmailService, verificationService VerificationService) AuthService {
	return &authServiceImpl{
		userRepo:            userRepo,
		jwtService:          jwtService,
		emailService:        emailService,
		verificationService: verificationService,
	}
}

// Login 用户登录（使用邮箱+密码）
func (s *authServiceImpl) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	// 验证用户凭据（使用邮箱作为用户名）
	user, err := s.ValidateUserByEmail(ctx, req.Username, req.Password)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	// 检查用户状态
	if !user.IsActive() {
		return nil, fmt.Errorf("user account is not active")
	}

	// 生成JWT令牌
	accessToken, refreshToken, err := s.jwtService.GenerateTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// 构造响应
	response := &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    24 * 60 * 60, // 24小时，应该从配置读取
		User: dto.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			FullName: "",
		},
	}

	if user.FullName != nil {
		response.User.FullName = *user.FullName
	}

	return response, nil
}

// Register 用户注册
func (s *authServiceImpl) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// 检查邮箱是否已存在
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("email already exists")
	}
	// 如果错误不是"用户不存在"，则返回错误
	if err != nil && err != entities.ErrUserNotFound {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}

	// 处理用户名：如果为空则自动生成
	username := req.Username
	if username == "" {
		username = s.generateUsername()
	}

	// 哈希密码
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 创建用户实体
	user := &entities.User{
		Username:     username,
		Email:        req.Email,
		PasswordHash: &hashedPassword,
		Status:       entities.UserStatusActive,
		Balance:      0.0,
	}

	// 保存用户
	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 构造响应
	response := &dto.RegisterResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  "",
		Message:   "User registered successfully",
		CreatedAt: user.CreatedAt,
	}

	return response, nil
}

// RefreshToken 刷新令牌
func (s *authServiceImpl) RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error) {
	// 使用JWT服务刷新令牌
	newAccessToken, newRefreshToken, err := s.jwtService.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh tokens: %w", err)
	}

	return &dto.RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    24 * 60 * 60, // 24小时，应该从配置读取
	}, nil
}

// ChangePassword 修改密码
func (s *authServiceImpl) ChangePassword(ctx context.Context, userID int64, req *dto.ChangePasswordRequest) error {
	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// 验证旧密码
	if user.PasswordHash == nil {
		return fmt.Errorf("user has no password set")
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.OldPassword))
	if err != nil {
		return fmt.Errorf("old password is incorrect")
	}

	// 哈希新密码
	newHashedPassword, err := s.hashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// 更新密码
	user.PasswordHash = &newHashedPassword
	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// GetUserProfile 获取用户资料
func (s *authServiceImpl) GetUserProfile(ctx context.Context, userID int64) (*dto.GetUserProfileResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	response := &dto.GetUserProfileResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Balance:   user.Balance,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	if user.FullName != nil {
		response.FullName = *user.FullName
	}

	return response, nil
}

// Recharge 用户充值
func (s *authServiceImpl) Recharge(ctx context.Context, userID int64, req *dto.UserRechargeRequest) (*dto.GetUserProfileResponse, error) {
	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 添加余额
	if err := user.AddBalance(req.Amount); err != nil {
		return nil, fmt.Errorf("failed to add balance: %w", err)
	}

	// 更新数据库中的余额
	if err := s.userRepo.UpdateBalance(ctx, user.ID, user.Balance); err != nil {
		return nil, fmt.Errorf("failed to update user balance: %w", err)
	}

	// 返回更新后的用户资料
	response := &dto.GetUserProfileResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Balance:   user.Balance,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	if user.FullName != nil {
		response.FullName = *user.FullName
	}

	return response, nil
}

// ValidateUser 验证用户凭据（通过用户名）
func (s *authServiceImpl) ValidateUser(ctx context.Context, username, password string) (*entities.User, error) {
	// 根据用户名获取用户
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// 检查密码哈希是否存在
	if user.PasswordHash == nil {
		return nil, fmt.Errorf("user has no password set")
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

// ValidateUserByEmail 验证用户凭据（通过邮箱）
func (s *authServiceImpl) ValidateUserByEmail(ctx context.Context, email, password string) (*entities.User, error) {
	// 根据邮箱获取用户
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// 检查密码哈希是否存在
	if user.PasswordHash == nil {
		return nil, fmt.Errorf("user has no password set")
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

// hashPassword 哈希密码
func (s *authServiceImpl) hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// generateUsername 生成随机用户名
func (s *authServiceImpl) generateUsername() string {
	// 使用当前时间作为随机种子
	rand.Seed(time.Now().UnixNano())
	// 生成6位随机数字
	randomNum := rand.Intn(999999)
	return fmt.Sprintf("718_%06d", randomNum)
}

// SendVerificationCode 发送验证码
func (s *authServiceImpl) SendVerificationCode(ctx context.Context, req *dto.SendVerificationCodeRequest) (*dto.SendVerificationCodeResponse, error) {
	// 转换验证码类型
	var codeType services.VerificationCodeType
	switch req.Type {
	case "register":
		codeType = services.VerificationCodeTypeRegister
	case "password_reset":
		codeType = services.VerificationCodeTypePasswordReset
	default:
		return nil, fmt.Errorf("invalid verification code type")
	}

	// 生成验证码
	code, err := s.verificationService.GenerateCode(ctx, req.Email, codeType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification code: %w", err)
	}

	// 发送邮件
	switch codeType {
	case services.VerificationCodeTypeRegister:
		err = s.emailService.SendVerificationCode(ctx, req.Email, code)
	case services.VerificationCodeTypePasswordReset:
		err = s.emailService.SendPasswordResetCode(ctx, req.Email, code)
	}

	if err != nil {
		// 如果邮件发送失败，使验证码失效
		s.verificationService.InvalidateCode(ctx, req.Email, codeType)
		return nil, fmt.Errorf("failed to send verification email: %w", err)
	}

	return &dto.SendVerificationCodeResponse{
		Message:   "Verification code sent successfully",
		ExpiresIn: 600, // 10分钟
	}, nil
}

// VerifyCode 验证验证码
func (s *authServiceImpl) VerifyCode(ctx context.Context, req *dto.VerifyCodeRequest) (*dto.VerifyCodeResponse, error) {
	// 转换验证码类型
	var codeType services.VerificationCodeType
	switch req.Type {
	case "register":
		codeType = services.VerificationCodeTypeRegister
	case "password_reset":
		codeType = services.VerificationCodeTypePasswordReset
	default:
		return nil, fmt.Errorf("invalid verification code type")
	}

	// 验证验证码
	err := s.verificationService.VerifyCode(ctx, req.Email, req.Code, codeType)
	if err != nil {
		return &dto.VerifyCodeResponse{
			Valid:   false,
			Message: err.Error(),
		}, nil
	}

	return &dto.VerifyCodeResponse{
		Valid:   true,
		Message: "Verification code is valid",
	}, nil
}

// RegisterWithCode 带验证码的注册
func (s *authServiceImpl) RegisterWithCode(ctx context.Context, req *dto.RegisterWithCodeRequest) (*dto.RegisterResponse, error) {
	// 验证验证码
	err := s.verificationService.VerifyCode(ctx, req.Email, req.VerificationCode, services.VerificationCodeTypeRegister)
	if err != nil {
		return nil, fmt.Errorf("invalid verification code: %w", err)
	}

	// 检查邮箱是否已存在
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("email already exists")
	}
	// 如果错误不是"用户不存在"，则返回错误
	if err != nil && err != entities.ErrUserNotFound {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}

	// 处理用户名：如果为空则自动生成
	username := req.Username
	if username == "" {
		username = s.generateUsername()
	}

	// 哈希密码
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 创建用户实体
	user := &entities.User{
		Username:     username,
		Email:        req.Email,
		PasswordHash: &hashedPassword,
		Status:       entities.UserStatusActive,
		Balance:      0.0,
	}

	// 保存用户
	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 构造响应
	response := &dto.RegisterResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  "",
		Message:   "User registered successfully",
		CreatedAt: user.CreatedAt,
	}

	return response, nil
}

// ResetPassword 重置密码
func (s *authServiceImpl) ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) (*dto.ResetPasswordResponse, error) {
	// 验证验证码
	err := s.verificationService.VerifyCode(ctx, req.Email, req.VerificationCode, services.VerificationCodeTypePasswordReset)
	if err != nil {
		return nil, fmt.Errorf("invalid verification code: %w", err)
	}

	// 获取用户信息
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 哈希新密码
	hashedPassword, err := s.hashPassword(req.NewPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 更新密码
	user.PasswordHash = &hashedPassword
	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	return &dto.ResetPasswordResponse{
		Message: "Password reset successfully",
	}, nil
}
