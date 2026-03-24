package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Kisotu/neobank/internal/domain"
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, code int, errCode, message string, details interface{}) {
	respondWithJSON(w, code, ErrorResponse{
		Error: ErrorDetail{
			Code:    errCode,
			Message: message,
			Details: details,
		},
	})
}

func handleDomainError(w http.ResponseWriter, err error) {
	var insuffFunds *domain.InsufficientFundsError
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	case errors.Is(err, domain.ErrForbidden):
		respondWithError(w, http.StatusForbidden, "FORBIDDEN", "forbidden", nil)
	case errors.Is(err, domain.ErrInvalidCredentials):
		respondWithError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password", nil)
	case errors.Is(err, domain.ErrDuplicateTransfer):
		respondWithError(w, http.StatusConflict, "DUPLICATE_TRANSFER", "duplicate transfer", nil)
	case errors.Is(err, domain.ErrDuplicateUser):
		respondWithError(w, http.StatusConflict, "DUPLICATE_USER", "user already exists", nil)
	case errors.Is(err, domain.ErrUserNotFound):
		respondWithError(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found", nil)
	case errors.Is(err, domain.ErrAccountNotFound):
		respondWithError(w, http.StatusNotFound, "ACCOUNT_NOT_FOUND", "account not found", nil)
	case errors.Is(err, domain.ErrTransactionNotFound):
		respondWithError(w, http.StatusNotFound, "TRANSACTION_NOT_FOUND", "transaction not found", nil)
	case errors.Is(err, domain.ErrAccountFrozen):
		respondWithError(w, http.StatusForbidden, "ACCOUNT_FROZEN", "account is frozen", nil)
	case errors.Is(err, domain.ErrInsufficientFunds):
		respondWithError(w, http.StatusBadRequest, "INSUFFICIENT_FUNDS", "insufficient funds", nil)
	case errors.As(err, &insuffFunds):
		respondWithError(w, http.StatusBadRequest, "INSUFFICIENT_FUNDS", insuffFunds.Error(), nil)
	case errors.Is(err, domain.ErrSameAccountTransfer):
		respondWithError(w, http.StatusBadRequest, "SAME_ACCOUNT_TRANSFER", "cannot transfer to same account", nil)
	case errors.Is(err, domain.ErrInvalidTransactionType):
		respondWithError(w, http.StatusBadRequest, "INVALID_TRANSACTION_TYPE", "invalid transaction type", nil)
	case errors.Is(err, domain.ErrInvalidTransfer), errors.Is(err, domain.ErrInvalidAmount), errors.Is(err, domain.ErrInvalidCurrency):
		respondWithError(w, http.StatusBadRequest, "INVALID_TRANSFER", err.Error(), nil)
	default:
		respondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}
