package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type UserService interface {
	Register(ctx context.Context, req *RegisterRequest) (*UserResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*UserResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest) error
}

type AccountService interface {
	CreateAccount(ctx context.Context, userID uuid.UUID, req *CreateAccountRequest) (*AccountResponse, error)
	GetAccount(ctx context.Context, userID, accountID uuid.UUID) (*AccountResponse, error)
	ListAccounts(ctx context.Context, userID uuid.UUID) ([]*AccountResponse, error)
	GetBalance(ctx context.Context, userID, accountID uuid.UUID) (*BalanceResponse, error)
}

type TransferService interface {
	Transfer(ctx context.Context, userID uuid.UUID, req *TransferRequest) (*TransferResponse, error)
	GetTransfer(ctx context.Context, userID, transferID uuid.UUID) (*TransferResponse, error)
	ListTransfers(ctx context.Context, userID, accountID uuid.UUID, limit, offset int) ([]*TransferResponse, error)
}

type TransactionService interface {
	GetByID(ctx context.Context, userID, transactionID uuid.UUID) (*TransactionResponse, error)
	ListByAccount(ctx context.Context, userID, accountID uuid.UUID, filter *TransactionListFilter) ([]*TransactionResponse, error)
}

type TransactionListFilter struct {
	TransactionType string
	StartDate       *time.Time
	EndDate         *time.Time
	Limit           int
	Offset          int
}

type RegisterRequest struct {
	Email    string
	Password string
	FullName string
}

type LoginRequest struct {
	Email    string
	Password string
}

type UpdateProfileRequest struct {
	Email    string
	FullName string
}

type UserResponse struct {
	ID        uuid.UUID
	Email     string
	FullName  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AuthResponse struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type CreateAccountRequest struct {
	AccountType string
	Currency    string
}

type AccountResponse struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	AccountNumber string
	AccountType   string
	Balance       string
	Currency      string
	Status        string
	Version       int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type BalanceResponse struct {
	AccountID uuid.UUID
	Balance   string
	Currency  string
	Version   int
}

type TransferRequest struct {
	FromAccountID  uuid.UUID
	ToAccountID    uuid.UUID
	Amount         string
	Currency       string
	Description    string
	IdempotencyKey string
}

type TransferResponse struct {
	ID              uuid.UUID
	FromAccountID   uuid.UUID
	ToAccountID     uuid.UUID
	Amount          string
	Currency        string
	Status          string
	ReferenceNumber string
	Description     string
	CreatedAt       time.Time
	CompletedAt     *time.Time
}

type TransactionResponse struct {
	ID              uuid.UUID
	AccountID       uuid.UUID
	TransactionType string
	Amount          string
	BalanceAfter    string
	ReferenceID     *uuid.UUID
	Description     string
	CreatedAt       time.Time
}
