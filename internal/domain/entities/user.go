package entities

import (
	"time"
)

// UserStatus 用户状态枚举
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDeleted   UserStatus = "deleted"
)

// AuthMethod 认证方式枚举
type AuthMethod string

const (
	AuthMethodPassword AuthMethod = "password"
	AuthMethodGoogle   AuthMethod = "google"
	AuthMethodGitHub   AuthMethod = "github"
)

// User 用户实体
type User struct {
	ID           int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	Username     string     `json:"username" gorm:"uniqueIndex;not null;size:100"`
	Email        string     `json:"email" gorm:"uniqueIndex;not null;size:255"`
	PasswordHash *string    `json:"password_hash" gorm:"size:255;column:password_hash"` // 密码哈希，不在JSON中返回
	FullName     *string    `json:"full_name,omitempty" gorm:"size:255"`
	Avatar       *string    `json:"avatar,omitempty" gorm:"size:500"` // 用户头像URL
	Status       UserStatus `json:"status" gorm:"not null;default:active;size:20"`
	Balance      float64    `json:"balance" gorm:"type:numeric(15,6);not null;default:0.000001"`

	// OAuth相关字段
	GoogleID   *string `json:"google_id,omitempty" gorm:"uniqueIndex;size:100"`      // Google OAuth ID
	GitHubID   *string `json:"github_id,omitempty" gorm:"uniqueIndex;size:100"`      // GitHub OAuth ID
	AuthMethod string  `json:"auth_method" gorm:"not null;default:password;size:20"` // 认证方式: password, google, github

	CreatedAt time.Time `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// IsActive 检查用户是否处于活跃状态
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// CanMakeRequest 检查用户是否可以发起请求
func (u *User) CanMakeRequest() bool {
	return u.IsActive()
}

func (u *User) HasBalance() bool {
	return u.Balance > 0
}

// DeductBalance 扣减用户余额（允许余额变负数）
func (u *User) DeductBalance(amount float64) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	// 移除余额检查，允许余额变负数
	u.Balance -= amount
	u.UpdatedAt = time.Now()
	return nil
}

// AddBalance 增加用户余额
func (u *User) AddBalance(amount float64) error {
	if amount < 0 {
		return ErrInvalidAmount
	}
	u.Balance += amount
	u.UpdatedAt = time.Now()
	return nil
}

// IsOAuthUser 检查用户是否是OAuth用户
func (u *User) IsOAuthUser() bool {
	return u.AuthMethod == string(AuthMethodGoogle) || u.AuthMethod == string(AuthMethodGitHub)
}

// IsPasswordUser 检查用户是否是密码用户
func (u *User) IsPasswordUser() bool {
	return u.AuthMethod == string(AuthMethodPassword)
}

// HasGoogleAuth 检查用户是否绑定了Google账号
func (u *User) HasGoogleAuth() bool {
	return u.GoogleID != nil && *u.GoogleID != ""
}

// HasGitHubAuth 检查用户是否绑定了GitHub账号
func (u *User) HasGitHubAuth() bool {
	return u.GitHubID != nil && *u.GitHubID != ""
}
