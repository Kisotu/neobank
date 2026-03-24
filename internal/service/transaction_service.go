package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Kisotu/neobank/internal/domain"
	"github.com/Kisotu/neobank/internal/repository"
)

type transactionService struct {
	transactionRepo repository.TransactionRepository
	accountRepo     repository.AccountRepository
	logger          *slog.Logger
}

func NewTransactionService(transactionRepo repository.TransactionRepository, accountRepo repository.AccountRepository, logger *slog.Logger) TransactionService {
	if logger == nil {
		logger = slog.Default()
	}

	return &transactionService{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		logger:          logger,
	}
}

func (s *transactionService) GetByID(ctx context.Context, userID, transactionID uuid.UUID) (*TransactionResponse, error) {
	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	account, err := s.accountRepo.GetByID(ctx, tx.AccountID)
	if err != nil {
		return nil, err
	}
	if !account.IsOwner(userID) {
		return nil, domain.ErrForbidden
	}

	return mapTransactionResponse(tx), nil
}

func (s *transactionService) ListByAccount(ctx context.Context, userID, accountID uuid.UUID, filter *TransactionListFilter) ([]*TransactionResponse, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if !account.IsOwner(userID) {
		return nil, domain.ErrForbidden
	}

	limit := 50
	offset := 0
	var startDate *time.Time
	var endDate *time.Time
	var txType string
	if filter != nil {
		limit = filter.Limit
		offset = filter.Offset
		startDate = filter.StartDate
		endDate = filter.EndDate
		txType = strings.TrimSpace(filter.TransactionType)
	}

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var transactions []*domain.Transaction
	if startDate != nil || endDate != nil {
		from := time.Unix(0, 0).UTC()
		to := time.Now().UTC().Add(24 * time.Hour)
		if startDate != nil {
			from = startDate.UTC()
		}
		if endDate != nil {
			to = endDate.UTC()
		}
		if from.After(to) {
			return nil, domain.ErrInvalidTransfer
		}
		transactions, err = s.transactionRepo.ListByAccountInDateRange(ctx, accountID, from, to, limit, offset)
	} else {
		transactions, err = s.transactionRepo.ListByAccount(ctx, accountID, limit, offset)
	}
	if err != nil {
		return nil, err
	}

	txType = strings.ToLower(txType)
	if txType != "" && !domain.TransactionType(txType).IsValid() {
		return nil, domain.ErrInvalidTransactionType
	}

	items := make([]*TransactionResponse, 0, len(transactions))
	for _, tx := range transactions {
		if txType != "" && strings.ToLower(string(tx.TransactionType)) != txType {
			continue
		}
		items = append(items, mapTransactionResponse(tx))
	}

	return items, nil
}

func mapTransactionResponse(tx *domain.Transaction) *TransactionResponse {
	if tx == nil {
		return nil
	}

	return &TransactionResponse{
		ID:              tx.ID,
		AccountID:       tx.AccountID,
		TransactionType: string(tx.TransactionType),
		Amount:          tx.Amount.String(),
		BalanceAfter:    tx.BalanceAfter.String(),
		ReferenceID:     tx.ReferenceID,
		Description:     tx.Description,
		CreatedAt:       tx.CreatedAt,
	}
}
