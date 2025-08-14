package repositories

import (
	"context"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"

	"gorm.io/gorm"
)

// transactionRepository 交易流水仓储实现
type transactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository 创建交易流水仓储
func NewTransactionRepository(db *gorm.DB) repositories.TransactionRepository {
	return &transactionRepository{db: db}
}

// Create 创建交易记录
func (r *transactionRepository) Create(ctx context.Context, transaction *entities.Transaction) error {
	return r.db.WithContext(ctx).Create(transaction).Error
}

// GetByID 根据ID获取交易记录
func (r *transactionRepository) GetByID(ctx context.Context, id int64) (*entities.Transaction, error) {
	var transaction entities.Transaction
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

// GetByTransactionNo 根据交易号获取交易记录
func (r *transactionRepository) GetByTransactionNo(ctx context.Context, transactionNo string) (*entities.Transaction, error) {
	var transaction entities.Transaction
	err := r.db.WithContext(ctx).Where("transaction_no = ?", transactionNo).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

// GetByUserID 根据用户ID获取交易记录列表
func (r *transactionRepository) GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*entities.Transaction, int64, error) {
	var transactions []*entities.Transaction
	var total int64

	// 获取总数
	if err := r.db.WithContext(ctx).Model(&entities.Transaction{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取记录
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error

	return transactions, total, err
}

// GetByType 根据交易类型获取记录
func (r *transactionRepository) GetByType(ctx context.Context, userID *int64, transactionType entities.TransactionType, limit, offset int) ([]*entities.Transaction, int64, error) {
	var transactions []*entities.Transaction
	var total int64

	query := r.db.WithContext(ctx).Model(&entities.Transaction{}).Where("type = ?", transactionType)
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取记录
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error

	return transactions, total, err
}

// GetByDateRange 根据日期范围获取交易记录
func (r *transactionRepository) GetByDateRange(ctx context.Context, userID *int64, startTime, endTime time.Time, limit, offset int) ([]*entities.Transaction, int64, error) {
	var transactions []*entities.Transaction
	var total int64

	query := r.db.WithContext(ctx).Model(&entities.Transaction{}).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取记录
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error

	return transactions, total, err
}

// GetUserBalance 获取用户最新余额（从最后一笔交易记录）
func (r *transactionRepository) GetUserBalance(ctx context.Context, userID int64) (float64, error) {
	var transaction entities.Transaction
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&transaction).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil // 没有交易记录，余额为0
		}
		return 0, err
	}
	
	return transaction.BalanceAfter, nil
}

// GetUserTransactionSummary 获取用户交易汇总
func (r *transactionRepository) GetUserTransactionSummary(ctx context.Context, userID int64, startTime, endTime time.Time) (*repositories.TransactionSummary, error) {
	var summary repositories.TransactionSummary
	
	// 查询汇总数据
	err := r.db.WithContext(ctx).
		Model(&entities.Transaction{}).
		Select(`
			COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0) as total_income,
			COALESCE(SUM(CASE WHEN amount < 0 THEN ABS(amount) ELSE 0 END), 0) as total_expense,
			COALESCE(SUM(amount), 0) as net_amount,
			COUNT(*) as count
		`).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, startTime, endTime).
		Scan(&summary).Error
	
	return &summary, err
}
