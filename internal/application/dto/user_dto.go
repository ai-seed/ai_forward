package dto

import (
	"ai-api-gateway/internal/domain/entities"
	"time"
)

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string  `json:"username" validate:"required,min=3,max=50"`
	Email    string  `json:"email" validate:"required,email"`
	Password *string `json:"password,omitempty" validate:"omitempty,min=6,max=100"`
	FullName *string `json:"full_name,omitempty" validate:"omitempty,max=100"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Username *string              `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	Email    *string              `json:"email,omitempty" validate:"omitempty,email"`
	FullName *string              `json:"full_name,omitempty" validate:"omitempty,max=100"`
	Status   *entities.UserStatus `json:"status,omitempty"`
}

// UserResponse 用户响应
type UserResponse struct {
	ID         int64               `json:"id"`
	Username   string              `json:"username"`
	Email      string              `json:"email"`
	FullName   *string             `json:"full_name,omitempty"`
	Avatar     *string             `json:"avatar,omitempty"`
	Status     entities.UserStatus `json:"status"`
	Balance    float64             `json:"balance"`
	AuthMethod string              `json:"auth_method"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	Users      []*UserResponse `json:"users"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// BalanceUpdateRequest 余额更新请求
type BalanceUpdateRequest struct {
	Amount      float64 `json:"amount" validate:"required"`
	Operation   string  `json:"operation" validate:"required,oneof=add deduct"`
	Description string  `json:"description,omitempty"`
}

// OAuthUserInfo OAuth用户信息
type OAuthUserInfo struct {
	ID       string  `json:"id"`
	Email    string  `json:"email"`
	Name     string  `json:"name"`
	Avatar   *string `json:"avatar,omitempty"`
	Username *string `json:"username,omitempty"`
}

// OAuthLoginRequest OAuth登录请求
type OAuthLoginRequest struct {
	Provider string `json:"provider" validate:"required,oneof=google github"`
	Code     string `json:"code" validate:"required"`
	State    string `json:"state" validate:"required"`
}

// ToEntity 转换为实体
func (r *CreateUserRequest) ToEntity() *entities.User {
	return &entities.User{
		Username:   r.Username,
		Email:      r.Email,
		FullName:   r.FullName,
		Status:     entities.UserStatusActive,
		Balance:    0.000001,
		AuthMethod: string(entities.AuthMethodPassword),
	}
}

// FromEntity 从实体转换
func (r *UserResponse) FromEntity(user *entities.User) *UserResponse {
	return &UserResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		FullName:   user.FullName,
		Avatar:     user.Avatar,
		Status:     user.Status,
		Balance:    user.Balance,
		AuthMethod: user.AuthMethod,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}
}

// FromEntities 从实体列表转换
func FromUserEntities(users []*entities.User) []*UserResponse {
	responses := make([]*UserResponse, len(users))
	for i, user := range users {
		responses[i] = (&UserResponse{}).FromEntity(user)
	}
	return responses
}
