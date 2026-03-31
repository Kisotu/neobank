package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Kisotu/neobank/internal/domain"
	"github.com/Kisotu/neobank/internal/repository"
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

type transferIntegrationFixture struct {
	fromUserID       uuid.UUID
	toUserID         uuid.UUID
	fromAccountID    uuid.UUID
	toAccountID      uuid.UUID
	initialFrom      decimal.Decimal
	initialTo        decimal.Decimal
	referenceKeySeed string
}

func TestTransferService_Transfer_IdempotencyIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	if !integrationSchemaReady(ctx, pool) {
		t.Skip("database schema not initialized; run migrations before integration tests")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewTransferService(pool, logger)

	t.Run("same request same key immediate repeat", func(t *testing.T) {
		fixture, cleanup := seedTransferFixture(t, ctx, pool, logger, decimal.RequireFromString("100.0000"), decimal.RequireFromString("50.0000"))
		defer cleanup()

		request := &TransferRequest{
			FromAccountID:  fixture.fromAccountID,
			ToAccountID:    fixture.toAccountID,
			Amount:         "10.0000",
			Currency:       "USD",
			Description:    "idempotency immediate",
			IdempotencyKey: fixture.referenceKeySeed + "-immediate",
		}

		first, err := svc.Transfer(ctx, fixture.fromUserID, request)
		if err != nil {
			t.Fatalf("first transfer failed: %v", err)
		}

		second, err := svc.Transfer(ctx, fixture.fromUserID, request)
		if err != nil {
			t.Fatalf("second transfer failed: %v", err)
		}

		assertIdempotentTransferResult(t, first, second)
		assertSingleBusinessEffect(t, ctx, pool, logger, fixture, first, decimal.RequireFromString("90.0000"), decimal.RequireFromString("60.0000"), request.IdempotencyKey)
	})

	t.Run("same request same key delayed repeat", func(t *testing.T) {
		fixture, cleanup := seedTransferFixture(t, ctx, pool, logger, decimal.RequireFromString("120.0000"), decimal.RequireFromString("10.0000"))
		defer cleanup()

		request := &TransferRequest{
			FromAccountID:  fixture.fromAccountID,
			ToAccountID:    fixture.toAccountID,
			Amount:         "20.0000",
			Currency:       "USD",
			Description:    "idempotency delayed",
			IdempotencyKey: fixture.referenceKeySeed + "-delayed",
		}

		first, err := svc.Transfer(ctx, fixture.fromUserID, request)
		if err != nil {
			t.Fatalf("first transfer failed: %v", err)
		}

		time.Sleep(25 * time.Millisecond)

		second, err := svc.Transfer(ctx, fixture.fromUserID, request)
		if err != nil {
			t.Fatalf("second transfer failed: %v", err)
		}

		assertIdempotentTransferResult(t, first, second)
		assertSingleBusinessEffect(t, ctx, pool, logger, fixture, first, decimal.RequireFromString("100.0000"), decimal.RequireFromString("30.0000"), request.IdempotencyKey)
	})

	t.Run("same key changed payload returns original transfer", func(t *testing.T) {
		fixture, cleanup := seedTransferFixture(t, ctx, pool, logger, decimal.RequireFromString("200.0000"), decimal.RequireFromString("80.0000"))
		defer cleanup()

		idempotencyKey := fixture.referenceKeySeed + "-changed"
		firstRequest := &TransferRequest{
			FromAccountID:  fixture.fromAccountID,
			ToAccountID:    fixture.toAccountID,
			Amount:         "15.0000",
			Currency:       "USD",
			Description:    "first payload",
			IdempotencyKey: idempotencyKey,
		}

		first, err := svc.Transfer(ctx, fixture.fromUserID, firstRequest)
		if err != nil {
			t.Fatalf("first transfer failed: %v", err)
		}

		secondRequest := &TransferRequest{
			FromAccountID:  fixture.fromAccountID,
			ToAccountID:    fixture.toAccountID,
			Amount:         "50.0000",
			Currency:       "USD",
			Description:    "second payload",
			IdempotencyKey: idempotencyKey,
		}

		second, err := svc.Transfer(ctx, fixture.fromUserID, secondRequest)
		if err != nil {
			t.Fatalf("second transfer with changed payload failed: %v", err)
		}

		assertIdempotentTransferResult(t, first, second)
		if second.Amount != first.Amount {
			t.Fatalf("expected replay amount %s, got %s", first.Amount, second.Amount)
		}
		assertSingleBusinessEffect(t, ctx, pool, logger, fixture, first, decimal.RequireFromString("185.0000"), decimal.RequireFromString("95.0000"), idempotencyKey)
	})
}

type concurrentAccountFixture struct {
	userID    uuid.UUID
	accountID uuid.UUID
	balance   decimal.Decimal
}

type concurrentTransferOperation struct {
	fromAccountID uuid.UUID
	toAccountID   uuid.UUID
	ownerUserID   uuid.UUID
	amount        string
	description   string
	idemKey       string
}

func TestTransferService_Transfer_ConcurrencyIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	if !integrationSchemaReady(ctx, pool) {
		t.Skip("database schema not initialized; run migrations before integration tests")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewTransferService(pool, logger)

	t.Run("low contention A to B and B to A", func(t *testing.T) {
		accounts, cleanup := seedConcurrentAccounts(t, ctx, pool, logger, 2, decimal.RequireFromString("5000.0000"))
		defer cleanup()

		scenario := "concurrency-low"
		ops := make([]concurrentTransferOperation, 0, 24)
		for i := 0; i < 24; i++ {
			from := accounts[i%2]
			to := accounts[(i+1)%2]
			ops = append(ops, concurrentTransferOperation{
				fromAccountID: from.accountID,
				toAccountID:   to.accountID,
				ownerUserID:   from.userID,
				amount:        "1.0000",
				description:   scenario,
				idemKey:       fmt.Sprintf("%s-%03d", scenario, i),
			})
		}

		errs := runConcurrentTransfers(t, svc, ops)
		assertNoTransferErrors(t, errs)
		assertConcurrencyInvariants(t, ctx, pool, logger, accounts, ops, scenario)
	})

	t.Run("medium contention fan in fan out", func(t *testing.T) {
		accounts, cleanup := seedConcurrentAccounts(t, ctx, pool, logger, 4, decimal.RequireFromString("5000.0000"))
		defer cleanup()

		hub := accounts[0]
		spokes := accounts[1:]
		scenario := "concurrency-medium"
		ops := make([]concurrentTransferOperation, 0, 90)
		for i := 0; i < 90; i++ {
			spoke := spokes[i%len(spokes)]
			if i%2 == 0 {
				ops = append(ops, concurrentTransferOperation{
					fromAccountID: spoke.accountID,
					toAccountID:   hub.accountID,
					ownerUserID:   spoke.userID,
					amount:        "1.0000",
					description:   scenario,
					idemKey:       fmt.Sprintf("%s-%03d", scenario, i),
				})
				continue
			}

			ops = append(ops, concurrentTransferOperation{
				fromAccountID: hub.accountID,
				toAccountID:   spoke.accountID,
				ownerUserID:   hub.userID,
				amount:        "1.0000",
				description:   scenario,
				idemKey:       fmt.Sprintf("%s-%03d", scenario, i),
			})
		}

		errs := runConcurrentTransfers(t, svc, ops)
		assertNoTransferErrors(t, errs)
		assertConcurrencyInvariants(t, ctx, pool, logger, accounts, ops, scenario)
	})

	t.Run("high contention burst on shared hub", func(t *testing.T) {
		accounts, cleanup := seedConcurrentAccounts(t, ctx, pool, logger, 6, decimal.RequireFromString("10000.0000"))
		defer cleanup()

		hub := accounts[0]
		spokes := accounts[1:]
		scenario := "concurrency-high"
		ops := make([]concurrentTransferOperation, 0, 300)
		for i := 0; i < 300; i++ {
			spoke := spokes[i%len(spokes)]
			if i%3 == 0 {
				ops = append(ops, concurrentTransferOperation{
					fromAccountID: hub.accountID,
					toAccountID:   spoke.accountID,
					ownerUserID:   hub.userID,
					amount:        "1.0000",
					description:   scenario,
					idemKey:       fmt.Sprintf("%s-%03d", scenario, i),
				})
				continue
			}

			ops = append(ops, concurrentTransferOperation{
				fromAccountID: spoke.accountID,
				toAccountID:   hub.accountID,
				ownerUserID:   spoke.userID,
				amount:        "1.0000",
				description:   scenario,
				idemKey:       fmt.Sprintf("%s-%03d", scenario, i),
			})
		}

		errs := runConcurrentTransfers(t, svc, ops)
		assertNoTransferErrors(t, errs)
		assertConcurrencyInvariants(t, ctx, pool, logger, accounts, ops, scenario)
	})
}

func seedConcurrentAccounts(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	count int,
	initialBalance decimal.Decimal,
) ([]concurrentAccountFixture, func()) {
	t.Helper()

	userRepo := repository.NewUserRepository(pool, logger)
	accountRepo := repository.NewAccountRepository(pool, logger)
	prefix := uuid.NewString()

	fixtures := make([]concurrentAccountFixture, 0, count)
	userIDs := make([]uuid.UUID, 0, count)
	accountIDs := make([]uuid.UUID, 0, count)

	for i := 0; i < count; i++ {
		user := &domain.User{
			Email:        fmt.Sprintf("%s-conc-user-%02d@example.com", prefix, i),
			PasswordHash: "hash",
			FullName:     fmt.Sprintf("Concurrent User %d", i),
			Status:       domain.UserStatusActive,
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create concurrent user %d: %v", i, err)
		}

		account := &domain.Account{
			UserID:        user.ID,
			AccountNumber: fmt.Sprintf("C%017d", time.Now().UnixNano()+int64(i)),
			AccountType:   domain.AccountTypeChecking,
			Balance:       initialBalance,
			Currency:      "USD",
			Status:        domain.AccountStatusActive,
		}
		if err := accountRepo.Create(ctx, account); err != nil {
			t.Fatalf("create concurrent account %d: %v", i, err)
		}

		fixtures = append(fixtures, concurrentAccountFixture{
			userID:    user.ID,
			accountID: account.ID,
			balance:   initialBalance,
		})
		userIDs = append(userIDs, user.ID)
		accountIDs = append(accountIDs, account.ID)
	}

	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM transactions WHERE account_id = ANY($1)`, accountIDs)
		_, _ = pool.Exec(ctx, `DELETE FROM transfers WHERE from_account_id = ANY($1) OR to_account_id = ANY($1)`, accountIDs)
		_, _ = pool.Exec(ctx, `DELETE FROM accounts WHERE id = ANY($1)`, accountIDs)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = ANY($1)`, userIDs)
	}

	return fixtures, cleanup
}

func runConcurrentTransfers(t *testing.T, svc TransferService, ops []concurrentTransferOperation) []error {
	t.Helper()

	start := make(chan struct{})
	errs := make([]error, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := range ops {
		op := ops[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			_, err := svc.Transfer(ctx, op.ownerUserID, &TransferRequest{
				FromAccountID:  op.fromAccountID,
				ToAccountID:    op.toAccountID,
				Amount:         op.amount,
				Currency:       "USD",
				Description:    op.description,
				IdempotencyKey: op.idemKey,
			})
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}

	close(start)
	wg.Wait()

	return errs
}

func assertNoTransferErrors(t *testing.T, errs []error) {
	t.Helper()

	if len(errs) == 0 {
		return
	}

	for i := 0; i < len(errs) && i < 3; i++ {
		t.Logf("concurrent transfer error[%d]: %v", i, errs[i])
	}
	t.Fatalf("expected all concurrent transfers to succeed, got %d errors", len(errs))
}

func assertConcurrencyInvariants(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	accounts []concurrentAccountFixture,
	ops []concurrentTransferOperation,
	description string,
) {
	t.Helper()

	accountRepo := repository.NewAccountRepository(pool, logger)
	deltaByAccount := make(map[uuid.UUID]decimal.Decimal, len(accounts))
	for _, account := range accounts {
		deltaByAccount[account.accountID] = decimal.Zero
	}

	for _, op := range ops {
		amount := decimal.RequireFromString(op.amount)
		deltaByAccount[op.fromAccountID] = deltaByAccount[op.fromAccountID].Sub(amount)
		deltaByAccount[op.toAccountID] = deltaByAccount[op.toAccountID].Add(amount)
	}

	for _, account := range accounts {
		stored, err := accountRepo.GetByID(ctx, account.accountID)
		if err != nil {
			t.Fatalf("get account %s: %v", account.accountID, err)
		}

		expected := account.balance.Add(deltaByAccount[account.accountID])
		if !stored.Balance.Equal(expected) {
			t.Fatalf("unexpected balance for account %s: expected %s, got %s", account.accountID, expected, stored.Balance)
		}
		if stored.Balance.IsNegative() {
			t.Fatalf("negative balance for account %s: %s", account.accountID, stored.Balance)
		}
	}

	accountIDs := make([]uuid.UUID, 0, len(accounts))
	for _, account := range accounts {
		accountIDs = append(accountIDs, account.accountID)
	}

	var transferCount int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM transfers
		 WHERE description = $1
		   AND (from_account_id = ANY($2) OR to_account_id = ANY($2))`,
		description,
		accountIDs,
	).Scan(&transferCount); err != nil {
		t.Fatalf("count transfers for scenario %s: %v", description, err)
	}
	if transferCount != len(ops) {
		t.Fatalf("expected %d transfer rows, got %d", len(ops), transferCount)
	}

	var transactionCount int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM transactions
		 WHERE reference_id IN (
			SELECT id FROM transfers
			WHERE description = $1
			  AND (from_account_id = ANY($2) OR to_account_id = ANY($2))
		 )`,
		description,
		accountIDs,
	).Scan(&transactionCount); err != nil {
		t.Fatalf("count transactions for scenario %s: %v", description, err)
	}
	if transactionCount != transferCount*2 {
		t.Fatalf("expected %d transactions, got %d", transferCount*2, transactionCount)
	}
}

func seedTransferFixture(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	initialFrom decimal.Decimal,
	initialTo decimal.Decimal,
) (*transferIntegrationFixture, func()) {
	t.Helper()

	fixture := &transferIntegrationFixture{
		referenceKeySeed: uuid.NewString(),
		initialFrom:      initialFrom,
		initialTo:        initialTo,
	}

	userRepo := repository.NewUserRepository(pool, logger)
	accountRepo := repository.NewAccountRepository(pool, logger)

	fromUser := &domain.User{
		Email:        fmt.Sprintf("%s-owner@example.com", fixture.referenceKeySeed),
		PasswordHash: "hash",
		FullName:     "From Owner",
		Status:       domain.UserStatusActive,
	}
	if err := userRepo.Create(ctx, fromUser); err != nil {
		t.Fatalf("create from user: %v", err)
	}
	fixture.fromUserID = fromUser.ID

	toUser := &domain.User{
		Email:        fmt.Sprintf("%s-recipient@example.com", fixture.referenceKeySeed),
		PasswordHash: "hash",
		FullName:     "To Owner",
		Status:       domain.UserStatusActive,
	}
	if err := userRepo.Create(ctx, toUser); err != nil {
		t.Fatalf("create to user: %v", err)
	}
	fixture.toUserID = toUser.ID

	fromAccount := &domain.Account{
		UserID:        fixture.fromUserID,
		AccountNumber: uniqueAccountNumber(fixture.referenceKeySeed, "F"),
		AccountType:   domain.AccountTypeChecking,
		Balance:       initialFrom,
		Currency:      "USD",
		Status:        domain.AccountStatusActive,
	}
	if err := accountRepo.Create(ctx, fromAccount); err != nil {
		t.Fatalf("create from account: %v", err)
	}
	fixture.fromAccountID = fromAccount.ID

	toAccount := &domain.Account{
		UserID:        fixture.toUserID,
		AccountNumber: uniqueAccountNumber(fixture.referenceKeySeed, "T"),
		AccountType:   domain.AccountTypeChecking,
		Balance:       initialTo,
		Currency:      "USD",
		Status:        domain.AccountStatusActive,
	}
	if err := accountRepo.Create(ctx, toAccount); err != nil {
		t.Fatalf("create to account: %v", err)
	}
	fixture.toAccountID = toAccount.ID

	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM transactions WHERE account_id = ANY($1)`, []uuid.UUID{fixture.fromAccountID, fixture.toAccountID})
		_, _ = pool.Exec(ctx, `DELETE FROM transfers WHERE from_account_id = $1 OR to_account_id = $2`, fixture.fromAccountID, fixture.toAccountID)
		_, _ = pool.Exec(ctx, `DELETE FROM accounts WHERE id = ANY($1)`, []uuid.UUID{fixture.fromAccountID, fixture.toAccountID})
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = ANY($1)`, []uuid.UUID{fixture.fromUserID, fixture.toUserID})
	}

	return fixture, cleanup
}

func assertSingleBusinessEffect(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	fixture *transferIntegrationFixture,
	transfer *TransferResponse,
	expectedFrom decimal.Decimal,
	expectedTo decimal.Decimal,
	referenceKey string,
) {
	t.Helper()

	accountRepo := repository.NewAccountRepository(pool, logger)

	fromAccount, err := accountRepo.GetByID(ctx, fixture.fromAccountID)
	if err != nil {
		t.Fatalf("get from account: %v", err)
	}
	toAccount, err := accountRepo.GetByID(ctx, fixture.toAccountID)
	if err != nil {
		t.Fatalf("get to account: %v", err)
	}

	if !fromAccount.Balance.Equal(expectedFrom) {
		t.Fatalf("expected from balance %s, got %s", expectedFrom, fromAccount.Balance)
	}
	if !toAccount.Balance.Equal(expectedTo) {
		t.Fatalf("expected to balance %s, got %s", expectedTo, toAccount.Balance)
	}

	var transferCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM transfers WHERE reference_number = $1`, referenceKey).Scan(&transferCount); err != nil {
		t.Fatalf("count transfers by reference: %v", err)
	}
	if transferCount != 1 {
		t.Fatalf("expected exactly one transfer row, got %d", transferCount)
	}

	var txCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE reference_id = $1`, transfer.ID).Scan(&txCount); err != nil {
		t.Fatalf("count transactions by transfer reference: %v", err)
	}
	if txCount != 2 {
		t.Fatalf("expected exactly two transaction rows for transfer, got %d", txCount)
	}
}

func assertIdempotentTransferResult(t *testing.T, first, second *TransferResponse) {
	t.Helper()

	if first == nil || second == nil {
		t.Fatalf("expected non-nil transfer responses, got first=%v second=%v", first, second)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same transfer ID on replay, got %s and %s", first.ID, second.ID)
	}
	if first.ReferenceNumber != second.ReferenceNumber {
		t.Fatalf("expected same reference number on replay, got %s and %s", first.ReferenceNumber, second.ReferenceNumber)
	}
}

func integrationSchemaReady(ctx context.Context, pool *pgxpool.Pool) bool {
	var usersTable *string
	err := pool.QueryRow(ctx, `SELECT to_regclass('public.users')::text`).Scan(&usersTable)
	return err == nil && usersTable != nil && *usersTable == "users"
}

func uniqueAccountNumber(seed, suffix string) string {
	base := "A" + seed
	maxBaseLen := 20 - len(suffix)
	if maxBaseLen < 1 {
		maxBaseLen = 1
	}
	if len(base) > maxBaseLen {
		base = base[:maxBaseLen]
	}
	return base + suffix
}
