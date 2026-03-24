package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Kisotu/neobank/internal/db"
	"github.com/Kisotu/neobank/internal/domain"
)

type transferRepository struct {
	queries *db.Queries
	logger  *slog.Logger
}

func NewTransferRepository(dbtx db.DBTX, logger *slog.Logger) TransferRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &transferRepository{
		queries: db.New(dbtx),
		logger:  logger,
	}
}

func (r *transferRepository) Create(ctx context.Context, transfer *domain.Transfer) error {
	if err := transfer.Validate(); err != nil {
		return err
	}

	created, err := r.queries.CreateTransfer(ctx, &db.CreateTransferParams{
		FromAccountID:   toPgUUID(transfer.FromAccountID),
		ToAccountID:     toPgUUID(transfer.ToAccountID),
		Amount:          transfer.Amount,
		Currency:        transfer.Currency,
		Status:          string(transfer.Status),
		ReferenceNumber: transfer.ReferenceNumber,
		Description:     toNullableText(transfer.Description),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDuplicateTransfer
		}
		return fmt.Errorf("create transfer: %w", err)
	}

	*transfer = *toDomainTransfer(created)
	r.logger.InfoContext(ctx, "transfer created", "transfer_id", transfer.ID.String(), "reference", transfer.ReferenceNumber)
	return nil
}

func (r *transferRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transfer, error) {
	row, err := r.queries.GetTransferByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidTransfer
		}
		return nil, fmt.Errorf("get transfer by id: %w", err)
	}

	return toDomainTransfer(row), nil
}

func (r *transferRepository) GetByReference(ctx context.Context, ref string) (*domain.Transfer, error) {
	row, err := r.queries.GetTransferByReference(ctx, ref)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidTransfer
		}
		return nil, fmt.Errorf("get transfer by reference: %w", err)
	}

	return toDomainTransfer(row), nil
}

func (r *transferRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TransferStatus) error {
	err := r.queries.UpdateTransferStatus(ctx, &db.UpdateTransferStatusParams{
		ID:     toPgUUID(id),
		Status: string(status),
	})
	if err != nil {
		return fmt.Errorf("update transfer status: %w", err)
	}

	r.logger.InfoContext(ctx, "transfer status updated", "transfer_id", id.String(), "status", string(status))
	return nil
}

func (r *transferRepository) ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*domain.Transfer, error) {
	rows, err := r.queries.ListTransfersByAccount(ctx, &db.ListTransfersByAccountParams{
		FromAccountID: toPgUUID(accountID),
		Limit:         int32(limit),
		Offset:        int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list transfers by account: %w", err)
	}

	transfers := make([]*domain.Transfer, 0, len(rows))
	for _, row := range rows {
		transfers = append(transfers, toDomainTransfer(row))
	}

	return transfers, nil
}
