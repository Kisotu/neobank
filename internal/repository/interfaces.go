package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/username/banking-app/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type AccountRepository interface {
	Create(ctx context.Context, account *domain.Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error)
	GetByNumber(ctx context.Context, number string) (*domain.Account, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Account, error)
	UpdateBalance(ctx context.Context, id uuid.UUID, balance decimal.Decimal, version int) error
	LockForUpdate(ctx context.Context, ids ...uuid.UUID) ([]*domain.Account, error)
}

type TransferRepository interface {
	Create(ctx context.Context, transfer *domain.Transfer) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Transfer, error)
	GetByReference(ctx context.Context, ref string) (*domain.Transfer, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TransferStatus) error
	ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*domain.Transfer, error)
}

type TransactionRepository interface {
	Create(ctx context.Context, tx *domain.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error)
	ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*domain.Transaction, error)
	ListByAccountInDateRange(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time, limit, offset int) ([]*domain.Transaction, error)
}
