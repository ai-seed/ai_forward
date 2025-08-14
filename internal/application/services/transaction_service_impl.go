package services

import (
	"context"
	"fmt"
	"time"

	"ai-api-gateway/internal/application/dto"
	"ai-api-gateway/internal/domain/entities"
	"ai-api-gateway/internal/domain/repositories"
	"ai-api-gateway/internal/domain/services"
	"ai-api-gateway/internal/infrastructure/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// transactionServiceImpl 交易服务实现
type transactionServiceImpl struct {
	transactionRepo repositories.TransactionRepository
	userRepo        repositories.UserRepository
	db              *gorm.DB
	logger          logger.Logger
}

// NewTransactionService 创建交易服务实例
func NewTransactionService(
	transactionRepo repositories.TransactionRepository,
	userRepo repositories.UserRepository,
	db *gorm.DB,
	logger logger.Logger,
) services.TransactionService {
	return &transactionServiceImpl{
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		db:              db,
		logger:          logger,
	}
}

// CreateTransaction 创建交易记录
func (s *transactionServiceImpl) CreateTransaction(
	ctx context.Context,
	userID int64,
	transactionType entities.TransactionType,
	amount float64,
	relatedType *string,
	relatedID *int64,
	description string,
) (*dto.TransactionResponse, error) {
	// 获取用户当前余额
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	balanceBefore := user.Balance
	balanceAfter := balanceBefore + amount

	// 生成交易流水号
	transactionNo := s.generateTransactionNo()

	// 创建交易记录
	transaction := &entities.Transaction{
		UserID:        userID,
		TransactionNo: transactionNo,
		Type:          transactionType,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		RelatedType:   relatedType,
		RelatedID:     relatedID,
		Description:   description,
	}

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return s.toTransactionResponse(transaction), nil
}

// QueryTransactions 查询交易记录
func (s *transactionServiceImpl) QueryTransactions(ctx context.Context, req *dto.QueryTransactionsRequest) (*dto.PaginatedTransactionResponse, error) {
	offset := (req.Page - 1) * req.PageSize
	
	var transactions []*entities.Transaction
	var total int64
	var err error

	if req.StartTime != nil && req.EndTime != nil {
		transactions, total, err = s.transactionRepo.GetByDateRange(ctx, req.UserID, *req.StartTime, *req.EndTime, req.PageSize, offset)
	} else if req.Type != nil {
		transactions, total, err = s.transactionRepo.GetByType(ctx, req.UserID, *req.Type, req.PageSize, offset)
	} else if req.UserID != nil {
		transactions, total, err = s.transactionRepo.GetByUserID(ctx, *req.UserID, req.PageSize, offset)
	} else {
		return nil, fmt.Errorf("invalid query parameters")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}

	// 转换为响应格式
	responses := make([]*dto.TransactionResponse, len(transactions))
	for i, transaction := range transactions {
		responses[i] = s.toTransactionResponse(transaction)
	}

	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &dto.PaginatedTransactionResponse{
		Data:       responses,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetTransaction 获取交易记录详情
func (s *transactionServiceImpl) GetTransaction(ctx context.Context, userID int64, transactionID int64) (*dto.TransactionResponse, error) {
	transaction, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}

	if transaction.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return s.toTransactionResponse(transaction), nil
}

// GetUserBalance 获取用户余额
func (s *transactionServiceImpl) GetUserBalance(ctx context.Context, userID int64) (*dto.BalanceResponse, error) {
	// 从用户表获取余额
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 从交易记录获取最新余额进行校验
	transactionBalance, err := s.transactionRepo.GetUserBalance(ctx, userID)
	if err != nil && err != gorm.ErrRecordNotFound {
		s.logger.WithFields(map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		}).Warn("Failed to get balance from transactions")
	}

	// 如果交易记录中的余额与用户表中的余额不一致，记录警告
	if transactionBalance != user.Balance && transactionBalance != 0 {
		s.logger.WithFields(map[string]interface{}{
			"user_id":            userID,
			"user_balance":       user.Balance,
			"transaction_balance": transactionBalance,
		}).Warn("Balance mismatch between user table and transaction records")
	}

	return &dto.BalanceResponse{
		UserID:        userID,
		Balance:       user.Balance,
		FrozenBalance: 0, // 暂时不支持冻结余额
		TotalBalance:  user.Balance,
	}, nil
}

// GetTransactionSummary 获取交易汇总
func (s *transactionServiceImpl) GetTransactionSummary(ctx context.Context, userID int64, startTime, endTime *time.Time) (*dto.TransactionSummaryResponse, error) {
	var start, end time.Time
	
	if startTime != nil {
		start = *startTime
	} else {
		start = time.Now().AddDate(0, -1, 0) // 默认最近一个月
	}
	
	if endTime != nil {
		end = *endTime
	} else {
		end = time.Now()
	}

	summary, err := s.transactionRepo.GetUserTransactionSummary(ctx, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction summary: %w", err)
	}

	return &dto.TransactionSummaryResponse{
		TotalIncome:  summary.TotalIncome,
		TotalExpense: summary.TotalExpense,
		NetAmount:    summary.NetAmount,
		Count:        summary.Count,
	}, nil
}

// UpdateUserBalance 更新用户余额（原子操作）
func (s *transactionServiceImpl) UpdateUserBalance(
	ctx context.Context,
	userID int64,
	amount float64,
	transactionType entities.TransactionType,
	relatedType *string,
	relatedID *int64,
	description string,
) error {
	// 使用数据库事务确保原子性
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取用户并锁定行
		var user entities.User
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", userID).First(&user).Error; err != nil {
			return fmt.Errorf("user not found: %w", err)
		}

		balanceBefore := user.Balance
		balanceAfter := balanceBefore + amount

		// 检查余额是否足够（对于支出交易）
		if amount < 0 && balanceAfter < 0 {
			// 允许余额变负数，但记录警告
			s.logger.WithFields(map[string]interface{}{
				"user_id":        userID,
				"balance_before": balanceBefore,
				"amount":         amount,
				"balance_after":  balanceAfter,
			}).Warn("User balance will become negative")
		}

		// 更新用户余额
		if err := tx.Model(&user).Update("balance", balanceAfter).Error; err != nil {
			return fmt.Errorf("failed to update user balance: %w", err)
		}

		// 创建交易记录
		transactionNo := s.generateTransactionNo()
		transaction := &entities.Transaction{
			UserID:        userID,
			TransactionNo: transactionNo,
			Type:          transactionType,
			Amount:        amount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			RelatedType:   relatedType,
			RelatedID:     relatedID,
			Description:   description,
		}

		if err := tx.Create(transaction).Error; err != nil {
			return fmt.Errorf("failed to create transaction record: %w", err)
		}

		s.logger.WithFields(map[string]interface{}{
			"user_id":        userID,
			"transaction_no": transactionNo,
			"type":           transactionType,
			"amount":         amount,
			"balance_before": balanceBefore,
			"balance_after":  balanceAfter,
		}).Info("User balance updated successfully")

		return nil
	})
}

// 辅助方法

// generateTransactionNo 生成交易流水号
func (s *transactionServiceImpl) generateTransactionNo() string {
	return fmt.Sprintf("T%d%s", time.Now().Unix(), uuid.New().String()[:8])
}

// toTransactionResponse 转换为交易响应
func (s *transactionServiceImpl) toTransactionResponse(transaction *entities.Transaction) *dto.TransactionResponse {
	return &dto.TransactionResponse{
		ID:            transaction.ID,
		TransactionNo: transaction.TransactionNo,
		Type:          transaction.Type,
		Amount:        transaction.Amount,
		BalanceBefore: transaction.BalanceBefore,
		BalanceAfter:  transaction.BalanceAfter,
		RelatedType:   transaction.RelatedType,
		RelatedID:     transaction.RelatedID,
		Description:   transaction.Description,
		CreatedAt:     transaction.CreatedAt,
	}
}
