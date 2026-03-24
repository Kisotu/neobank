package events

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Event interface {
	EventName() string
	OccurredAt() time.Time
}

type Dispatcher interface {
	Dispatch(ctx context.Context, event Event) error
}

type TransferInitiatedEvent struct {
	TransferID    uuid.UUID
	FromAccountID uuid.UUID
	ToAccountID   uuid.UUID
	Amount        decimal.Decimal
	Currency      string
	Reference     string
	When          time.Time
}

func (e TransferInitiatedEvent) EventName() string     { return "transfer.initiated" }
func (e TransferInitiatedEvent) OccurredAt() time.Time { return e.When }

type TransferCompletedEvent struct {
	TransferID    uuid.UUID
	FromAccountID uuid.UUID
	ToAccountID   uuid.UUID
	Amount        decimal.Decimal
	Currency      string
	Reference     string
	When          time.Time
}

func (e TransferCompletedEvent) EventName() string     { return "transfer.completed" }
func (e TransferCompletedEvent) OccurredAt() time.Time { return e.When }

type TransferFailedEvent struct {
	TransferID    uuid.UUID
	FromAccountID uuid.UUID
	ToAccountID   uuid.UUID
	Reason        string
	When          time.Time
}

func (e TransferFailedEvent) EventName() string     { return "transfer.failed" }
func (e TransferFailedEvent) OccurredAt() time.Time { return e.When }

type AccountBalanceChangedEvent struct {
	AccountID uuid.UUID
	Previous  decimal.Decimal
	Current   decimal.Decimal
	Currency  string
	When      time.Time
}

func (e AccountBalanceChangedEvent) EventName() string     { return "account.balance_changed" }
func (e AccountBalanceChangedEvent) OccurredAt() time.Time { return e.When }
