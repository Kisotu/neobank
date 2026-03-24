package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/username/banking-app/internal/auth"
	"github.com/username/banking-app/internal/handler/dto"
	"github.com/username/banking-app/internal/service"
)

type TransactionHandler struct {
	service service.TransactionService
}

func NewTransactionHandler(s service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: s}
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

	filter, err := parseTransactionFilters(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_FILTER", err.Error(), nil)
		return
	}

	items, err := h.service.ListByAccount(r.Context(), userID, accountID, filter)
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
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context", nil)
		return
	}

	transactionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_TRANSACTION_ID", "invalid transaction id", nil)
		return
	}

	item, err := h.service.GetByID(r.Context(), userID, transactionID)
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

func parseTransactionFilters(r *http.Request) (*service.TransactionListFilter, error) {
	filter := &service.TransactionListFilter{
		Limit:  parseQueryInt(r, "limit", 50),
		Offset: parseQueryInt(r, "offset", 0),
	}

	if rawType := strings.TrimSpace(r.URL.Query().Get("type")); rawType != "" {
		filter.TransactionType = rawType
	}

	if rawFrom := strings.TrimSpace(r.URL.Query().Get("from")); rawFrom != "" {
		parsed := parseDate(rawFrom)
		if parsed == nil {
			return nil, errors.New("invalid 'from' date format, expected RFC3339 or YYYY-MM-DD")
		}
		filter.StartDate = parsed
	}

	if rawTo := strings.TrimSpace(r.URL.Query().Get("to")); rawTo != "" {
		parsed := parseDate(rawTo)
		if parsed == nil {
			return nil, errors.New("invalid 'to' date format, expected RFC3339 or YYYY-MM-DD")
		}
		filter.EndDate = parsed
	}

	return filter, nil
}

func parseDate(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		utc := ts.UTC()
		return &utc
	}
	if d, err := time.Parse("2006-01-02", value); err == nil {
		utc := d.UTC()
		return &utc
	}
	return nil
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
