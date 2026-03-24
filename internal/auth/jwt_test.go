package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type unauthorizedResponse struct {
	Error struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Details interface{} `json:"details,omitempty"`
	} `json:"error"`
}

func TestRequireAuth_MissingBearerHeaderReturnsJSON(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)
	protected := RequireAuth(jwtManager)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	res := httptest.NewRecorder()

	protected.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
	if got := res.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected application/json content type, got %q", got)
	}

	var payload unauthorizedResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON response body, got error: %v", err)
	}
	if payload.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected error code UNAUTHORIZED, got %q", payload.Error.Code)
	}
	if payload.Error.Message != "unauthorized" {
		t.Fatalf("expected error message unauthorized, got %q", payload.Error.Message)
	}
}

func TestRequireAuth_InvalidBearerTokenReturnsJSON(t *testing.T) {
	jwtManager := NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)
	protected := RequireAuth(jwtManager)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	res := httptest.NewRecorder()

	protected.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}
	if got := res.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected application/json content type, got %q", got)
	}

	var payload unauthorizedResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON response body, got error: %v", err)
	}
	if payload.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("expected error code UNAUTHORIZED, got %q", payload.Error.Code)
	}
	if payload.Error.Message != "unauthorized" {
		t.Fatalf("expected error message unauthorized, got %q", payload.Error.Message)
	}
}
