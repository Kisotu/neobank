package vo

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var referenceNumberPattern = regexp.MustCompile(`^[A-Z0-9\-]{8,50}$`)

type ReferenceNumber string

func NewReferenceNumber(value string) (ReferenceNumber, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if !referenceNumberPattern.MatchString(value) {
		return "", fmt.Errorf("invalid reference number format")
	}
	return ReferenceNumber(value), nil
}

func GenerateReferenceNumber(prefix string) (ReferenceNumber, error) {
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	if prefix == "" {
		prefix = "TRX"
	}

	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate reference number entropy: %w", err)
	}

	value := fmt.Sprintf("%s-%s-%s", prefix, time.Now().UTC().Format("20060102150405"), strings.ToUpper(hex.EncodeToString(buf)))
	return NewReferenceNumber(value)
}

func (r ReferenceNumber) String() string {
	return string(r)
}