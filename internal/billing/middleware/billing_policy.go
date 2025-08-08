package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// BillingPolicy 计费策略
type BillingPolicy struct {
	// 完全跳过计费的路径
	SkipPaths map[string]bool
	
	// 管理员路径 - 不计费但需要记录
	AdminPaths map[string]bool
	
	// 免费路径 - 创建使用日志但不扣费
	FreePaths map[string]bool
	
	// 特殊计费规则
	SpecialRules map[string]SpecialBillingRule
}

// SpecialBillingRule 特殊计费规则
type SpecialBillingRule struct {
	Type        SpecialRuleType `json:"type"`
	Description string          `json:"description"`
	Config      map[string]interface{} `json:"config"`
}

// SpecialRuleType 特殊规则类型
type SpecialRuleType string

const (
	SpecialRuleSkip     SpecialRuleType = "skip"      // 跳过计费
	SpecialRuleAdmin    SpecialRuleType = "admin"     // 管理员接口
	SpecialRuleFree     SpecialRuleType = "free"      // 免费接口
	SpecialRuleInternal SpecialRuleType = "internal"  // 内部接口
	SpecialRuleMonitor  SpecialRuleType = "monitor"   // 监控接口
)

// NewBillingPolicy 创建默认的计费策略
func NewBillingPolicy() *BillingPolicy {
	return &BillingPolicy{
		// 完全跳过计费的路径
		SkipPaths: map[string]bool{
			"/health":           true,
			"/metrics":          true,
			"/swagger":          true,
			"/favicon.ico":      true,
			"/robots.txt":       true,
		},
		
		// 管理员路径 - 不计费但记录使用日志
		AdminPaths: map[string]bool{
			"/admin":                    true,
			"/api/v1/admin":            true,
			"/api/admin":               true,
			"/dashboard":               true,
			"/api/v1/users":            true,  // 用户管理
			"/api/v1/api-keys":         true,  // API密钥管理
			"/api/v1/models":           true,  // 模型管理
			"/api/v1/quotas":           true,  // 配额管理
			"/api/v1/billing/records":  true,  // 计费记录查询
			"/api/v1/stats":            true,  // 统计接口
			"/api/v1/system":           true,  // 系统管理
		},
		
		// 免费路径 - 创建使用日志但不扣费
		FreePaths: map[string]bool{
			"/api/v1/models/list":      true,  // 模型列表
			"/api/v1/user/profile":     true,  // 用户资料
			"/api/v1/user/balance":     true,  // 余额查询
			"/api/v1/usage/stats":      true,  // 使用统计
		},
		
		// 特殊规则
		SpecialRules: map[string]SpecialBillingRule{
			"/api/v1/auth": {
				Type:        SpecialRuleSkip,
				Description: "认证接口不计费",
			},
			"/api/v1/test": {
				Type:        SpecialRuleFree,
				Description: "测试接口免费",
				Config: map[string]interface{}{
					"max_requests_per_day": 100,
				},
			},
		},
	}
}

// ShouldSkipBilling 判断是否应该完全跳过计费
func (bp *BillingPolicy) ShouldSkipBilling(c *gin.Context) bool {
	path := c.Request.URL.Path
	method := c.Request.Method
	
	// 检查完全跳过的路径
	if bp.matchPath(path, bp.SkipPaths) {
		return true
	}
	
	// GET请求通常不计费（除非是API调用）
	if method == "GET" && !bp.isAPICall(path) {
		return true
	}
	
	// 检查特殊规则
	if rule, exists := bp.getSpecialRule(path); exists {
		return rule.Type == SpecialRuleSkip
	}
	
	return false
}

// ShouldCreateUsageLog 判断是否应该创建使用日志（不计费）
func (bp *BillingPolicy) ShouldCreateUsageLog(c *gin.Context) bool {
	path := c.Request.URL.Path
	
	// 管理员路径需要记录但不计费
	if bp.matchPath(path, bp.AdminPaths) {
		return true
	}
	
	// 免费路径需要记录但不计费
	if bp.matchPath(path, bp.FreePaths) {
		return true
	}
	
	// 检查特殊规则
	if rule, exists := bp.getSpecialRule(path); exists {
		return rule.Type == SpecialRuleAdmin || rule.Type == SpecialRuleFree
	}
	
	return false
}

// GetBillingBehavior 获取计费行为
func (bp *BillingPolicy) GetBillingBehavior(c *gin.Context) BillingBehavior {
	path := c.Request.URL.Path
	
	// 完全跳过
	if bp.ShouldSkipBilling(c) {
		return BillingBehaviorSkip
	}
	
	// 管理员接口
	if bp.matchPath(path, bp.AdminPaths) {
		return BillingBehaviorLogOnly
	}
	
	// 免费接口
	if bp.matchPath(path, bp.FreePaths) {
		return BillingBehaviorLogOnly
	}
	
	// 检查特殊规则
	if rule, exists := bp.getSpecialRule(path); exists {
		switch rule.Type {
		case SpecialRuleSkip:
			return BillingBehaviorSkip
		case SpecialRuleAdmin, SpecialRuleFree:
			return BillingBehaviorLogOnly
		}
	}
	
	// 默认正常计费
	return BillingBehaviorNormal
}

// IsAdminRequest 判断是否为管理员请求
func (bp *BillingPolicy) IsAdminRequest(c *gin.Context) bool {
	path := c.Request.URL.Path
	return bp.matchPath(path, bp.AdminPaths)
}

// GetAdminInfo 获取管理员信息
func (bp *BillingPolicy) GetAdminInfo(c *gin.Context) *AdminInfo {
	// 从JWT token或session中获取管理员信息
	if adminIDInterface, exists := c.Get("admin_id"); exists {
		if adminID, ok := adminIDInterface.(int64); ok {
			return &AdminInfo{
				AdminID:   adminID,
				AdminRole: bp.getAdminRole(c),
				IsAdmin:   true,
			}
		}
	}
	
	// 检查是否为系统内部调用
	if c.GetHeader("X-Internal-Call") == "true" {
		return &AdminInfo{
			AdminID:   0,
			AdminRole: "system",
			IsAdmin:   true,
		}
	}
	
	return nil
}

// BillingBehavior 计费行为枚举
type BillingBehavior int

const (
	BillingBehaviorSkip    BillingBehavior = iota // 完全跳过
	BillingBehaviorLogOnly                        // 只记录日志，不计费
	BillingBehaviorNormal                         // 正常计费
)

// AdminInfo 管理员信息
type AdminInfo struct {
	AdminID   int64  `json:"admin_id"`
	AdminRole string `json:"admin_role"`
	IsAdmin   bool   `json:"is_admin"`
}

// 私有辅助方法

func (bp *BillingPolicy) matchPath(path string, pathMap map[string]bool) bool {
	// 精确匹配
	if pathMap[path] {
		return true
	}
	
	// 前缀匹配
	for skipPath := range pathMap {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	
	return false
}

func (bp *BillingPolicy) isAPICall(path string) bool {
	// 判断是否为API调用
	return strings.HasPrefix(path, "/api/")
}

func (bp *BillingPolicy) getSpecialRule(path string) (SpecialBillingRule, bool) {
	// 精确匹配
	if rule, exists := bp.SpecialRules[path]; exists {
		return rule, true
	}
	
	// 前缀匹配
	for rulePath, rule := range bp.SpecialRules {
		if strings.HasPrefix(path, rulePath) {
			return rule, true
		}
	}
	
	return SpecialBillingRule{}, false
}

func (bp *BillingPolicy) getAdminRole(c *gin.Context) string {
	if roleInterface, exists := c.Get("admin_role"); exists {
		if role, ok := roleInterface.(string); ok {
			return role
		}
	}
	
	// 默认角色
	return "admin"
}

// AddSkipPath 动态添加跳过路径
func (bp *BillingPolicy) AddSkipPath(path string) {
	bp.SkipPaths[path] = true
}

// AddAdminPath 动态添加管理员路径
func (bp *BillingPolicy) AddAdminPath(path string) {
	bp.AdminPaths[path] = true
}

// AddFreePath 动态添加免费路径
func (bp *BillingPolicy) AddFreePath(path string) {
	bp.FreePaths[path] = true
}

// AddSpecialRule 动态添加特殊规则
func (bp *BillingPolicy) AddSpecialRule(path string, rule SpecialBillingRule) {
	bp.SpecialRules[path] = rule
}