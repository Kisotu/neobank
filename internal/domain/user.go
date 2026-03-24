package domain

import (
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBlocked  UserStatus = "blocked"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FullName     string
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

func (u *User) Validate() error {
	if _, err := mail.ParseAddress(u.Email); err != nil {
		return ErrInvalidUserEmail
	}

	if strings.TrimSpace(u.FullName) == "" {
		return ErrInvalidUserName
	}

	if u.Status == "" {
		u.Status = UserStatusActive
	}

	if !u.Status.IsValid() {
		return ErrInvalidUserStatus
	}

	return nil
}

func (s UserStatus) IsValid() bool {
	switch s {
	case UserStatusActive, UserStatusInactive, UserStatusBlocked:
		return true
	default:
		return false
	}
}

func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

func (u *User) MarkDeleted(now time.Time) error {
	if u.DeletedAt != nil {
		return errors.New("user already deleted")
	}
	u.DeletedAt = &now
	return nil
}
