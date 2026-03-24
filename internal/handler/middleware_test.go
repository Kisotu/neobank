package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitBlocksAfterLimit(t *testing.T) {
	mw := RateLimit(2, 2, 2)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r1 := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected first request to pass, got %d", w1.Code)
	}

	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected second request to pass, got %d", w2.Code)
	}

	r3 := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, r3)
	if w3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected third request to be rate limited, got %d", w3.Code)
	}
}
