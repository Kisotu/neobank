package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/Kisotu/neobank/internal/domain"
)

type updateBalanceCall struct {
	id      uuid.UUID
	balance decimal.Decimal
	version int
}

type fakeAccountRepo struct {
	createFn        func(ctx context.Context, account *domain.Account) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*domain.Account, error)
	getByNumberFn   func(ctx context.Context, number string) (*domain.Account, error)
	listByUserIDFn  func(ctx context.Context, userID uuid.UUID) ([]*domain.Account, error)
	updateBalanceFn func(ctx context.Context, id uuid.UUID, balance decimal.Decimal, version int) error
	lockForUpdateFn func(ctx context.Context, ids ...uuid.UUID) ([]*domain.Account, error)

	updateCalls []updateBalanceCall
}

func (f *fakeAccountRepo) Create(ctx context.Context, account *domain.Account) error {
	if f.createFn != nil {
		return f.createFn(ctx, account)
	}
	return nil
}

func (f *fakeAccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return nil, domain.ErrAccountNotFound
}

func (f *fakeAccountRepo) GetByNumber(ctx context.Context, number string) (*domain.Account, error) {
	if f.getByNumberFn != nil {
		return f.getByNumberFn(ctx, number)
	}
	return nil, domain.ErrAccountNotFound
}

func (f *fakeAccountRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Account, error) {
	if f.listByUserIDFn != nil {
		return f.listByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func (f *fakeAccountRepo) UpdateBalance(ctx context.Context, id uuid.UUID, balance decimal.Decimal, version int) error {
	f.updateCalls = append(f.updateCalls, updateBalanceCall{id: id, balance: balance, version: version})
	if f.updateBalanceFn != nil {
		return f.updateBalanceFn(ctx, id, balance, version)
	}
	return nil
}

func (f *fakeAccountRepo) LockForUpdate(ctx context.Context, ids ...uuid.UUID) ([]*domain.Account, error) {
	if f.lockForUpdateFn != nil {
		return f.lockForUpdateFn(ctx, ids...)
	}
	return nil, nil
}

type fakeTransferRepo struct {
	createFn        func(ctx context.Context, transfer *domain.Transfer) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*domain.Transfer, error)
	getByRefFn      func(ctx context.Context, ref string) (*domain.Transfer, error)
	updateStatusFn  func(ctx context.Context, id uuid.UUID, status domain.TransferStatus) error
	listByAccountFn func(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*domain.Transfer, error)

	createCalls       int
	updateStatusCalls int
}

func (f *fakeTransferRepo) Create(ctx context.Context, transfer *domain.Transfer) error {
	f.createCalls++
	if f.createFn != nil {
		return f.createFn(ctx, transfer)
	}
	transfer.ID = uuid.New()
	transfer.CreatedAt = time.Now().UTC()
	return nil
}

func (f *fakeTransferRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transfer, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return nil, domain.ErrInvalidTransfer
}

func (f *fakeTransferRepo) GetByReference(ctx context.Context, ref string) (*domain.Transfer, error) {
	if f.getByRefFn != nil {
		return f.getByRefFn(ctx, ref)
	}
	return nil, domain.ErrInvalidTransfer
}

func (f *fakeTransferRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TransferStatus) error {
	f.updateStatusCalls++
	if f.updateStatusFn != nil {
		return f.updateStatusFn(ctx, id, status)
	}
	return nil
}

func (f *fakeTransferRepo) ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*domain.Transfer, error) {
	if f.listByAccountFn != nil {
		return f.listByAccountFn(ctx, accountID, limit, offset)
	}
	return nil, nil
}

type fakeTransactionRepo struct {
	createFn              func(ctx context.Context, tx *domain.Transaction) error
	getByIDFn             func(ctx context.Context, id uuid.UUID) (*domain.Transaction, error)
	listByAccountFn       func(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*domain.Transaction, error)
	listByDateRangeFn     func(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time, limit, offset int) ([]*domain.Transaction, error)
	createCalls           int
	createdTransactionLog []*domain.Transaction
}

func (f *fakeTransactionRepo) Create(ctx context.Context, tx *domain.Transaction) error {
	f.createCalls++
	copyTx := *tx
	f.createdTransactionLog = append(f.createdTransactionLog, &copyTx)
	if f.createFn != nil {
		return f.createFn(ctx, tx)
	}
	return nil
}

func (f *fakeTransactionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return nil, domain.ErrTransactionNotFound
}

func (f *fakeTransactionRepo) ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*domain.Transaction, error) {
	if f.listByAccountFn != nil {
		return f.listByAccountFn(ctx, accountID, limit, offset)
	}
	return nil, nil
}

func (f *fakeTransactionRepo) ListByAccountInDateRange(ctx context.Context, accountID uuid.UUID, startDate, endDate time.Time, limit, offset int) ([]*domain.Transaction, error) {
	if f.listByDateRangeFn != nil {
		return f.listByDateRangeFn(ctx, accountID, startDate, endDate, limit, offset)
	}
	return nil, nil
}

func TestTransferService_Transfer_Scenarios(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{t: t}, nil))
	userID := uuid.New()
	fromID := uuid.New()
	toID := uuid.New()

	newHappyTxDeps := func(owner uuid.UUID, fromBalance, toBalance int64) transferTxDeps {
		accountRepo := &fakeAccountRepo{}
		accountRepo.lockForUpdateFn = func(_ context.Context, ids ...uuid.UUID) ([]*domain.Account, error) {
			if len(ids) != 2 {
				return nil, fmt.Errorf("expected 2 account ids, got %d", len(ids))
			}
			return []*domain.Account{
				{
					ID:       fromID,
					UserID:   owner,
					Balance:  decimal.NewFromInt(fromBalance),
					Currency: "USD",
					Status:   domain.AccountStatusActive,
					Version:  1,
				},
				{
					ID:       toID,
					UserID:   uuid.New(),
					Balance:  decimal.NewFromInt(toBalance),
					Currency: "USD",
					Status:   domain.AccountStatusActive,
					Version:  3,
				},
			}, nil
		}

		transferRepo := &fakeTransferRepo{}
		transactionRepo := &fakeTransactionRepo{}
		return transferTxDeps{
			accountRepo:     accountRepo,
			transferRepo:    transferRepo,
			transactionRepo: transactionRepo,
		}
	}

	tests := []struct {
		name          string
		req           *TransferRequest
		arrange       func() (*transferService, *fakeAccountRepo, *fakeTransferRepo, *fakeTransactionRepo, *int)
		wantErrIs     error
		wantRunInTx   int
		wantTxCreates int
		wantUpdate    int
		wantStatus    int
	}{
		{
			name: "happy path",
			req: &TransferRequest{
				FromAccountID: fromID,
				ToAccountID:   toID,
				Amount:        "10.00",
				Currency:      "USD",
			},
			arrange: func() (*transferService, *fakeAccountRepo, *fakeTransferRepo, *fakeTransactionRepo, *int) {
				runs := 0
				txDeps := newHappyTxDeps(userID, 100, 50)
				baseAccountRepo := &fakeAccountRepo{}
				baseTransferRepo := &fakeTransferRepo{}
				svc := newTransferServiceWithDependencies(baseAccountRepo, baseTransferRepo, func(ctx context.Context, fn func(deps transferTxDeps) error) error {
					runs++
					return fn(txDeps)
				}, logger)
				return svc, txDeps.accountRepo.(*fakeAccountRepo), txDeps.transferRepo.(*fakeTransferRepo), txDeps.transactionRepo.(*fakeTransactionRepo), &runs
			},
			wantRunInTx:   1,
			wantTxCreates: 2,
			wantUpdate:    2,
			wantStatus:    1,
		},
		{
			name: "insufficient funds",
			req: &TransferRequest{
				FromAccountID: fromID,
				ToAccountID:   toID,
				Amount:        "10",
				Currency:      "USD",
			},
			arrange: func() (*transferService, *fakeAccountRepo, *fakeTransferRepo, *fakeTransactionRepo, *int) {
				runs := 0
				txDeps := newHappyTxDeps(userID, 5, 50)
				svc := newTransferServiceWithDependencies(&fakeAccountRepo{}, &fakeTransferRepo{}, func(ctx context.Context, fn func(deps transferTxDeps) error) error {
					runs++
					return fn(txDeps)
				}, logger)
				return svc, txDeps.accountRepo.(*fakeAccountRepo), txDeps.transferRepo.(*fakeTransferRepo), txDeps.transactionRepo.(*fakeTransactionRepo), &runs
			},
			wantErrIs:   domain.ErrInsufficientFunds,
			wantRunInTx: 1,
		},
		{
			name: "unauthorized account owner",
			req: &TransferRequest{
				FromAccountID: fromID,
				ToAccountID:   toID,
				Amount:        "10",
				Currency:      "USD",
			},
			arrange: func() (*transferService, *fakeAccountRepo, *fakeTransferRepo, *fakeTransactionRepo, *int) {
				runs := 0
				txDeps := newHappyTxDeps(uuid.New(), 100, 50)
				svc := newTransferServiceWithDependencies(&fakeAccountRepo{}, &fakeTransferRepo{}, func(ctx context.Context, fn func(deps transferTxDeps) error) error {
					runs++
					return fn(txDeps)
				}, logger)
				return svc, txDeps.accountRepo.(*fakeAccountRepo), txDeps.transferRepo.(*fakeTransferRepo), txDeps.transactionRepo.(*fakeTransactionRepo), &runs
			},
			wantErrIs:   domain.ErrForbidden,
			wantRunInTx: 1,
		},
		{
			name: "idempotency replay short-circuits",
			req: &TransferRequest{
				FromAccountID:  fromID,
				ToAccountID:    toID,
				Amount:         "10",
				Currency:       "USD",
				IdempotencyKey: "idem-1",
			},
			arrange: func() (*transferService, *fakeAccountRepo, *fakeTransferRepo, *fakeTransactionRepo, *int) {
				runs := 0
				existing := &domain.Transfer{
					ID:              uuid.New(),
					FromAccountID:   fromID,
					ToAccountID:     toID,
					Amount:          decimal.NewFromInt(10),
					Currency:        "USD",
					Status:          domain.TransferStatusCompleted,
					ReferenceNumber: "idem-1",
					CreatedAt:       time.Now().UTC(),
				}
				baseTransferRepo := &fakeTransferRepo{
					getByRefFn: func(context.Context, string) (*domain.Transfer, error) {
						return existing, nil
					},
				}
				svc := newTransferServiceWithDependencies(&fakeAccountRepo{}, baseTransferRepo, func(ctx context.Context, fn func(deps transferTxDeps) error) error {
					runs++
					return fn(transferTxDeps{})
				}, logger)
				return svc, &fakeAccountRepo{}, &fakeTransferRepo{}, &fakeTransactionRepo{}, &runs
			},
			wantRunInTx: 0,
		},
		{
			name: "retryable conflict then success",
			req: &TransferRequest{
				FromAccountID: fromID,
				ToAccountID:   toID,
				Amount:        "10",
				Currency:      "USD",
			},
			arrange: func() (*transferService, *fakeAccountRepo, *fakeTransferRepo, *fakeTransactionRepo, *int) {
				runs := 0
				txDeps := newHappyTxDeps(userID, 100, 50)
				svc := newTransferServiceWithDependencies(&fakeAccountRepo{}, &fakeTransferRepo{}, func(ctx context.Context, fn func(deps transferTxDeps) error) error {
					runs++
					if runs == 1 {
						return domain.ErrOptimisticLock
					}
					return fn(txDeps)
				}, logger)
				return svc, txDeps.accountRepo.(*fakeAccountRepo), txDeps.transferRepo.(*fakeTransferRepo), txDeps.transactionRepo.(*fakeTransactionRepo), &runs
			},
			wantRunInTx:   2,
			wantTxCreates: 2,
			wantUpdate:    2,
			wantStatus:    1,
		},
		{
			name: "same account transfer",
			req: &TransferRequest{
				FromAccountID: fromID,
				ToAccountID:   fromID,
				Amount:        "10",
				Currency:      "USD",
			},
			arrange: func() (*transferService, *fakeAccountRepo, *fakeTransferRepo, *fakeTransactionRepo, *int) {
				runs := 0
				svc := newTransferServiceWithDependencies(&fakeAccountRepo{}, &fakeTransferRepo{}, func(ctx context.Context, fn func(deps transferTxDeps) error) error {
					runs++
					return fn(transferTxDeps{})
				}, logger)
				return svc, &fakeAccountRepo{}, &fakeTransferRepo{}, &fakeTransactionRepo{}, &runs
			},
			wantErrIs:   domain.ErrSameAccountTransfer,
			wantRunInTx: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, accountRepo, transferRepo, txRepo, runs := tt.arrange()
			resp, err := svc.Transfer(context.Background(), userID, tt.req)

			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("expected error %v, got %v", tt.wantErrIs, err)
				}
				if resp != nil {
					t.Fatalf("expected nil response on error, got %#v", resp)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp == nil {
					t.Fatalf("expected non-nil response")
				}
			}

			if *runs != tt.wantRunInTx {
				t.Fatalf("expected runInTx calls %d, got %d", tt.wantRunInTx, *runs)
			}
			if txRepo.createCalls != tt.wantTxCreates {
				t.Fatalf("expected transaction create calls %d, got %d", tt.wantTxCreates, txRepo.createCalls)
			}
			if len(accountRepo.updateCalls) != tt.wantUpdate {
				t.Fatalf("expected account updates %d, got %d", tt.wantUpdate, len(accountRepo.updateCalls))
			}
			if transferRepo.updateStatusCalls != tt.wantStatus {
				t.Fatalf("expected update status calls %d, got %d", tt.wantStatus, transferRepo.updateStatusCalls)
			}
		})
	}
}

type testWriter struct {
	t *testing.T
}

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}
