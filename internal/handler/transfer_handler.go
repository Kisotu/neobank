package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/username/banking-app/internal/handler/dto"
	"github.com/username/banking-app/internal/service"
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

	result, err := h.service.Transfer(r.Context(), &service.TransferRequest{
		FromAccountID: fromID,
		ToAccountID:   toID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Description:   req.Description,
	})
	if err != nil {
		handleDomainError(w, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, mapTransferResponse(result))
}

func (h *TransferHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	transferID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_TRANSFER_ID", "invalid transfer id", nil)
		return
	}

	result, err := h.service.GetTransfer(r.Context(), transferID)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, mapTransferResponse(result))
}

func (h *TransferHandler) ListByAccount(w http.ResponseWriter, r *http.Request) {
	accountID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_ACCOUNT_ID", "invalid account id", nil)
		return
	}

	items, err := h.service.ListTransfers(r.Context(), accountID, 20, 0)
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
