package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/username/banking-app/internal/auth"
	"github.com/username/banking-app/internal/handler/dto"
	"github.com/username/banking-app/internal/service"
)

type TransactionHandler struct {
	service        service.TransactionService
	accountService service.AccountService
}

func NewTransactionHandler(s service.TransactionService, accountService service.AccountService) *TransactionHandler {
	return &TransactionHandler{service: s, accountService: accountService}
}

func (h *TransactionHandler) ListByAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context", nil)
		return
	}

	accountID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_ACCOUNT_ID", "invalid account id", nil)
		return
	}

	account, err := h.accountService.GetAccount(r.Context(), accountID)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	if account.UserID != userID {
		respondWithError(w, http.StatusForbidden, "FORBIDDEN", "account does not belong to user", nil)
		return
	}

	limit := parseQueryInt(r, "limit", 50)
	offset := parseQueryInt(r, "offset", 0)

	items, err := h.service.ListByAccount(r.Context(), accountID, limit, offset)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	resp := make([]dto.TransactionResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, mapTransactionDTO(item))
	}

	respondWithJSON(w, http.StatusOK, resp)
}

func (h *TransactionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context", nil)
		return
	}

	transactionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_TRANSACTION_ID", "invalid transaction id", nil)
		return
	}

	item, err := h.service.GetByID(r.Context(), transactionID)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, mapTransactionDTO(item))
}

func parseQueryInt(r *http.Request, key string, fallback int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func mapTransactionDTO(item *service.TransactionResponse) dto.TransactionResponse {
	var referenceID *string
	if item.ReferenceID != nil {
		value := item.ReferenceID.String()
		referenceID = &value
	}

	return dto.TransactionResponse{
		ID:              item.ID.String(),
		AccountID:       item.AccountID.String(),
		TransactionType: item.TransactionType,
		Amount:          item.Amount,
		BalanceAfter:    item.BalanceAfter,
		ReferenceID:     referenceID,
		Description:     item.Description,
		CreatedAt:       item.CreatedAt,
	}
}
