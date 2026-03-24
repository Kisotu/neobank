package repository

import (
	"time"

	"github.com/google/uuid"

	"github.com/username/banking-app/internal/db"
	"github.com/username/banking-app/internal/domain"
)

func toDomainUser(m *db.User) *domain.User {
	if m == nil {
		return nil
	}

	var deletedAt *time.Time
	if m.DeletedAt.Valid {
		value := m.DeletedAt.Time
		deletedAt = &value
	}

	return &domain.User{
		ID:           m.ID,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		FullName:     m.FullName,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		DeletedAt:    deletedAt,
	}
}

func toDomainAccount(m *db.Account) *domain.Account {
	if m == nil {
		return nil
	}

	return &domain.Account{
		ID:            m.ID,
		UserID:        m.UserID,
		AccountNumber: m.AccountNumber,
		AccountType:   domain.AccountType(m.AccountType),
		Balance:       m.Balance,
		Currency:      m.Currency,
		Status:        domain.AccountStatus(m.Status),
		Version:       int(m.Version),
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
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
		ID:              m.ID,
		AccountID:       m.AccountID,
		TransactionType: domain.TransactionType(m.TransactionType),
		Amount:          m.Amount,
		BalanceAfter:    m.BalanceAfter,
		ReferenceID:     ref,
		Description:     m.Description.String,
		CreatedAt:       m.CreatedAt,
	}
}

func toDomainTransfer(m *db.Transfer) *domain.Transfer {
	if m == nil {
		return nil
	}

	var completedAt *time.Time
	if m.CompletedAt.Valid {
		value := m.CompletedAt.Time
		completedAt = &value
	}

	return &domain.Transfer{
		ID:              m.ID,
		FromAccountID:   m.FromAccountID,
		ToAccountID:     m.ToAccountID,
		Amount:          m.Amount,
		Currency:        m.Currency,
		Status:          domain.TransferStatus(m.Status),
		ReferenceNumber: m.ReferenceNumber,
		Description:     m.Description.String,
		CreatedAt:       m.CreatedAt,
		CompletedAt:     completedAt,
	}
}