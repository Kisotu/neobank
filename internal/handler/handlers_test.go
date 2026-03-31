package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Kisotu/neobank/internal/auth"
	"github.com/Kisotu/neobank/internal/service"
)

type transferServiceSpy struct {
	called    bool
	userID    uuid.UUID
	accountID uuid.UUID
	limit     int
	offset    int
	listErr   error
	items     []*service.TransferResponse
}

func (s *transferServiceSpy) Transfer(_ context.Context, _ uuid.UUID, _ *service.TransferRequest) (*service.TransferResponse, error) {
	panic("unexpected call to Transfer")
}

func (s *transferServiceSpy) GetTransfer(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*service.TransferResponse, error) {
	panic("unexpected call to GetTransfer")
}

func (s *transferServiceSpy) ListTransfers(_ context.Context, userID, accountID uuid.UUID, limit, offset int) ([]*service.TransferResponse, error) {
	s.called = true
	s.userID = userID
	s.accountID = accountID
	s.limit = limit
	s.offset = offset
	return s.items, s.listErr
}

func withAccountIDParam(r *http.Request, accountID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", accountID)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func decodeErrorResponse(t *testing.T, body *httptest.ResponseRecorder) ErrorResponse {
	t.Helper()

	var resp ErrorResponse
	if err := json.Unmarshal(body.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	return resp
}

func TestTransferHandler_ListByAccount_UsesDefaultPagination(t *testing.T) {
	serviceSpy := &transferServiceSpy{items: []*service.TransferResponse{}}
	h := NewTransferHandler(serviceSpy)

	userID := uuid.New()
	accountID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/"+accountID.String()+"/transfers", nil)
	req = req.WithContext(auth.ContextWithUserID(req.Context(), userID))
	req = withAccountIDParam(req, accountID.String())

	rr := httptest.NewRecorder()
	h.ListByAccount(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if !serviceSpy.called {
		t.Fatal("expected ListTransfers to be called")
	}
	if serviceSpy.limit != 20 {
		t.Fatalf("expected default limit 20, got %d", serviceSpy.limit)
	}
	if serviceSpy.offset != 0 {
		t.Fatalf("expected default offset 0, got %d", serviceSpy.offset)
	}
	if serviceSpy.userID != userID {
		t.Fatalf("expected user id %s, got %s", userID, serviceSpy.userID)
	}
	if serviceSpy.accountID != accountID {
		t.Fatalf("expected account id %s, got %s", accountID, serviceSpy.accountID)
	}
}

func TestTransferHandler_ListByAccount_UsesProvidedPagination(t *testing.T) {
	serviceSpy := &transferServiceSpy{items: []*service.TransferResponse{}}
	h := NewTransferHandler(serviceSpy)

	userID := uuid.New()
	accountID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/"+accountID.String()+"/transfers?limit=35&offset=9", nil)
	req = req.WithContext(auth.ContextWithUserID(req.Context(), userID))
	req = withAccountIDParam(req, accountID.String())

	rr := httptest.NewRecorder()
	h.ListByAccount(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if !serviceSpy.called {
		t.Fatal("expected ListTransfers to be called")
	}
	if serviceSpy.limit != 35 {
		t.Fatalf("expected limit 35, got %d", serviceSpy.limit)
	}
	if serviceSpy.offset != 9 {
		t.Fatalf("expected offset 9, got %d", serviceSpy.offset)
	}
}

func TestTransferHandler_ListByAccount_InvalidPagination(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		expectMsg string
	}{
		{name: "non integer limit", query: "limit=abc", expectMsg: "limit must be an integer"},
		{name: "non integer offset", query: "offset=abc", expectMsg: "offset must be an integer"},
		{name: "negative offset", query: "offset=-1", expectMsg: "offset must be greater than or equal to 0"},
		{name: "zero limit", query: "limit=0", expectMsg: "limit must be greater than 0"},
		{name: "negative limit", query: "limit=-1", expectMsg: "limit must be greater than 0"},
		{name: "limit too large", query: "limit=201", expectMsg: "limit must be less than or equal to 200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceSpy := &transferServiceSpy{}
			h := NewTransferHandler(serviceSpy)

			userID := uuid.New()
			accountID := uuid.New()

			req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/"+accountID.String()+"/transfers?"+tt.query, nil)
			req = req.WithContext(auth.ContextWithUserID(req.Context(), userID))
			req = withAccountIDParam(req, accountID.String())

			rr := httptest.NewRecorder()
			h.ListByAccount(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", rr.Code)
			}
			if serviceSpy.called {
				t.Fatal("expected ListTransfers not to be called")
			}

			errResp := decodeErrorResponse(t, rr)
			if errResp.Error.Code != "INVALID_PAGINATION" {
				t.Fatalf("expected error code INVALID_PAGINATION, got %s", errResp.Error.Code)
			}
			if errResp.Error.Message != tt.expectMsg {
				t.Fatalf("expected error message %q, got %q", tt.expectMsg, errResp.Error.Message)
			}
		})
	}
}

func TestAuthHandler_Login_Scaffold(t *testing.T) {
	t.Skip("TODO: add handler tests for auth login endpoint")
}

func TestAccountHandler_Create_Scaffold(t *testing.T) {
	t.Skip("TODO: add handler tests for account creation endpoint")
}

func TestTransactionHandler_ListByAccount_Scaffold(t *testing.T) {
	t.Skip("TODO: add handler tests for transaction listing endpoint")
}

func TestMapTransferResponse_Scaffold(t *testing.T) {
	completedAt := time.Now().UTC()
	_ = mapTransferResponse(&service.TransferResponse{CompletedAt: &completedAt})
}
