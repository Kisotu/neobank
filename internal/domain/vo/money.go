package vo

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type Money struct {
	Amount   decimal.Decimal
	Currency string
}

func NewMoney(amount decimal.Decimal, currency string) (Money, error) {
	if amount.IsNegative() {
		return Money{}, fmt.Errorf("amount cannot be negative")
	}

	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return Money{}, fmt.Errorf("currency is required")
	}

	return Money{Amount: amount, Currency: currency}, nil
}

func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: %s != %s", m.Currency, other.Currency)
	}
	return Money{Amount: m.Amount.Add(other.Amount), Currency: m.Currency}, nil
}

func (m Money) Subtract(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: %s != %s", m.Currency, other.Currency)
	}

	result := m.Amount.Sub(other.Amount)
	if result.IsNegative() {
		return Money{}, fmt.Errorf("resulting amount cannot be negative")
	}

	return Money{Amount: result, Currency: m.Currency}, nil
}

func (m Money) IsPositive() bool {
	return m.Amount.GreaterThan(decimal.Zero)
}