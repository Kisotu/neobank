package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/username/banking-app/internal/db"
	"github.com/username/banking-app/internal/domain"
)

type accountRepository struct {
	queries *db.Queries
	logger  *slog.Logger
}

func NewAccountRepository(dbtx db.DBTX, logger *slog.Logger) AccountRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &accountRepository{
		queries: db.New(dbtx),
		logger:  logger,
	}
}

func (r *accountRepository) Create(ctx context.Context, account *domain.Account) error {
	if err := account.Validate(); err != nil {
		return err
	}

	created, err := r.queries.CreateAccount(ctx, &db.CreateAccountParams{
		UserID:        toPgUUID(account.UserID),
		AccountNumber: account.AccountNumber,
		AccountType:   string(account.AccountType),
		Balance:       account.Balance,
		Currency:      account.Currency,
		Status:        string(account.Status),
	})
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}

	*account = *toDomainAccount(created)
	r.logger.InfoContext(ctx, "account created", "account_id", account.ID.String(), "user_id", account.UserID.String())
	return nil
}

func (r *accountRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	row, err := r.queries.GetAccountByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, fmt.Errorf("get account by id: %w", err)
	}

	return toDomainAccount(row), nil
}

func (r *accountRepository) GetByNumber(ctx context.Context, number string) (*domain.Account, error) {
	row, err := r.queries.GetAccountByNumber(ctx, number)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, fmt.Errorf("get account by number: %w", err)
	}

	return toDomainAccount(row), nil
}

func (r *accountRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Account, error) {
	rows, err := r.queries.ListAccountsByUserID(ctx, toPgUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list accounts by user id: %w", err)
	}

	accounts := make([]*domain.Account, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, toDomainAccount(row))
	}

	return accounts, nil
}

func (r *accountRepository) UpdateBalance(ctx context.Context, id uuid.UUID, balance decimal.Decimal, version int) error {
	err := r.queries.UpdateAccountBalance(ctx, &db.UpdateAccountBalanceParams{
		ID:      toPgUUID(id),
		Balance: balance,
		Version: int32(version),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrOptimisticLock
		}
		return fmt.Errorf("update account balance: %w", err)
	}

	r.logger.InfoContext(ctx, "account balance updated", "account_id", id.String(), "version", version)
	return nil
}

func (r *accountRepository) LockForUpdate(ctx context.Context, ids ...uuid.UUID) ([]*domain.Account, error) {
	pgIDs := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		pgIDs = append(pgIDs, toPgUUID(id))
	}

	rows, err := r.queries.ListAccountsWithLock(ctx, pgIDs)
	if err != nil {
		return nil, fmt.Errorf("lock accounts for update: %w", err)
	}

	accounts := make([]*domain.Account, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, toDomainAccount(row))
	}

	return accounts, nil
}
