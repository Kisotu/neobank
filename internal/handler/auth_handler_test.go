package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Kisotu/neobank/internal/auth"
	"github.com/Kisotu/neobank/internal/handler/dto"
)

func TestAuthHandler_Logout_ReturnsNoContent(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)
	h := NewAuthHandler(nil, jwtManager)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()

	h.Logout(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("expected empty response body, got %q", rr.Body.String())
	}
}

func TestAuthHandler_Refresh_RemainsValidAfterLogout(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)
	h := NewAuthHandler(nil, jwtManager)

	userID := uuid.New()
	_, refreshToken, _, err := jwtManager.GenerateTokenPair(userID)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRes := httptest.NewRecorder()
	h.Logout(logoutRes, logoutReq)
	if logoutRes.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, logoutRes.Code)
	}

	payload, err := json.Marshal(dto.RefreshTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		t.Fatalf("failed to marshal refresh request: %v", err)
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(payload))
	refreshRes := httptest.NewRecorder()
	h.Refresh(refreshRes, refreshReq)

	if refreshRes.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, refreshRes.Code)
	}

	var resp dto.AuthResponse
	if err := json.Unmarshal(refreshRes.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
	if resp.ExpiresIn <= 0 {
		t.Fatalf("expected positive expires_in, got %d", resp.ExpiresIn)
	}
}
