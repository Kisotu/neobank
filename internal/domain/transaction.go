package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionType string

const (
	TransactionTypeDeposit     TransactionType = "deposit"
	TransactionTypeWithdrawal  TransactionType = "withdrawal"
	TransactionTypeTransferIn  TransactionType = "transfer_in"
	TransactionTypeTransferOut TransactionType = "transfer_out"
)

type Transaction struct {
	ID              uuid.UUID
	AccountID       uuid.UUID
	TransactionType TransactionType
	Amount          decimal.Decimal
	BalanceAfter    decimal.Decimal
	ReferenceID     *uuid.UUID
	Description     string
	CreatedAt       time.Time
}

func (t *Transaction) Validate() error {
	if t.AccountID == uuid.Nil {
		return ErrAccountNotFound
	}

	if !t.TransactionType.IsValid() {
		return ErrInvalidTransactionType
	}

	if t.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}

	if strings.TrimSpace(t.Description) == "" {
		return ErrInvalidTransactionDescription
	}

	return nil
}

func (t TransactionType) IsValid() bool {
	switch t {
	case TransactionTypeDeposit, TransactionTypeWithdrawal, TransactionTypeTransferIn, TransactionTypeTransferOut:
		return true
	default:
		return false
	}
}
