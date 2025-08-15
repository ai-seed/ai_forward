package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	appLogger "ai-api-gateway/internal/infrastructure/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormConfig GORM数据库配置
type GormConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	TimeZone string
}

// NewGormDB 创建GORM数据库连接
func NewGormDB(config GormConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		config.Host, config.User, config.Password, config.DBName, config.Port, config.SSLMode, config.TimeZone)

	// 配置GORM日志
	gormConfig := &gorm.Config{
		Logger: logger.New(
			log.New(log.Writer(), "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second, // 慢SQL阈值
				LogLevel:                  logger.Info, // 日志级别
				IgnoreRecordNotFoundError: true,        // 忽略ErrRecordNotFound错误
				Colorful:                  false,       // 禁用彩色打印
			},
		),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 获取底层sql.DB对象进行连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生存时间

	return db, nil
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB) error {
	// 定义所有需要迁移的模型
	models := []interface{}{
		// &entities.User{}, // 启用用户表迁移以支持OAuth字段
		// &entities.APIKey{},
		// &entities.Provider{},
		// &entities.Model{},
		// &entities.ModelPricing{},
		// &entities.ProviderModelSupport{},
		// &entities.Quota{},
		// &entities.QuotaUsage{},
		// &entities.UsageLog{},
		// &entities.BillingRecord{},
		// &entities.Tool{},
		// &entities.UserToolInstance{},
		// &entities.ToolModelSupport{}, // 添加工具-模型支持关联表
		// &entities.ToolUsageLog{},
		// &entities.MidjourneyJob{},

		// 支付系统相关实体
		// &entities.PaymentProvider{}, // 支付服务商
		// &entities.PaymentMethod{},   // 支付方式
		// &entities.PaymentChannel{},  // 支付渠道
		// &entities.RechargeRecord{},  // 充值记录
		// &entities.GiftRecord{},      // 赠送记录
		// &entities.Transaction{},     // 交易流水
		// &entities.RechargeOption{},  // 充值金额选项
		// &entities.GiftRule{},        // 赠送规则
	}

	// 执行自动迁移
	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	return nil
}

// CreateIndexes 创建额外的索引
func CreateIndexes(db *gorm.DB) error {
	// 用户表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)").Error; err != nil {
		return fmt.Errorf("failed to create users username index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)").Error; err != nil {
		return fmt.Errorf("failed to create users email index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)").Error; err != nil {
		return fmt.Errorf("failed to create users status index: %w", err)
	}

	// API密钥表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id)").Error; err != nil {
		return fmt.Errorf("failed to create api_keys user_id index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix)").Error; err != nil {
		return fmt.Errorf("failed to create api_keys key_prefix index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status)").Error; err != nil {
		return fmt.Errorf("failed to create api_keys status index: %w", err)
	}

	// 支付系统相关索引
	// 充值记录表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_recharge_records_user_id ON recharge_records(user_id)").Error; err != nil {
		return fmt.Errorf("failed to create recharge_records user_id index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_recharge_records_status ON recharge_records(status)").Error; err != nil {
		return fmt.Errorf("failed to create recharge_records status index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_recharge_records_created_at ON recharge_records(created_at)").Error; err != nil {
		return fmt.Errorf("failed to create recharge_records created_at index: %w", err)
	}

	// 赠送记录表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_gift_records_user_id ON gift_records(user_id)").Error; err != nil {
		return fmt.Errorf("failed to create gift_records user_id index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_gift_records_type ON gift_records(gift_type)").Error; err != nil {
		return fmt.Errorf("failed to create gift_records gift_type index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_gift_records_status ON gift_records(status)").Error; err != nil {
		return fmt.Errorf("failed to create gift_records status index: %w", err)
	}

	// 交易流水表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id)").Error; err != nil {
		return fmt.Errorf("failed to create transactions user_id index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type)").Error; err != nil {
		return fmt.Errorf("failed to create transactions type index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at)").Error; err != nil {
		return fmt.Errorf("failed to create transactions created_at index: %w", err)
	}

	// 支付服务商表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payment_providers_code ON payment_providers(code)").Error; err != nil {
		return fmt.Errorf("failed to create payment_providers code index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payment_providers_status ON payment_providers(status)").Error; err != nil {
		return fmt.Errorf("failed to create payment_providers status index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payment_providers_type ON payment_providers(type)").Error; err != nil {
		return fmt.Errorf("failed to create payment_providers type index: %w", err)
	}

	// 支付方式表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payment_methods_code ON payment_methods(code)").Error; err != nil {
		return fmt.Errorf("failed to create payment_methods code index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payment_methods_status ON payment_methods(status)").Error; err != nil {
		return fmt.Errorf("failed to create payment_methods status index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payment_methods_provider_id ON payment_methods(provider_id)").Error; err != nil {
		return fmt.Errorf("failed to create payment_methods provider_id index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payment_methods_sort_order ON payment_methods(sort_order)").Error; err != nil {
		return fmt.Errorf("failed to create payment_methods sort_order index: %w", err)
	}

	// 支付渠道表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payment_channels_method_id ON payment_channels(method_id)").Error; err != nil {
		return fmt.Errorf("failed to create payment_channels method_id index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payment_channels_provider_id ON payment_channels(provider_id)").Error; err != nil {
		return fmt.Errorf("failed to create payment_channels provider_id index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payment_channels_status ON payment_channels(status)").Error; err != nil {
		return fmt.Errorf("failed to create payment_channels status index: %w", err)
	}

	// 赠送规则表索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_gift_rules_type ON gift_rules(type)").Error; err != nil {
		return fmt.Errorf("failed to create gift_rules type index: %w", err)
	}

	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_gift_rules_status ON gift_rules(status)").Error; err != nil {
		return fmt.Errorf("failed to create gift_rules status index: %w", err)
	}

	return nil
}

// InitializeDatabase 初始化数据库（迁移+索引）
func InitializeDatabase(db *gorm.DB, log appLogger.Logger) error {
	// 执行自动迁移
	if err := AutoMigrate(db); err != nil {
		return fmt.Errorf("auto migration failed: %w", err)
	}

	// 创建基础索引
	// if err := CreateIndexes(db); err != nil {
	// 	return fmt.Errorf("create indexes failed: %w", err)
	// }

	// 创建性能优化索引
	if err := CreatePerformanceIndexes(db, log); err != nil {
		// 性能索引创建失败不应该阻止应用启动，只记录警告
		log.WithField("error", err.Error()).Warn("Failed to create performance indexes, continuing with startup")
	}

	return nil
}

// HealthCheck 数据库健康检查
func HealthCheck(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// GetDBStats 获取数据库连接池统计信息
func GetDBStats(db *gorm.DB) (map[string]interface{}, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}, nil
}

// ConnectionKeepAliveService 数据库连接保活服务
type ConnectionKeepAliveService struct {
	db       *gorm.DB
	logger   appLogger.Logger
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewConnectionKeepAliveService 创建数据库连接保活服务
func NewConnectionKeepAliveService(db *gorm.DB, logger appLogger.Logger, interval time.Duration) *ConnectionKeepAliveService {
	if interval <= 0 {
		interval = 30 * time.Second // 默认30秒
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &ConnectionKeepAliveService{
		db:       db,
		logger:   logger,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动连接保活服务
func (s *ConnectionKeepAliveService) Start() {
	s.wg.Add(1)
	go s.keepAliveLoop()
	s.logger.WithField("interval", s.interval).Info("Database connection keep-alive service started")
}

// Stop 停止连接保活服务
func (s *ConnectionKeepAliveService) Stop() {
	s.cancel()
	s.wg.Wait()
	s.logger.Info("Database connection keep-alive service stopped")
}

// keepAliveLoop 保活循环
func (s *ConnectionKeepAliveService) keepAliveLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// 执行简单的SELECT 1查询来保持连接活跃
			var result int
			if err := s.db.Raw("SELECT 1").Scan(&result).Error; err != nil {
				s.logger.WithField("error", err.Error()).Error("Database keep-alive query failed")
			}
		}
	}
}
