package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Kisotu/neobank/internal/domain"
	"github.com/Kisotu/neobank/internal/domain/vo"
	"github.com/Kisotu/neobank/internal/repository"
)

type transferTxDeps struct {
	accountRepo     repository.AccountRepository
	transferRepo    repository.TransferRepository
	transactionRepo repository.TransactionRepository
}

type transferRunInTx func(ctx context.Context, fn func(deps transferTxDeps) error) error

type transferService struct {
	accountRepo  repository.AccountRepository
	transferRepo repository.TransferRepository
	runInTx      transferRunInTx
	logger       *slog.Logger
}

const (
	transferMaxRetries = 3
	retryBackoffStep   = 50 * time.Millisecond
)

func NewTransferService(db *pgxpool.Pool, logger *slog.Logger) TransferService {
	if logger == nil {
		logger = slog.Default()
	}

	if db == nil {
		return &transferService{logger: logger}
	}

	baseAccountRepo := repository.NewAccountRepository(db, logger)
	baseTransferRepo := repository.NewTransferRepository(db, logger)

	runInTx := func(ctx context.Context, fn func(deps transferTxDeps) error) error {
		tx, err := db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}

		defer func() {
			if p := recover(); p != nil {
				_ = tx.Rollback(ctx)
				panic(p)
			}
		}()

		deps := transferTxDeps{
			accountRepo:     repository.NewAccountRepository(tx, logger),
			transferRepo:    repository.NewTransferRepository(tx, logger),
			transactionRepo: repository.NewTransactionRepository(tx, logger),
		}

		if err := fn(deps); err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				return fmt.Errorf("tx error: %v, rollback error: %w", err, rbErr)
			}
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit tx: %w", err)
		}

		return nil
	}

	return newTransferServiceWithDependencies(baseAccountRepo, baseTransferRepo, runInTx, logger)
}

func newTransferServiceWithDependencies(
	accountRepo repository.AccountRepository,
	transferRepo repository.TransferRepository,
	runInTx transferRunInTx,
	logger *slog.Logger,
) *transferService {
	if logger == nil {
		logger = slog.Default()
	}

	return &transferService{
		accountRepo:  accountRepo,
		transferRepo: transferRepo,
		runInTx:      runInTx,
		logger:       logger,
	}
}

func (s *transferService) Transfer(ctx context.Context, userID uuid.UUID, req *TransferRequest) (*TransferResponse, error) {
	if s.transferRepo == nil || s.runInTx == nil {
		return nil, fmt.Errorf("transfer service dependencies are not configured")
	}

	if req == nil {
		return nil, domain.ErrInvalidTransfer
	}

	if req.FromAccountID == req.ToAccountID {
		return nil, domain.ErrSameAccountTransfer
	}

	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidAmount
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "USD"
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.transferRepo.GetByReference(ctx, idempotencyKey)
		if err == nil {
			return toTransferResponse(existing), nil
		}
		if !errors.Is(err, domain.ErrInvalidTransfer) {
			return nil, err
		}
	}

	var transfer *domain.Transfer
	for attempt := 1; attempt <= transferMaxRetries; attempt++ {
		transfer = nil
		err = s.runInTx(ctx, func(deps transferTxDeps) error {
			if deps.accountRepo == nil || deps.transferRepo == nil || deps.transactionRepo == nil {
				return fmt.Errorf("transfer transactional dependencies are nil")
			}

			accountRepo := deps.accountRepo
			transferRepo := deps.transferRepo
			transactionRepo := deps.transactionRepo

			accountIDs := []uuid.UUID{req.FromAccountID, req.ToAccountID}
			sort.Slice(accountIDs, func(i, j int) bool {
				return accountIDs[i].String() < accountIDs[j].String()
			})

			accounts, err := accountRepo.LockForUpdate(ctx, accountIDs...)
			if err != nil {
				return err
			}

			fromAccount, toAccount := findAccounts(accounts, req.FromAccountID, req.ToAccountID)
			if fromAccount == nil || toAccount == nil {
				return domain.ErrAccountNotFound
			}
			if !fromAccount.IsOwner(userID) {
				return domain.ErrForbidden
			}

			if !strings.EqualFold(fromAccount.Currency, currency) || !strings.EqualFold(toAccount.Currency, currency) {
				return domain.ErrInvalidCurrency
			}

			if err := fromAccount.Debit(amount); err != nil {
				return err
			}

			if err := toAccount.Credit(amount); err != nil {
				return err
			}

			refValue := idempotencyKey
			if refValue == "" {
				ref, err := vo.GenerateReferenceNumber("TRX")
				if err != nil {
					return fmt.Errorf("generate reference number: %w", err)
				}
				refValue = ref.String()
			}

			transfer, err = domain.NewTransfer(
				req.FromAccountID,
				req.ToAccountID,
				amount,
				currency,
				refValue,
				req.Description,
			)
			if err != nil {
				return err
			}

			if err := transferRepo.Create(ctx, transfer); err != nil {
				if errors.Is(err, domain.ErrDuplicateTransfer) && idempotencyKey != "" {
					existing, getErr := transferRepo.GetByReference(ctx, idempotencyKey)
					if getErr != nil {
						return getErr
					}
					transfer = existing
					return nil
				}
				return err
			}

			if err := accountRepo.UpdateBalance(ctx, fromAccount.ID, fromAccount.Balance, fromAccount.Version); err != nil {
				return err
			}

			if err := accountRepo.UpdateBalance(ctx, toAccount.ID, toAccount.Balance, toAccount.Version); err != nil {
				return err
			}

			referenceID := transfer.ID
			outTx := &domain.Transaction{
				AccountID:       fromAccount.ID,
				TransactionType: domain.TransactionTypeTransferOut,
				Amount:          amount,
				BalanceAfter:    fromAccount.Balance,
				ReferenceID:     &referenceID,
				Description:     strings.TrimSpace(req.Description),
			}
			if outTx.Description == "" {
				outTx.Description = "transfer out"
			}

			if err := transactionRepo.Create(ctx, outTx); err != nil {
				return err
			}

			inTx := &domain.Transaction{
				AccountID:       toAccount.ID,
				TransactionType: domain.TransactionTypeTransferIn,
				Amount:          amount,
				BalanceAfter:    toAccount.Balance,
				ReferenceID:     &referenceID,
				Description:     strings.TrimSpace(req.Description),
			}
			if inTx.Description == "" {
				inTx.Description = "transfer in"
			}

			if err := transactionRepo.Create(ctx, inTx); err != nil {
				return err
			}

			if err := transfer.Complete(nowUTC()); err != nil {
				return err
			}

			if err := transferRepo.UpdateStatus(ctx, transfer.ID, transfer.Status); err != nil {
				return err
			}

			return nil
		})
		if err == nil {
			break
		}
		if !isRetryableTransferError(err) || attempt == transferMaxRetries {
			break
		}
		time.Sleep(time.Duration(attempt) * retryBackoffStep)
	}

	if err != nil {
		s.logger.ErrorContext(ctx, "transfer failed", "from_account_id", req.FromAccountID.String(), "to_account_id", req.ToAccountID.String(), "error", err.Error())
		return nil, err
	}

	resp := toTransferResponse(transfer)
	s.logger.InfoContext(ctx, "transfer completed", "transfer_id", transfer.ID.String(), "reference", transfer.ReferenceNumber)
	return resp, nil
}

func (s *transferService) GetTransfer(ctx context.Context, userID, transferID uuid.UUID) (*TransferResponse, error) {
	if s.transferRepo == nil || s.accountRepo == nil {
		return nil, fmt.Errorf("transfer service dependencies are not configured")
	}

	transfer, err := s.transferRepo.GetByID(ctx, transferID)
	if err != nil {
		return nil, err
	}
	fromAccount, err := s.accountRepo.GetByID(ctx, transfer.FromAccountID)
	if err != nil {
		return nil, err
	}
	toAccount, err := s.accountRepo.GetByID(ctx, transfer.ToAccountID)
	if err != nil {
		return nil, err
	}
	if !fromAccount.IsOwner(userID) && !toAccount.IsOwner(userID) {
		return nil, domain.ErrForbidden
	}

	return toTransferResponse(transfer), nil
}

func (s *transferService) ListTransfers(ctx context.Context, userID, accountID uuid.UUID, limit, offset int) ([]*TransferResponse, error) {
	if s.transferRepo == nil || s.accountRepo == nil {
		return nil, fmt.Errorf("transfer service dependencies are not configured")
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if !account.IsOwner(userID) {
		return nil, domain.ErrForbidden
	}

	transfers, err := s.transferRepo.ListByAccount(ctx, accountID, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]*TransferResponse, 0, len(transfers))
	for _, transfer := range transfers {
		responses = append(responses, toTransferResponse(transfer))
	}

	return responses, nil
}
func findAccounts(accounts []*domain.Account, fromID, toID uuid.UUID) (from *domain.Account, to *domain.Account) {
	for _, account := range accounts {
		if account.ID == fromID {
			from = account
		}
		if account.ID == toID {
			to = account
		}
	}
	return from, to
}

func toTransferResponse(transfer *domain.Transfer) *TransferResponse {
	if transfer == nil {
		return nil
	}

	return &TransferResponse{
		ID:              transfer.ID,
		FromAccountID:   transfer.FromAccountID,
		ToAccountID:     transfer.ToAccountID,
		Amount:          transfer.Amount.String(),
		Currency:        transfer.Currency,
		Status:          string(transfer.Status),
		ReferenceNumber: transfer.ReferenceNumber,
		Description:     transfer.Description,
		CreatedAt:       transfer.CreatedAt,
		CompletedAt:     transfer.CompletedAt,
	}
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

func isRetryableTransferError(err error) bool {
	if errors.Is(err, domain.ErrOptimisticLock) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}
	return false
}
