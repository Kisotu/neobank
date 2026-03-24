package dto

import "time"

type TransactionResponse struct {
	ID              string    `json:"id"`
	AccountID       string    `json:"account_id"`
	TransactionType string    `json:"transaction_type"`
	Amount          string    `json:"amount"`
	BalanceAfter    string    `json:"balance_after"`
	ReferenceID     *string   `json:"reference_id,omitempty"`
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"created_at"`
}
