package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-api-gateway/internal/infrastructure/logger"

	"gorm.io/gorm"
)

// IndexInfo 索引信息
type IndexInfo struct {
	Name        string
	Table       string
	Columns     []string
	Unique      bool
	Condition   string // WHERE 条件，用于部分索引
	Concurrent  bool
	Description string
}

// PerformanceIndexes 性能优化索引列表
var PerformanceIndexes = []IndexInfo{
	// 1. 使用日志表索引
	{
		Name:        "idx_usage_logs_user_id_created",
		Table:       "usage_logs",
		Columns:     []string{"user_id", "created_at DESC"},
		Description: "按用户ID和创建时间查询使用日志",
		Concurrent:  true,
	},
	{
		Name:        "idx_usage_logs_api_key_id_created",
		Table:       "usage_logs",
		Columns:     []string{"api_key_id", "created_at DESC"},
		Description: "按API密钥ID和创建时间查询",
		Concurrent:  true,
	},
	{
		Name:        "idx_usage_logs_model_created",
		Table:       "usage_logs",
		Columns:     []string{"model", "created_at DESC"},
		Description: "按模型和创建时间查询",
		Concurrent:  true,
	},
	{
		Name:        "idx_usage_logs_success_created",
		Table:       "usage_logs",
		Columns:     []string{"success", "created_at DESC"},
		Description: "按成功状态和创建时间查询",
		Concurrent:  true,
	},

	// 2. API密钥表索引
	{
		Name:        "idx_api_keys_user_id_status",
		Table:       "api_keys",
		Columns:     []string{"user_id", "status"},
		Description: "按用户ID和状态查询API密钥",
		Concurrent:  true,
	},
	{
		Name:        "idx_api_keys_status_created",
		Table:       "api_keys",
		Columns:     []string{"status", "created_at DESC"},
		Description: "按状态和创建时间查询",
		Concurrent:  true,
	},

	// 3. 提供商表索引
	{
		Name:        "idx_providers_status_priority",
		Table:       "providers",
		Columns:     []string{"status", "priority DESC"},
		Description: "按状态和优先级查询提供商",
		Concurrent:  true,
	},
	{
		Name:        "idx_providers_type_status",
		Table:       "providers",
		Columns:     []string{"provider_type", "status"},
		Description: "按类型和状态查询",
		Concurrent:  true,
	},

	// 4. 模型表索引
	{
		Name:        "idx_models_provider_id_status",
		Table:       "models",
		Columns:     []string{"provider_id", "status"},
		Description: "按提供商ID和状态查询模型",
		Concurrent:  true,
	},
	{
		Name:        "idx_models_model_type_status",
		Table:       "models",
		Columns:     []string{"model_type", "status"},
		Description: "按模型类型和状态查询",
		Concurrent:  true,
	},
	{
		Name:        "idx_models_slug",
		Table:       "models",
		Columns:     []string{"slug"},
		Description: "按slug查询模型",
		Concurrent:  true,
	},

	// 5. 配额表索引
	{
		Name:        "idx_quotas_user_id_status",
		Table:       "quotas",
		Columns:     []string{"user_id", "status"},
		Description: "按用户ID和状态查询配额",
		Concurrent:  true,
	},
	{
		Name:        "idx_quotas_api_key_id_status",
		Table:       "quotas",
		Columns:     []string{"api_key_id", "status"},
		Description: "按API密钥ID和状态查询配额",
		Concurrent:  true,
	},
	{
		Name:        "idx_quotas_expires_at",
		Table:       "quotas",
		Columns:     []string{"expires_at"},
		Description: "按过期时间查询配额",
		Concurrent:  true,
	},

	// 6. 用户表索引
	{
		Name:        "idx_users_username",
		Table:       "users",
		Columns:     []string{"username"},
		Description: "按用户名查询",
		Concurrent:  true,
	},
	{
		Name:        "idx_users_email",
		Table:       "users",
		Columns:     []string{"email"},
		Description: "按邮箱查询",
		Concurrent:  true,
	},
	{
		Name:        "idx_users_status_created",
		Table:       "users",
		Columns:     []string{"status", "created_at DESC"},
		Description: "按状态和创建时间查询",
		Concurrent:  true,
	},

	// 7. 提供商模型支持表索引
	{
		Name:        "idx_provider_model_support_model_slug",
		Table:       "provider_model_support",
		Columns:     []string{"model_slug"},
		Description: "按模型slug查询支持的提供商",
		Concurrent:  true,
	},
	{
		Name:        "idx_provider_model_support_provider_id",
		Table:       "provider_model_support",
		Columns:     []string{"provider_id"},
		Description: "按提供商ID查询支持的模型",
		Concurrent:  true,
	},

	// 8. 部分索引优化
	{
		Name:        "idx_api_keys_active_user_id",
		Table:       "api_keys",
		Columns:     []string{"user_id"},
		Condition:   "status = 'active'",
		Description: "只索引活跃的API密钥",
		Concurrent:  true,
	},
	{
		Name:        "idx_providers_active_priority",
		Table:       "providers",
		Columns:     []string{"priority DESC"},
		Condition:   "status = 'active'",
		Description: "只索引活跃的提供商",
		Concurrent:  true,
	},
	{
		Name:        "idx_models_active_slug",
		Table:       "models",
		Columns:     []string{"slug"},
		Condition:   "status = 'active'",
		Description: "只索引活跃的模型",
		Concurrent:  true,
	},
}

// CreatePerformanceIndexes 创建性能优化索引
func CreatePerformanceIndexes(db *gorm.DB, log logger.Logger) error {
	log.Info("Starting to create performance indexes...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	createdCount := 0
	skippedCount := 0

	for _, indexInfo := range PerformanceIndexes {
		// 检查索引是否已存在
		exists, err := indexExists(db, indexInfo.Name)
		if err != nil {
			log.WithFields(map[string]interface{}{
				"index_name": indexInfo.Name,
				"error":      err.Error(),
			}).Warn("Failed to check if index exists, skipping")
			skippedCount++
			continue
		}

		if exists {
			log.WithFields(map[string]interface{}{
				"index_name": indexInfo.Name,
				"table":      indexInfo.Table,
			}).Debug("Index already exists, skipping")
			skippedCount++
			continue
		}

		// 创建索引
		err = createIndex(ctx, db, indexInfo, log)
		if err != nil {
			log.WithFields(map[string]interface{}{
				"index_name":  indexInfo.Name,
				"table":       indexInfo.Table,
				"description": indexInfo.Description,
				"error":       err.Error(),
			}).Error("Failed to create index")
			// 继续处理其他索引，不要因为一个失败就停止
			continue
		}

		createdCount++
		log.WithFields(map[string]interface{}{
			"index_name":  indexInfo.Name,
			"table":       indexInfo.Table,
			"description": indexInfo.Description,
		}).Info("Successfully created performance index")
	}

	log.WithFields(map[string]interface{}{
		"created_count": createdCount,
		"skipped_count": skippedCount,
		"total_count":   len(PerformanceIndexes),
	}).Info("Performance indexes creation completed")

	return nil
}

// indexExists 检查索引是否存在
func indexExists(db *gorm.DB, indexName string) (bool, error) {
	var count int64
	
	// PostgreSQL
	if isPostgreSQL(db) {
		err := db.Raw(`
			SELECT COUNT(*) 
			FROM pg_indexes 
			WHERE indexname = ? AND schemaname = 'public'
		`, indexName).Scan(&count).Error
		return count > 0, err
	}

	// SQLite
	if isSQLite(db) {
		err := db.Raw(`
			SELECT COUNT(*) 
			FROM sqlite_master 
			WHERE type = 'index' AND name = ?
		`, indexName).Scan(&count).Error
		return count > 0, err
	}

	// MySQL
	if isMySQL(db) {
		var dbName string
		db.Raw("SELECT DATABASE()").Scan(&dbName)
		err := db.Raw(`
			SELECT COUNT(*) 
			FROM information_schema.statistics 
			WHERE index_name = ? AND table_schema = ?
		`, indexName, dbName).Scan(&count).Error
		return count > 0, err
	}

	return false, fmt.Errorf("unsupported database type")
}

// createIndex 创建索引
func createIndex(ctx context.Context, db *gorm.DB, indexInfo IndexInfo, log logger.Logger) error {
	sql := buildCreateIndexSQL(indexInfo)
	
	log.WithFields(map[string]interface{}{
		"index_name": indexInfo.Name,
		"sql":        sql,
	}).Debug("Executing index creation SQL")

	// 使用上下文执行，避免长时间阻塞
	return db.WithContext(ctx).Exec(sql).Error
}

// buildCreateIndexSQL 构建创建索引的SQL
func buildCreateIndexSQL(indexInfo IndexInfo) string {
	var sql strings.Builder
	
	sql.WriteString("CREATE")
	
	if indexInfo.Concurrent && isPostgreSQL(nil) {
		sql.WriteString(" INDEX CONCURRENTLY")
	} else {
		sql.WriteString(" INDEX")
	}
	
	sql.WriteString(" IF NOT EXISTS ")
	sql.WriteString(indexInfo.Name)
	sql.WriteString(" ON ")
	sql.WriteString(indexInfo.Table)
	sql.WriteString(" (")
	sql.WriteString(strings.Join(indexInfo.Columns, ", "))
	sql.WriteString(")")
	
	if indexInfo.Condition != "" {
		sql.WriteString(" WHERE ")
		sql.WriteString(indexInfo.Condition)
	}

	return sql.String()
}

// isPostgreSQL 检查是否为PostgreSQL
func isPostgreSQL(db *gorm.DB) bool {
	if db != nil {
		return db.Dialector.Name() == "postgres"
	}
	// 简单检查，实际应用中可以更精确
	return true // 假设默认使用PostgreSQL
}

// isSQLite 检查是否为SQLite
func isSQLite(db *gorm.DB) bool {
	return db.Dialector.Name() == "sqlite"
}

// isMySQL 检查是否为MySQL
func isMySQL(db *gorm.DB) bool {
	return db.Dialector.Name() == "mysql"
}

// AnalyzeIndexPerformance 分析索引性能
func AnalyzeIndexPerformance(db *gorm.DB, log logger.Logger) error {
	if !isPostgreSQL(db) {
		log.Warn("Index performance analysis is currently only supported for PostgreSQL")
		return nil
	}

	log.Info("Analyzing index performance...")

	// 获取索引使用统计
	var indexStats []struct {
		SchemaName   string `json:"schemaname"`
		TableName    string `json:"tablename"`
		IndexName    string `json:"indexname"`
		IdxTupRead   int64  `json:"idx_tup_read"`
		IdxTupFetch  int64  `json:"idx_tup_fetch"`
	}

	err := db.Raw(`
		SELECT 
			schemaname,
			tablename,
			indexname,
			idx_tup_read,
			idx_tup_fetch
		FROM pg_stat_user_indexes 
		WHERE schemaname = 'public'
		ORDER BY idx_tup_read DESC
		LIMIT 20
	`).Scan(&indexStats).Error

	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to get index statistics")
		return err
	}

	// 输出索引使用统计
	for _, stat := range indexStats {
		log.WithFields(map[string]interface{}{
			"table":       stat.TableName,
			"index":       stat.IndexName,
			"tup_read":    stat.IdxTupRead,
			"tup_fetch":   stat.IdxTupFetch,
		}).Info("Index usage statistics")
	}

	// 获取表扫描统计
	var tableStats []struct {
		SchemaName   string `json:"schemaname"`
		TableName    string `json:"tablename"`
		SeqScan      int64  `json:"seq_scan"`
		SeqTupRead   int64  `json:"seq_tup_read"`
		IdxScan      int64  `json:"idx_scan"`
		IdxTupFetch  int64  `json:"idx_tup_fetch"`
	}

	err = db.Raw(`
		SELECT 
			schemaname,
			tablename,
			seq_scan,
			seq_tup_read,
			idx_scan,
			idx_tup_fetch
		FROM pg_stat_user_tables 
		WHERE schemaname = 'public'
		ORDER BY seq_tup_read DESC
		LIMIT 10
	`).Scan(&tableStats).Error

	if err != nil {
		log.WithField("error", err.Error()).Error("Failed to get table statistics")
		return err
	}

	// 输出表扫描统计
	for _, stat := range tableStats {
		scanRatio := float64(stat.IdxScan) / float64(stat.SeqScan + stat.IdxScan)
		log.WithFields(map[string]interface{}{
			"table":         stat.TableName,
			"seq_scan":      stat.SeqScan,
			"seq_tup_read":  stat.SeqTupRead,
			"idx_scan":      stat.IdxScan,
			"idx_tup_fetch": stat.IdxTupFetch,
			"index_ratio":   fmt.Sprintf("%.2f%%", scanRatio*100),
		}).Info("Table scan statistics")
	}

	return nil
}