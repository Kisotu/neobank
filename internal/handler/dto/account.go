package dto

import "time"

type CreateAccountRequest struct {
	AccountType string `json:"account_type" validate:"required,oneof=checking savings"`
	Currency    string `json:"currency" validate:"omitempty,len=3"`
}

type AccountResponse struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	AccountNumber string    `json:"account_number"`
	AccountType   string    `json:"account_type"`
	Balance       string    `json:"balance"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type BalanceResponse struct {
	AccountID string `json:"account_id"`
	Balance   string `json:"balance"`
	Currency  string `json:"currency"`
	Version   int    `json:"version"`
}
