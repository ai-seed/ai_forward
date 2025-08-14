package repositories

import (
	"context"
	"time"

	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"

	"gorm.io/gorm"
)

// giftRecordRepository 赠送记录仓储实现
type giftRecordRepository struct {
	db *gorm.DB
}

// NewGiftRecordRepository 创建赠送记录仓储
func NewGiftRecordRepository(db *gorm.DB) repositories.GiftRecordRepository {
	return &giftRecordRepository{db: db}
}

// Create 创建赠送记录
func (r *giftRecordRepository) Create(ctx context.Context, record *entities.GiftRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// GetByID 根据ID获取赠送记录
func (r *giftRecordRepository) GetByID(ctx context.Context, id int64) (*entities.GiftRecord, error) {
	var record entities.GiftRecord
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByUserID 根据用户ID获取赠送记录列表
func (r *giftRecordRepository) GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*entities.GiftRecord, int64, error) {
	var records []*entities.GiftRecord
	var total int64

	// 获取总数
	if err := r.db.WithContext(ctx).Model(&entities.GiftRecord{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
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

// GetByType 根据赠送类型获取记录
func (r *giftRecordRepository) GetByType(ctx context.Context, userID *int64, giftType entities.GiftType, limit, offset int) ([]*entities.GiftRecord, int64, error) {
	var records []*entities.GiftRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&entities.GiftRecord{}).Where("gift_type = ?", giftType)
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

// Update 更新赠送记录
func (r *giftRecordRepository) Update(ctx context.Context, record *entities.GiftRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

// UpdateStatus 更新赠送状态
func (r *giftRecordRepository) UpdateStatus(ctx context.Context, id int64, status entities.GiftStatus) error {
	return r.db.WithContext(ctx).
		Model(&entities.GiftRecord{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// GetPendingRecords 获取待处理的赠送记录
func (r *giftRecordRepository) GetPendingRecords(ctx context.Context) ([]*entities.GiftRecord, error) {
	var records []*entities.GiftRecord
	err := r.db.WithContext(ctx).Where("status = ?", entities.GiftStatusPending).Find(&records).Error
	return records, err
}

// GetByRelatedID 根据关联ID获取赠送记录
func (r *giftRecordRepository) GetByRelatedID(ctx context.Context, relatedID int64, giftType entities.GiftType) ([]*entities.GiftRecord, error) {
	var records []*entities.GiftRecord
	err := r.db.WithContext(ctx).
		Where("related_id = ? AND gift_type = ?", relatedID, giftType).
		Find(&records).Error
	return records, err
}

// GetByDateRange 根据日期范围获取赠送记录
func (r *giftRecordRepository) GetByDateRange(ctx context.Context, userID *int64, startTime, endTime time.Time, limit, offset int) ([]*entities.GiftRecord, int64, error) {
	var records []*entities.GiftRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&entities.GiftRecord{}).
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
