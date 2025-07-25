package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/infrastructure/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// OAuthService OAuth认证服务接口
type OAuthService interface {
	// GetAuthURL 获取OAuth认证URL
	GetAuthURL(ctx context.Context, provider string) (string, string, error)

	// HandleCallback 处理OAuth回调
	HandleCallback(ctx context.Context, req *dto.OAuthLoginRequest) (*dto.LoginResponse, error)

	// GetProviderConfig 获取提供商配置
	GetProviderConfig(provider string) (*oauth2.Config, error)
}

// oauthServiceImpl OAuth服务实现
type oauthServiceImpl struct {
	userRepo   repositories.UserRepository
	jwtService JWTService
	config     *config.Config
}

// NewOAuthService 创建OAuth服务
func NewOAuthService(
	userRepo repositories.UserRepository,
	jwtService JWTService,
	config *config.Config,
) OAuthService {
	return &oauthServiceImpl{
		userRepo:   userRepo,
		jwtService: jwtService,
		config:     config,
	}
}

// GetAuthURL 获取OAuth认证URL
func (s *oauthServiceImpl) GetAuthURL(ctx context.Context, provider string) (string, string, error) {
	oauthConfig, err := s.GetProviderConfig(provider)
	if err != nil {
		return "", "", fmt.Errorf("failed to get provider config: %w", err)
	}

	// 生成随机state
	state, err := generateRandomState()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	// 生成认证URL
	authURL := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return authURL, state, nil
}

// HandleCallback 处理OAuth回调
func (s *oauthServiceImpl) HandleCallback(ctx context.Context, req *dto.OAuthLoginRequest) (*dto.LoginResponse, error) {
	// 获取OAuth配置
	oauthConfig, err := s.GetProviderConfig(req.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider config: %w", err)
	}

	// 交换授权码获取token
	token, err := oauthConfig.Exchange(ctx, req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// 获取用户信息
	userInfo, err := s.getUserInfo(ctx, req.Provider, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// 查找或创建用户
	user, err := s.findOrCreateUser(ctx, req.Provider, userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %w", err)
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
		ExpiresIn:    24 * 60 * 60, // 24小时
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

// GetProviderConfig 获取提供商配置
func (s *oauthServiceImpl) GetProviderConfig(provider string) (*oauth2.Config, error) {
	switch strings.ToLower(provider) {
	case "google":
		if !s.config.OAuth.Google.Enabled {
			return nil, fmt.Errorf("google oauth is not enabled")
		}
		return &oauth2.Config{
			ClientID:     s.config.OAuth.Google.ClientID,
			ClientSecret: s.config.OAuth.Google.ClientSecret,
			RedirectURL:  s.config.OAuth.Google.RedirectURL,
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint:     google.Endpoint,
		}, nil
	case "github":
		if !s.config.OAuth.GitHub.Enabled {
			return nil, fmt.Errorf("github oauth is not enabled")
		}
		return &oauth2.Config{
			ClientID:     s.config.OAuth.GitHub.ClientID,
			ClientSecret: s.config.OAuth.GitHub.ClientSecret,
			RedirectURL:  s.config.OAuth.GitHub.RedirectURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported oauth provider: %s", provider)
	}
}

// getUserInfo 获取用户信息
func (s *oauthServiceImpl) getUserInfo(ctx context.Context, provider, accessToken string) (*dto.OAuthUserInfo, error) {
	switch strings.ToLower(provider) {
	case "google":
		return s.getGoogleUserInfo(ctx, accessToken)
	case "github":
		return s.getGitHubUserInfo(ctx, accessToken)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// getGoogleUserInfo 获取Google用户信息
func (s *oauthServiceImpl) getGoogleUserInfo(ctx context.Context, accessToken string) (*dto.OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var googleUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &dto.OAuthUserInfo{
		ID:     googleUser.ID,
		Email:  googleUser.Email,
		Name:   googleUser.Name,
		Avatar: &googleUser.Picture,
	}, nil
}

// getGitHubUserInfo 获取GitHub用户信息
func (s *oauthServiceImpl) getGitHubUserInfo(ctx context.Context, accessToken string) (*dto.OAuthUserInfo, error) {
	// 获取用户基本信息
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var githubUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.Unmarshal(body, &githubUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 如果用户没有公开邮箱，需要单独获取
	email := githubUser.Email
	if email == "" {
		email, err = s.getGitHubUserEmail(ctx, accessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to get user email: %w", err)
		}
	}

	name := githubUser.Name
	if name == "" {
		name = githubUser.Login
	}

	return &dto.OAuthUserInfo{
		ID:       fmt.Sprintf("%d", githubUser.ID),
		Email:    email,
		Name:     name,
		Avatar:   &githubUser.AvatarURL,
		Username: &githubUser.Login,
	}, nil
}

// getGitHubUserEmail 获取GitHub用户邮箱
func (s *oauthServiceImpl) getGitHubUserEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get user emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get user emails, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}

	if err := json.Unmarshal(body, &emails); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 查找主邮箱
	for _, email := range emails {
		if email.Primary {
			return email.Email, nil
		}
	}

	// 如果没有主邮箱，返回第一个邮箱
	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", fmt.Errorf("no email found")
}

// findOrCreateUser 查找或创建用户
func (s *oauthServiceImpl) findOrCreateUser(ctx context.Context, provider string, userInfo *dto.OAuthUserInfo) (*entities.User, error) {
	var user *entities.User
	var err error

	// 根据提供商查找用户
	switch strings.ToLower(provider) {
	case "google":
		user, err = s.userRepo.GetByGoogleID(ctx, userInfo.ID)
	case "github":
		user, err = s.userRepo.GetByGitHubID(ctx, userInfo.ID)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	// 如果找到用户，直接返回
	if err == nil && user != nil {
		return user, nil
	}

	// 如果没有找到用户，尝试通过邮箱查找
	if userInfo.Email != "" {
		existingUser, err := s.userRepo.GetByEmail(ctx, userInfo.Email)
		if err == nil && existingUser != nil {
			// 用户存在，更新OAuth信息
			return s.linkOAuthToExistingUser(ctx, existingUser, provider, userInfo)
		}
	}

	// 创建新用户
	return s.createNewOAuthUser(ctx, provider, userInfo)
}

// linkOAuthToExistingUser 将OAuth信息关联到现有用户
func (s *oauthServiceImpl) linkOAuthToExistingUser(ctx context.Context, user *entities.User, provider string, userInfo *dto.OAuthUserInfo) (*entities.User, error) {
	switch strings.ToLower(provider) {
	case "google":
		user.GoogleID = &userInfo.ID
	case "github":
		user.GitHubID = &userInfo.ID
	}

	// 更新用户信息
	if userInfo.Avatar != nil && user.Avatar == nil {
		user.Avatar = userInfo.Avatar
	}

	if user.FullName == nil && userInfo.Name != "" {
		user.FullName = &userInfo.Name
	}

	// 保存更新
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// createNewOAuthUser 创建新的OAuth用户
func (s *oauthServiceImpl) createNewOAuthUser(ctx context.Context, provider string, userInfo *dto.OAuthUserInfo) (*entities.User, error) {
	// 生成用户名
	username := s.generateUsername(userInfo)

	// 确保用户名唯一
	username, err := s.ensureUniqueUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure unique username: %w", err)
	}

	// 创建用户实体
	user := &entities.User{
		Username:   username,
		Email:      userInfo.Email,
		FullName:   &userInfo.Name,
		Avatar:     userInfo.Avatar,
		Status:     entities.UserStatusActive,
		Balance:    0.000001,
		AuthMethod: string(entities.AuthMethodPassword), // 默认设置，下面会更新
	}

	// 设置OAuth信息
	switch strings.ToLower(provider) {
	case "google":
		user.GoogleID = &userInfo.ID
		user.AuthMethod = string(entities.AuthMethodGoogle)
	case "github":
		user.GitHubID = &userInfo.ID
		user.AuthMethod = string(entities.AuthMethodGitHub)
	}

	// 保存用户
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// generateUsername 生成用户名
func (s *oauthServiceImpl) generateUsername(userInfo *dto.OAuthUserInfo) string {
	if userInfo.Username != nil && *userInfo.Username != "" {
		return *userInfo.Username
	}

	// 从邮箱提取用户名
	if userInfo.Email != "" {
		parts := strings.Split(userInfo.Email, "@")
		if len(parts) > 0 {
			return parts[0]
		}
	}

	// 从姓名生成用户名
	if userInfo.Name != "" {
		return strings.ReplaceAll(strings.ToLower(userInfo.Name), " ", "_")
	}

	// 默认用户名
	return "user_" + userInfo.ID
}

// ensureUniqueUsername 确保用户名唯一
func (s *oauthServiceImpl) ensureUniqueUsername(ctx context.Context, baseUsername string) (string, error) {
	username := baseUsername
	counter := 1

	for {
		// 检查用户名是否存在
		_, err := s.userRepo.GetByUsername(ctx, username)
		if err != nil {
			// 用户名不存在，可以使用
			return username, nil
		}

		// 用户名存在，添加数字后缀
		username = fmt.Sprintf("%s_%d", baseUsername, counter)
		counter++

		// 防止无限循环
		if counter > 1000 {
			return "", fmt.Errorf("failed to generate unique username")
		}
	}
}

// generateRandomState 生成随机state
func generateRandomState() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
