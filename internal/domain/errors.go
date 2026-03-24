package domain

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

var (
	ErrAccountNotFound               = errors.New("account not found")
	ErrTransactionNotFound           = errors.New("transaction not found")
	ErrUserNotFound                  = errors.New("user not found")
	ErrDuplicateUser                 = errors.New("user already exists")
	ErrInvalidCredentials            = errors.New("invalid credentials")
	ErrUnauthorized                  = errors.New("unauthorized")
	ErrForbidden                     = errors.New("forbidden")
	ErrInsufficientFunds             = errors.New("insufficient funds")
	ErrAccountFrozen                 = errors.New("account is frozen")
	ErrInvalidTransfer               = errors.New("invalid transfer")
	ErrSameAccountTransfer           = errors.New("cannot transfer to same account")
	ErrDuplicateTransfer             = errors.New("duplicate transfer")
	ErrOptimisticLock                = errors.New("optimistic lock conflict")
	ErrInvalidAmount                 = errors.New("invalid amount")
	ErrInvalidCurrency               = errors.New("invalid currency")
	ErrInvalidReference              = errors.New("invalid reference number")
	ErrInvalidTransferStatus         = errors.New("invalid transfer status")
	ErrInvalidTransferTransition     = errors.New("invalid transfer status transition")
	ErrInvalidTransactionType        = errors.New("invalid transaction type")
	ErrInvalidTransactionDescription = errors.New("invalid transaction description")
	ErrInvalidAccountType            = errors.New("invalid account type")
	ErrInvalidAccountStatus          = errors.New("invalid account status")
	ErrInvalidAccountOwner           = errors.New("invalid account owner")
	ErrNegativeBalance               = errors.New("balance cannot be negative")
	ErrInvalidUserEmail              = errors.New("invalid user email")
	ErrInvalidUserName               = errors.New("invalid user name")
	ErrInvalidUserStatus             = errors.New("invalid user status")
	ErrInvalidAccountNumber          = errors.New("invalid account number")
)

type InsufficientFundsError struct {
	Available decimal.Decimal
	Requested decimal.Decimal
}

func (e *InsufficientFundsError) Error() string {
	return fmt.Sprintf("insufficient funds: available %s, requested %s", e.Available.String(), e.Requested.String())
}

func (e *InsufficientFundsError) Unwrap() error {
	return ErrInsufficientFunds
}

type AccountNotFoundError struct {
	ID string
}

func (e *AccountNotFoundError) Error() string {
	return fmt.Sprintf("account not found: %s", e.ID)
}

func (e *AccountNotFoundError) Unwrap() error {
	return ErrAccountNotFound
}

type InvalidTransferError struct {
	Reason string
}

func (e *InvalidTransferError) Error() string {
	if e.Reason == "" {
		return ErrInvalidTransfer.Error()
	}
	return fmt.Sprintf("%s: %s", ErrInvalidTransfer.Error(), e.Reason)
}

func (e *InvalidTransferError) Unwrap() error {
	return ErrInvalidTransfer
}

type AccountFrozenError struct {
	AccountID string
}

func (e *AccountFrozenError) Error() string {
	if e.AccountID == "" {
		return ErrAccountFrozen.Error()
	}
	return fmt.Sprintf("account is frozen: %s", e.AccountID)
}

func (e *AccountFrozenError) Unwrap() error {
	return ErrAccountFrozen
}

type DuplicateTransferError struct {
	Reference string
}

func (e *DuplicateTransferError) Error() string {
	if e.Reference == "" {
		return ErrDuplicateTransfer.Error()
	}
	return fmt.Sprintf("duplicate transfer reference: %s", e.Reference)
}

func (e *DuplicateTransferError) Unwrap() error {
	return ErrDuplicateTransfer
}

func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}
