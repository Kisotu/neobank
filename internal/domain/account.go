package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AccountType string

const (
	AccountTypeChecking AccountType = "checking"
	AccountTypeSavings  AccountType = "savings"
)

type AccountStatus string

const (
	AccountStatusActive AccountStatus = "active"
	AccountStatusFrozen AccountStatus = "frozen"
	AccountStatusClosed AccountStatus = "closed"
)

type Account struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	AccountNumber string
	AccountType   AccountType
	Balance       decimal.Decimal
	Currency      string
	Status        AccountStatus
	Version       int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (a *Account) Validate() error {
	if a.UserID == uuid.Nil {
		return ErrInvalidAccountOwner
	}

	if !a.AccountType.IsValid() {
		return ErrInvalidAccountType
	}

	if !a.Status.IsValid() {
		return ErrInvalidAccountStatus
	}

	if strings.TrimSpace(a.Currency) == "" {
		return ErrInvalidCurrency
	}

	if a.Balance.IsNegative() {
		return ErrNegativeBalance
	}

	return nil
}

func (t AccountType) IsValid() bool {
	switch t {
	case AccountTypeChecking, AccountTypeSavings:
		return true
	default:
		return false
	}
}

func (s AccountStatus) IsValid() bool {
	switch s {
	case AccountStatusActive, AccountStatusFrozen, AccountStatusClosed:
		return true
	default:
		return false
	}
}

func (a *Account) IsOwner(userID uuid.UUID) bool {
	return a.UserID == userID
}

func (a *Account) CanDebit(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}

	if a.Status != AccountStatusActive {
		return ErrAccountFrozen
	}

	if a.Balance.LessThan(amount) {
		return &InsufficientFundsError{Available: a.Balance, Requested: amount}
	}

	return nil
}

func (a *Account) CanCredit(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidAmount
	}

	if a.Status != AccountStatusActive {
		return ErrAccountFrozen
	}

	return nil
}

func (a *Account) Debit(amount decimal.Decimal) error {
	if err := a.CanDebit(amount); err != nil {
		return err
	}

	a.Balance = a.Balance.Sub(amount)
	return nil
}

func (a *Account) Credit(amount decimal.Decimal) error {
	if err := a.CanCredit(amount); err != nil {
		return err
	}

	a.Balance = a.Balance.Add(amount)
	return nil
}
