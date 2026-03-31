package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/Kisotu/neobank/internal/domain"
)

func TestTransactionService_ListByAccount_InvalidType(t *testing.T) {
	ownerID := uuid.New()
	accountID := uuid.New()

	accountRepo := &fakeAccountRepo{
		getByIDFn: func(context.Context, uuid.UUID) (*domain.Account, error) {
			return &domain.Account{ID: accountID, UserID: ownerID}, nil
		},
	}

	txRepo := &fakeTransactionRepo{
		listByAccountTypeFn: func(context.Context, uuid.UUID, domain.TransactionType, int, int) ([]*domain.Transaction, error) {
			t.Fatal("typed list query should not be called for invalid type")
			return nil, nil
		},
		listByAccountFn: func(context.Context, uuid.UUID, int, int) ([]*domain.Transaction, error) {
			t.Fatal("untyped list query should not be called for invalid type")
			return nil, nil
		},
	}

	svc := NewTransactionService(txRepo, accountRepo, nil)

	_, err := svc.ListByAccount(context.Background(), ownerID, accountID, &TransactionListFilter{TransactionType: "not-a-type"})
	if !errors.Is(err, domain.ErrInvalidTransactionType) {
		t.Fatalf("expected ErrInvalidTransactionType, got %v", err)
	}
}

func TestTransactionService_ListByAccount_UsesSQLTypeFilter(t *testing.T) {
	ownerID := uuid.New()
	accountID := uuid.New()

	accountRepo := &fakeAccountRepo{
		getByIDFn: func(context.Context, uuid.UUID) (*domain.Account, error) {
			return &domain.Account{ID: accountID, UserID: ownerID}, nil
		},
	}

	calledTyped := false
	calledUntyped := false
	txRepo := &fakeTransactionRepo{
		listByAccountTypeFn: func(_ context.Context, gotAccountID uuid.UUID, txType domain.TransactionType, limit, offset int) ([]*domain.Transaction, error) {
			calledTyped = true
			if gotAccountID != accountID {
				t.Fatalf("expected account id %s, got %s", accountID, gotAccountID)
			}
			if txType != domain.TransactionTypeDeposit {
				t.Fatalf("expected tx type %q, got %q", domain.TransactionTypeDeposit, txType)
			}
			if limit != 25 || offset != 5 {
				t.Fatalf("expected limit/offset 25/5, got %d/%d", limit, offset)
			}
			return []*domain.Transaction{{
				ID:              uuid.New(),
				AccountID:       accountID,
				TransactionType: domain.TransactionTypeDeposit,
				Amount:          decimal.RequireFromString("10.00"),
				BalanceAfter:    decimal.RequireFromString("100.00"),
				CreatedAt:       time.Now().UTC(),
			}}, nil
		},
		listByAccountFn: func(context.Context, uuid.UUID, int, int) ([]*domain.Transaction, error) {
			calledUntyped = true
			return nil, nil
		},
	}

	svc := NewTransactionService(txRepo, accountRepo, nil)

	items, err := svc.ListByAccount(context.Background(), ownerID, accountID, &TransactionListFilter{
		TransactionType: "DEPOSIT",
		Limit:           25,
		Offset:          5,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !calledTyped {
		t.Fatal("expected typed repository method to be called")
	}
	if calledUntyped {
		t.Fatal("expected untyped repository method not to be called")
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 transaction response, got %d", len(items))
	}
	if items[0].TransactionType != string(domain.TransactionTypeDeposit) {
		t.Fatalf("expected response transaction type %q, got %q", domain.TransactionTypeDeposit, items[0].TransactionType)
	}
}

func TestTransactionService_ListByAccount_UsesDateRangeSQLTypeFilter(t *testing.T) {
	ownerID := uuid.New()
	accountID := uuid.New()
	from := time.Now().UTC().Add(-24 * time.Hour)
	to := time.Now().UTC()

	accountRepo := &fakeAccountRepo{
		getByIDFn: func(context.Context, uuid.UUID) (*domain.Account, error) {
			return &domain.Account{ID: accountID, UserID: ownerID}, nil
		},
	}

	calledTyped := false
	calledUntyped := false
	txRepo := &fakeTransactionRepo{
		listByDateRangeTypeFn: func(_ context.Context, gotAccountID uuid.UUID, gotFrom, gotTo time.Time, txType domain.TransactionType, limit, offset int) ([]*domain.Transaction, error) {
			calledTyped = true
			if gotAccountID != accountID {
				t.Fatalf("expected account id %s, got %s", accountID, gotAccountID)
			}
			if txType != domain.TransactionTypeTransferOut {
				t.Fatalf("expected tx type %q, got %q", domain.TransactionTypeTransferOut, txType)
			}
			if !gotFrom.Equal(from) || !gotTo.Equal(to) {
				t.Fatalf("expected date range %v..%v, got %v..%v", from, to, gotFrom, gotTo)
			}
			if limit != 10 || offset != 2 {
				t.Fatalf("expected limit/offset 10/2, got %d/%d", limit, offset)
			}
			return []*domain.Transaction{}, nil
		},
		listByDateRangeFn: func(context.Context, uuid.UUID, time.Time, time.Time, int, int) ([]*domain.Transaction, error) {
			calledUntyped = true
			return nil, nil
		},
	}

	svc := NewTransactionService(txRepo, accountRepo, nil)

	_, err := svc.ListByAccount(context.Background(), ownerID, accountID, &TransactionListFilter{
		TransactionType: string(domain.TransactionTypeTransferOut),
		StartDate:       &from,
		EndDate:         &to,
		Limit:           10,
		Offset:          2,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !calledTyped {
		t.Fatal("expected date-range typed repository method to be called")
	}
	if calledUntyped {
		t.Fatal("expected date-range untyped repository method not to be called")
	}
}
