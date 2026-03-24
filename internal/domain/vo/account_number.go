package vo

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var accountNumberPattern = regexp.MustCompile(`^[0-9]{12,20}$`)

type AccountNumber string

func NewAccountNumber(value string) (AccountNumber, error) {
	value = strings.TrimSpace(value)
	if !accountNumberPattern.MatchString(value) {
		return "", fmt.Errorf("invalid account number format")
	}
	return AccountNumber(value), nil
}

func GenerateAccountNumber() (AccountNumber, error) {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("failed to generate account number entropy: %w", err)
	}

	nowPrefix := time.Now().UTC().Format("060102150405")
	randomPart := fmt.Sprintf("%03d", int(b[0])%1000)
	value := nowPrefix + randomPart

	return NewAccountNumber(value)
}

func (a AccountNumber) String() string {
	return string(a)
}
