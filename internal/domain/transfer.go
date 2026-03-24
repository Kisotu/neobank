package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransferStatus string

const (
	TransferStatusPending   TransferStatus = "pending"
	TransferStatusCompleted TransferStatus = "completed"
	TransferStatusFailed    TransferStatus = "failed"
	TransferStatusReversed  TransferStatus = "reversed"
)

type Transfer struct {
	ID              uuid.UUID
	FromAccountID   uuid.UUID
	ToAccountID     uuid.UUID
	Amount          decimal.Decimal
	Currency        string
	Status          TransferStatus
	ReferenceNumber string
	Description     string
	CreatedAt       time.Time
	CompletedAt     *time.Time
}

func NewTransfer(fromAccountID, toAccountID uuid.UUID, amount decimal.Decimal, currency, referenceNumber, description string) (*Transfer, error) {
	if fromAccountID == toAccountID {
		return nil, ErrSameAccountTransfer
	}

	transfer := &Transfer{
		FromAccountID:   fromAccountID,
		ToAccountID:     toAccountID,
		Amount:          amount,
		Currency:        currency,
		Status:          TransferStatusPending,
		ReferenceNumber: strings.TrimSpace(referenceNumber),
		Description:     strings.TrimSpace(description),
	}

	if err := transfer.Validate(); err != nil {
		return nil, err
	}

	return transfer, nil
}

func (t *Transfer) Validate() error {
	if t.FromAccountID == uuid.Nil || t.ToAccountID == uuid.Nil {
		return ErrInvalidTransfer
	}

	if t.FromAccountID == t.ToAccountID {
		return ErrSameAccountTransfer
	}

	if t.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}

	if strings.TrimSpace(t.Currency) == "" {
		return ErrInvalidCurrency
	}

	if strings.TrimSpace(t.ReferenceNumber) == "" {
		return ErrInvalidReference
	}

	if !t.Status.IsValid() {
		return ErrInvalidTransferStatus
	}

	return nil
}

func (s TransferStatus) IsValid() bool {
	switch s {
	case TransferStatusPending, TransferStatusCompleted, TransferStatusFailed, TransferStatusReversed:
		return true
	default:
		return false
	}
}

func (t *Transfer) Complete(at time.Time) error {
	if t.Status != TransferStatusPending {
		return ErrInvalidTransferTransition
	}
	t.Status = TransferStatusCompleted
	t.CompletedAt = &at
	return nil
}

func (t *Transfer) Fail() error {
	if t.Status != TransferStatusPending {
		return ErrInvalidTransferTransition
	}
	t.Status = TransferStatusFailed
	return nil
}

func (t *Transfer) Reverse() error {
	if t.Status != TransferStatusCompleted {
		return ErrInvalidTransferTransition
	}
	t.Status = TransferStatusReversed
	return nil
}