package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/username/banking-app/internal/domain"
	"github.com/username/banking-app/internal/domain/vo"
	"github.com/username/banking-app/internal/repository"
)

type accountService struct {
	accountRepo repository.AccountRepository
	userRepo    repository.UserRepository
	logger      *slog.Logger
}

func NewAccountService(accountRepo repository.AccountRepository, userRepo repository.UserRepository, logger *slog.Logger) AccountService {
	if logger == nil {
		logger = slog.Default()
	}

	return &accountService{
		accountRepo: accountRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

func (s *accountService) CreateAccount(ctx context.Context, userID uuid.UUID, req *CreateAccountRequest) (*AccountResponse, error) {
	if req == nil {
		return nil, domain.ErrInvalidAccountType
	}

	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return nil, err
	}

	accountType := domain.AccountType(strings.ToLower(strings.TrimSpace(req.AccountType)))
	if !accountType.IsValid() {
		return nil, domain.ErrInvalidAccountType
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "USD"
	}

	accountNumber, err := vo.GenerateAccountNumber()
	if err != nil {
		return nil, fmt.Errorf("generate account number: %w", err)
	}

	account := &domain.Account{
		UserID:        userID,
		AccountNumber: accountNumber.String(),
		AccountType:   accountType,
		Balance:       decimal.Zero,
		Currency:      currency,
		Status:        domain.AccountStatusActive,
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, err
	}

	return mapAccountResponse(account), nil
}

func (s *accountService) GetAccount(ctx context.Context, accountID uuid.UUID) (*AccountResponse, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return mapAccountResponse(account), nil
}

func (s *accountService) ListAccounts(ctx context.Context, userID uuid.UUID) ([]*AccountResponse, error) {
	accounts, err := s.accountRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]*AccountResponse, 0, len(accounts))
	for _, account := range accounts {
		items = append(items, mapAccountResponse(account))
	}

	return items, nil
}

func (s *accountService) GetBalance(ctx context.Context, accountID uuid.UUID) (*BalanceResponse, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return &BalanceResponse{
		AccountID: account.ID,
		Balance:   account.Balance.String(),
		Currency:  account.Currency,
		Version:   account.Version,
	}, nil
}

func mapAccountResponse(account *domain.Account) *AccountResponse {
	if account == nil {
		return nil
	}

	return &AccountResponse{
		ID:            account.ID,
		UserID:        account.UserID,
		AccountNumber: account.AccountNumber,
		AccountType:   string(account.AccountType),
		Balance:       account.Balance.String(),
		Currency:      account.Currency,
		Status:        string(account.Status),
		Version:       account.Version,
		CreatedAt:     account.CreatedAt,
		UpdatedAt:     account.UpdatedAt,
	}
}
