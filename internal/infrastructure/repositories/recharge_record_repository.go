package repositories

import (
	"context"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"

	"gorm.io/gorm"
)

// rechargeRecordRepository 充值记录仓储实现
type rechargeRecordRepository struct {
	db *gorm.DB
}

// NewRechargeRecordRepository 创建充值记录仓储
func NewRechargeRecordRepository(db *gorm.DB) repositories.RechargeRecordRepository {
	return &rechargeRecordRepository{db: db}
}

// Create 创建充值记录
func (r *rechargeRecordRepository) Create(ctx context.Context, record *entities.RechargeRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// GetByID 根据ID获取充值记录
func (r *rechargeRecordRepository) GetByID(ctx context.Context, id int64) (*entities.RechargeRecord, error) {
	var record entities.RechargeRecord
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByOrderNo 根据订单号获取充值记录
func (r *rechargeRecordRepository) GetByOrderNo(ctx context.Context, orderNo string) (*entities.RechargeRecord, error) {
	var record entities.RechargeRecord
	err := r.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByUserID 根据用户ID获取充值记录列表
func (r *rechargeRecordRepository) GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*entities.RechargeRecord, int64, error) {
	var records []*entities.RechargeRecord
	var total int64

	// 获取总数
	if err := r.db.WithContext(ctx).Model(&entities.RechargeRecord{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取记录
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error

	return records, total, err
}

// Update 更新充值记录
func (r *rechargeRecordRepository) Update(ctx context.Context, record *entities.RechargeRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

// UpdateStatus 更新充值状态
func (r *rechargeRecordRepository) UpdateStatus(ctx context.Context, id int64, status entities.RechargeStatus) error {
	return r.db.WithContext(ctx).
		Model(&entities.RechargeRecord{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// GetPendingRecords 获取待处理的充值记录
func (r *rechargeRecordRepository) GetPendingRecords(ctx context.Context, expiredBefore *time.Time) ([]*entities.RechargeRecord, error) {
	var records []*entities.RechargeRecord
	query := r.db.WithContext(ctx).Where("status = ?", entities.RechargeStatusPending)
	
	if expiredBefore != nil {
		query = query.Where("expired_at < ?", *expiredBefore)
	}
	
	err := query.Find(&records).Error
	return records, err
}

// GetByDateRange 根据日期范围获取充值记录
func (r *rechargeRecordRepository) GetByDateRange(ctx context.Context, userID *int64, startTime, endTime time.Time, limit, offset int) ([]*entities.RechargeRecord, int64, error) {
	var records []*entities.RechargeRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&entities.RechargeRecord{}).
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
		Find(&records).Error

	return records, total, err
}
