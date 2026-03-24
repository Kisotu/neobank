package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/username/banking-app/internal/db"
	"github.com/username/banking-app/internal/domain"
)

type transactionRepository struct {
	queries *db.Queries
	logger  *slog.Logger
}

func NewTransactionRepository(dbtx db.DBTX, logger *slog.Logger) TransactionRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &transactionRepository{
		queries: db.New(dbtx),
		logger:  logger,
	}
}

func (r *transactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
	if err := tx.Validate(); err != nil {
		return err
	}

	created, err := r.queries.CreateTransaction(ctx, db.CreateTransactionParams{
		AccountID:       tx.AccountID,
		TransactionType: string(tx.TransactionType),
		Amount:          tx.Amount,
		BalanceAfter:    tx.BalanceAfter,
		ReferenceID:     toNullableUUID(tx.ReferenceID),
		Description:     toNullableText(tx.Description),
	})
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	*tx = *toDomainTransaction(created)
	r.logger.InfoContext(ctx, "transaction created", "transaction_id", tx.ID.String(), "account_id", tx.AccountID.String())
	return nil
}

func (r *transactionRepository) ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*domain.Transaction, error) {
	rows, err := r.queries.ListTransactionsByAccountID(ctx, db.ListTransactionsByAccountIDParams{
		AccountID: accountID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list transactions by account: %w", err)
	}

	transactions := make([]*domain.Transaction, 0, len(rows))
	for _, row := range rows {
		transactions = append(transactions, toDomainTransaction(row))
	}

	return transactions, nil
}