package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Kisotu/neobank/internal/db"
	"github.com/Kisotu/neobank/internal/domain"
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

	created, err := r.queries.CreateTransaction(ctx, &db.CreateTransactionParams{
		AccountID:       toPgUUID(tx.AccountID),
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

func (r *transactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	row, err := r.queries.GetTransactionByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, fmt.Errorf("get transaction by id: %w", err)
	}

	return toDomainTransaction(row), nil
}

func (r *transactionRepository) ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*domain.Transaction, error) {
	rows, err := r.queries.ListTransactionsByAccountID(ctx, &db.ListTransactionsByAccountIDParams{
		AccountID: toPgUUID(accountID),
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

func (r *transactionRepository) ListByAccountAndType(ctx context.Context, accountID uuid.UUID, txType domain.TransactionType, limit, offset int) ([]*domain.Transaction, error) {
	rows, err := r.queries.ListTransactionsByAccountIDAndType(ctx, &db.ListTransactionsByAccountIDAndTypeParams{
		AccountID:       toPgUUID(accountID),
		TransactionType: string(txType),
		Limit:           int32(limit),
		Offset:          int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list transactions by account and type: %w", err)
	}

	transactions := make([]*domain.Transaction, 0, len(rows))
	for _, row := range rows {
		transactions = append(transactions, toDomainTransaction(row))
	}

	return transactions, nil
}

func (r *transactionRepository) ListByAccountInDateRange(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time, limit, offset int) ([]*domain.Transaction, error) {
	rows, err := r.queries.ListTransactionsByDateRange(ctx, &db.ListTransactionsByDateRangeParams{
		AccountID:   toPgUUID(accountID),
		CreatedAt:   pgtype.Timestamptz{Time: startDate, Valid: true},
		CreatedAt_2: pgtype.Timestamptz{Time: endDate, Valid: true},
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list transactions by date range: %w", err)
	}

	transactions := make([]*domain.Transaction, 0, len(rows))
	for _, row := range rows {
		transactions = append(transactions, toDomainTransaction(row))
	}

	return transactions, nil
}

func (r *transactionRepository) ListByAccountInDateRangeAndType(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time, txType domain.TransactionType, limit, offset int) ([]*domain.Transaction, error) {
	rows, err := r.queries.ListTransactionsByDateRangeAndType(ctx, &db.ListTransactionsByDateRangeAndTypeParams{
		AccountID:       toPgUUID(accountID),
		CreatedAt:       pgtype.Timestamptz{Time: startDate, Valid: true},
		CreatedAt_2:     pgtype.Timestamptz{Time: endDate, Valid: true},
		TransactionType: string(txType),
		Limit:           int32(limit),
		Offset:          int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list transactions by date range and type: %w", err)
	}

	transactions := make([]*domain.Transaction, 0, len(rows))
	for _, row := range rows {
		transactions = append(transactions, toDomainTransaction(row))
	}

	return transactions, nil
}
