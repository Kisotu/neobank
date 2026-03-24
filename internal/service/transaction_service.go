package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/username/banking-app/internal/domain"
	"github.com/username/banking-app/internal/repository"
)

type transactionService struct {
	transactionRepo repository.TransactionRepository
	logger          *slog.Logger
}

func NewTransactionService(transactionRepo repository.TransactionRepository, logger *slog.Logger) TransactionService {
	if logger == nil {
		logger = slog.Default()
	}

	return &transactionService{
		transactionRepo: transactionRepo,
		logger:          logger,
	}
}

func (s *transactionService) GetByID(ctx context.Context, transactionID uuid.UUID) (*TransactionResponse, error) {
	tx, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	return mapTransactionResponse(tx), nil
}

func (s *transactionService) ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*TransactionResponse, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	transactions, err := s.transactionRepo.ListByAccount(ctx, accountID, limit, offset)
	if err != nil {
		return nil, err
	}

	items := make([]*TransactionResponse, 0, len(transactions))
	for _, tx := range transactions {
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
