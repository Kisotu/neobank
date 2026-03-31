package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/Kisotu/neobank/internal/auth"
	"github.com/Kisotu/neobank/internal/domain"
	"github.com/Kisotu/neobank/internal/handler/dto"
	"github.com/Kisotu/neobank/internal/service"
)

type AuthHandler struct {
	service    service.UserService
	jwtManager *auth.JWTManager
	validator  *validator.Validate
}

func NewAuthHandler(s service.UserService, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		service:    s,
		jwtManager: jwtManager,
		validator:  validator.New(),
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respondWithError(w, http.StatusBadRequest, "VALIDATION_FAILED", "request validation failed", err.Error())
		return
	}

	result, err := h.service.Register(r.Context(), &service.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	})
	if err != nil {
		handleDomainError(w, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, dto.UserProfileResponse{
		ID:       result.ID.String(),
		Email:    result.Email,
		FullName: result.FullName,
		Status:   result.Status,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respondWithError(w, http.StatusBadRequest, "VALIDATION_FAILED", "request validation failed", err.Error())
		return
	}

	result, err := h.service.Login(r.Context(), &service.LoginRequest{Email: req.Email, Password: req.Password})
	if err != nil {
		handleDomainError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dto.AuthResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respondWithError(w, http.StatusBadRequest, "VALIDATION_FAILED", "request validation failed", err.Error())
		return
	}

	userID, tokenType, err := h.jwtManager.ParseToken(req.RefreshToken)
	if err != nil || tokenType != "refresh" {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token", nil)
		return
	}

	accessToken, refreshToken, expiresIn, err := h.jwtManager.GenerateTokenPair(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to refresh token", nil)
		return
	}

	respondWithJSON(w, http.StatusOK, dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, _ *http.Request) {
	// Logout is intentionally stateless: clients discard tokens locally.
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context", nil)
		return
	}

	result, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, dto.UserProfileResponse{
		ID:       result.ID.String(),
		Email:    result.Email,
		FullName: result.FullName,
		Status:   result.Status,
	})
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth context", nil)
		return
	}

	var req dto.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		respondWithError(w, http.StatusBadRequest, "VALIDATION_FAILED", "request validation failed", err.Error())
		return
	}

	err := h.service.UpdateProfile(r.Context(), userID, &service.UpdateProfileRequest{
		Email:    req.Email,
		FullName: req.FullName,
	})
	if err != nil {
		handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func isUnauthorized(err error) bool {
	return errors.Is(err, domain.ErrUnauthorized)
}
