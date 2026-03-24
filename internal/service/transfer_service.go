package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/username/banking-app/internal/domain"
	"github.com/username/banking-app/internal/domain/vo"
	"github.com/username/banking-app/internal/repository"
)

type transferService struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewTransferService(db *pgxpool.Pool, logger *slog.Logger) TransferService {
	if logger == nil {
		logger = slog.Default()
	}

	return &transferService{
		db:     db,
		logger: logger,
	}
}

func (s *transferService) Transfer(ctx context.Context, req *TransferRequest) (*TransferResponse, error) {
	if s.db == nil {
		return nil, fmt.Errorf("transfer service db is nil")
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

	var transfer *domain.Transfer
	err = s.execInTx(ctx, func(tx pgx.Tx) error {
		accountRepo := repository.NewAccountRepository(tx, s.logger)
		transferRepo := repository.NewTransferRepository(tx, s.logger)
		transactionRepo := repository.NewTransactionRepository(tx, s.logger)

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

		if !strings.EqualFold(fromAccount.Currency, currency) || !strings.EqualFold(toAccount.Currency, currency) {
			return domain.ErrInvalidCurrency
		}

		if err := fromAccount.Debit(amount); err != nil {
			return err
		}

		if err := toAccount.Credit(amount); err != nil {
			return err
		}

		ref, err := vo.GenerateReferenceNumber("TRX")
		if err != nil {
			return fmt.Errorf("generate reference number: %w", err)
		}

		transfer, err = domain.NewTransfer(
			req.FromAccountID,
			req.ToAccountID,
			amount,
			currency,
			ref.String(),
			req.Description,
		)
		if err != nil {
			return err
		}

		if err := transferRepo.Create(ctx, transfer); err != nil {
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
	if err != nil {
		s.logger.ErrorContext(ctx, "transfer failed", "from_account_id", req.FromAccountID.String(), "to_account_id", req.ToAccountID.String(), "error", err.Error())
		return nil, err
	}

	resp := toTransferResponse(transfer)
	s.logger.InfoContext(ctx, "transfer completed", "transfer_id", transfer.ID.String(), "reference", transfer.ReferenceNumber)
	return resp, nil
}

func (s *transferService) GetTransfer(ctx context.Context, transferID uuid.UUID) (*TransferResponse, error) {
	if s.db == nil {
		return nil, fmt.Errorf("transfer service db is nil")
	}

	transferRepo := repository.NewTransferRepository(s.db, s.logger)
	transfer, err := transferRepo.GetByID(ctx, transferID)
	if err != nil {
		return nil, err
	}

	return toTransferResponse(transfer), nil
}

func (s *transferService) ListTransfers(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*TransferResponse, error) {
	if s.db == nil {
		return nil, fmt.Errorf("transfer service db is nil")
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	transferRepo := repository.NewTransferRepository(s.db, s.logger)
	transfers, err := transferRepo.ListByAccount(ctx, accountID, limit, offset)
	if err != nil {
		return nil, err
	}

	responses := make([]*TransferResponse, 0, len(transfers))
	for _, transfer := range transfers {
		responses = append(responses, toTransferResponse(transfer))
	}

	return responses, nil
}

func (s *transferService) execInTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
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
