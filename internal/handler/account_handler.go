package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/username/banking-app/internal/auth"
	"github.com/username/banking-app/internal/handler/dto"
	"github.com/username/banking-app/internal/service"
)

type AccountHandler struct {
	service   service.AccountService
	validator *validator.Validate
}

func NewAccountHandler(s service.AccountService) *AccountHandler {
	return &AccountHandler{service: s, validator: validator.New()}
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context", nil)
		return
	}

	var req dto.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respondWithError(w, http.StatusBadRequest, "VALIDATION_FAILED", "request validation failed", err.Error())
		return
	}

	result, err := h.service.CreateAccount(r.Context(), userID, &service.CreateAccountRequest{
		AccountType: req.AccountType,
		Currency:    req.Currency,
	})
	if err != nil {
		handleDomainError(w, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, mapAccountResponse(result))
}

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context", nil)
		return
	}

	items, err := h.service.ListAccounts(r.Context(), userID)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	resp := make([]dto.AccountResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, mapAccountResponse(item))
	}

	respondWithJSON(w, http.StatusOK, resp)
}

func (h *AccountHandler) GetByID(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.service.GetAccount(r.Context(), accountID)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	if result.UserID != userID {
		respondWithError(w, http.StatusForbidden, "FORBIDDEN", "account does not belong to user", nil)
		return
	}

	respondWithJSON(w, http.StatusOK, mapAccountResponse(result))
}

func (h *AccountHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
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

	account, err := h.service.GetAccount(r.Context(), accountID)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	if account.UserID != userID {
		respondWithError(w, http.StatusForbidden, "FORBIDDEN", "account does not belong to user", nil)
		return
	}

	balance, err := h.service.GetBalance(r.Context(), accountID)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dto.BalanceResponse{
		AccountID: balance.AccountID.String(),
		Balance:   balance.Balance,
		Currency:  balance.Currency,
		Version:   balance.Version,
	})
}

func mapAccountResponse(s *service.AccountResponse) dto.AccountResponse {
	return dto.AccountResponse{
		ID:            s.ID.String(),
		UserID:        s.UserID.String(),
		AccountNumber: s.AccountNumber,
		AccountType:   s.AccountType,
		Balance:       s.Balance,
		Currency:      s.Currency,
		Status:        s.Status,
		Version:       s.Version,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}
