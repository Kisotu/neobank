package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/Kisotu/neobank/internal/auth"
	"github.com/Kisotu/neobank/internal/handler/dto"
	"github.com/Kisotu/neobank/internal/service"
)

const (
	defaultTransferListLimit = 20
	maxTransferListLimit     = 200
)

type TransferHandler struct {
	service   service.TransferService
	validator *validator.Validate
}

func NewTransferHandler(s service.TransferService) *TransferHandler {
	return &TransferHandler{
		service:   s,
		validator: validator.New(),
	}
}

func (h *TransferHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context", nil)
		return
	}

	var req dto.CreateTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respondWithError(w, http.StatusBadRequest, "VALIDATION_FAILED", "request validation failed", err.Error())
		return
	}

	fromID, err := uuid.Parse(req.FromAccountID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_FROM_ACCOUNT_ID", "invalid from account id", nil)
		return
	}

	toID, err := uuid.Parse(req.ToAccountID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_TO_ACCOUNT_ID", "invalid to account id", nil)
		return
	}

	result, err := h.service.Transfer(r.Context(), userID, &service.TransferRequest{
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Description:    req.Description,
		IdempotencyKey: pickIdempotencyKey(req.ReferenceNumber, r.Header.Get("Idempotency-Key")),
	})
	if err != nil {
		handleDomainError(w, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, mapTransferResponse(result))
}

func pickIdempotencyKey(reference, header string) string {
	if trimmed := strings.TrimSpace(reference); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(header)
}

func (h *TransferHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context", nil)
		return
	}

	transferID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_TRANSFER_ID", "invalid transfer id", nil)
		return
	}

	result, err := h.service.GetTransfer(r.Context(), userID, transferID)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, mapTransferResponse(result))
}

func (h *TransferHandler) ListByAccount(w http.ResponseWriter, r *http.Request) {
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

	limit, offset, err := parseTransferListPagination(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_PAGINATION", err.Error(), nil)
		return
	}

	items, err := h.service.ListTransfers(r.Context(), userID, accountID, limit, offset)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	resp := make([]dto.TransferResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, mapTransferResponse(item))
	}

	respondWithJSON(w, http.StatusOK, resp)
}

func parseTransferListPagination(r *http.Request) (int, int, error) {
	limit, err := parseRequiredIntQuery(r, "limit", defaultTransferListLimit)
	if err != nil {
		return 0, 0, err
	}
	if limit <= 0 {
		return 0, 0, &paginationError{message: "limit must be greater than 0"}
	}
	if limit > maxTransferListLimit {
		return 0, 0, &paginationError{message: "limit must be less than or equal to 200"}
	}

	offset, err := parseRequiredIntQuery(r, "offset", 0)
	if err != nil {
		return 0, 0, err
	}
	if offset < 0 {
		return 0, 0, &paginationError{message: "offset must be greater than or equal to 0"}
	}

	return limit, offset, nil
}

type paginationError struct {
	message string
}

func (e *paginationError) Error() string {
	return e.message
}

func parseRequiredIntQuery(r *http.Request, key string, fallback int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, &paginationError{message: key + " must be an integer"}
	}

	return value, nil
}

func mapTransferResponse(s *service.TransferResponse) dto.TransferResponse {
	return dto.TransferResponse{
		ID:              s.ID.String(),
		FromAccountID:   s.FromAccountID.String(),
		ToAccountID:     s.ToAccountID.String(),
		Amount:          s.Amount,
		Currency:        s.Currency,
		Status:          s.Status,
		ReferenceNumber: s.ReferenceNumber,
		Description:     s.Description,
		CreatedAt:       s.CreatedAt,
		CompletedAt:     s.CompletedAt,
	}
}
