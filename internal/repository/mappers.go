package repository

import (
	"github.com/google/uuid"

	"github.com/Kisotu/neobank/internal/db"
	"github.com/Kisotu/neobank/internal/domain"
)

func toDomainUser(m *db.User) *domain.User {
	if m == nil {
		return nil
	}

	return &domain.User{
		ID:           fromPgUUID(m.ID),
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		FullName:     m.FullName,
		Status:       domain.UserStatusActive,
		CreatedAt:    fromPgTime(m.CreatedAt),
		UpdatedAt:    fromPgTime(m.UpdatedAt),
		DeletedAt:    fromNullablePgTime(m.DeletedAt),
	}
}

func toDomainAccount(m *db.Account) *domain.Account {
	if m == nil {
		return nil
	}

	return &domain.Account{
		ID:            fromPgUUID(m.ID),
		UserID:        fromPgUUID(m.UserID),
		AccountNumber: m.AccountNumber,
		AccountType:   domain.AccountType(m.AccountType),
		Balance:       m.Balance,
		Currency:      m.Currency,
		Status:        domain.AccountStatus(m.Status),
		Version:       int(m.Version),
		CreatedAt:     fromPgTime(m.CreatedAt),
		UpdatedAt:     fromPgTime(m.UpdatedAt),
	}
}

func toDomainTransaction(m *db.Transaction) *domain.Transaction {
	if m == nil {
		return nil
	}

	var ref *uuid.UUID
	if m.ReferenceID.Valid {
		if parsed, err := uuid.FromBytes(m.ReferenceID.Bytes[:]); err == nil {
			ref = &parsed
		}
	}

	return &domain.Transaction{
		ID:              fromPgUUID(m.ID),
		AccountID:       fromPgUUID(m.AccountID),
		TransactionType: domain.TransactionType(m.TransactionType),
		Amount:          m.Amount,
		BalanceAfter:    m.BalanceAfter,
		ReferenceID:     ref,
		Description:     m.Description.String,
		CreatedAt:       fromPgTime(m.CreatedAt),
	}
}

func toDomainTransfer(m *db.Transfer) *domain.Transfer {
	if m == nil {
		return nil
	}

	return &domain.Transfer{
		ID:              fromPgUUID(m.ID),
		FromAccountID:   fromPgUUID(m.FromAccountID),
		ToAccountID:     fromPgUUID(m.ToAccountID),
		Amount:          m.Amount,
		Currency:        m.Currency,
		Status:          domain.TransferStatus(m.Status),
		ReferenceNumber: m.ReferenceNumber,
		Description:     m.Description.String,
		CreatedAt:       fromPgTime(m.CreatedAt),
		CompletedAt:     fromNullablePgTime(m.CompletedAt),
	}
}
