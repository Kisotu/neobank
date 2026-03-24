package dto

import "time"

type CreateTransferRequest struct {
	FromAccountID   string `json:"from_account_id" validate:"required,uuid4"`
	ToAccountID     string `json:"to_account_id" validate:"required,uuid4,nefield=FromAccountID"`
	Amount          string `json:"amount" validate:"required"`
	Currency        string `json:"currency" validate:"required,len=3"`
	ReferenceNumber string `json:"reference_number" validate:"omitempty,max=50"`
	Description     string `json:"description" validate:"max=500"`
}

type TransferResponse struct {
	ID              string     `json:"id"`
	FromAccountID   string     `json:"from_account_id"`
	ToAccountID     string     `json:"to_account_id"`
	Amount          string     `json:"amount"`
	Currency        string     `json:"currency"`
	Status          string     `json:"status"`
	ReferenceNumber string     `json:"reference_number"`
	Description     string     `json:"description,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}
