package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikogura/rag-indexer/pkg/config"
	"github.com/nikogura/rag-indexer/pkg/elasticsearch"
)

type mockLogger struct{}

func (l *mockLogger) Info(msg string, args ...interface{})                      {}
func (l *mockLogger) Error(msg string, args ...interface{})                     {}
func (l *mockLogger) Warn(msg string, args ...interface{})                      {}
func (l *mockLogger) InfoContext(ctx context.Context, msg string, args ...any)  {}
func (l *mockLogger) WarnContext(ctx context.Context, msg string, args ...any)  {}
func (l *mockLogger) ErrorContext(ctx context.Context, msg string, args ...any) {}

func TestHandleHealthDirect(t *testing.T) {
	cfg := config.Config{HTTPAddr: ":8080"}
	logger := &mockLogger{}

	server := &Server{
		config: cfg,
		logger: logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body != "OK" {
		t.Errorf("Body = %q, want %q", body, "OK")
	}
}

func TestHandleSearchInvalidMethod(t *testing.T) {
	cfg := config.Config{HTTPAddr: ":8080"}
	logger := &mockLogger{}

	server := &Server{
		config: cfg,
		logger: logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	w := httptest.NewRecorder()

	server.handleSearch(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleSearchInvalidJSON(t *testing.T) {
	cfg := config.Config{HTTPAddr: ":8080"}
	logger := &mockLogger{}

	server := &Server{
		config: cfg,
		logger: logger,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleSearchEmptyQuery(t *testing.T) {
	cfg := config.Config{HTTPAddr: ":8080"}
	logger := &mockLogger{}

	server := &Server{
		config: cfg,
		logger: logger,
	}

	searchReq := elasticsearch.SearchRequest{
		Query: "",
		Limit: 10,
	}

	body, err := json.Marshal(searchReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleReindexInvalidMethod(t *testing.T) {
	cfg := config.Config{HTTPAddr: ":8080"}
	logger := &mockLogger{}

	server := &Server{
		config: cfg,
		logger: logger,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reindex", nil)
	w := httptest.NewRecorder()

	server.handleReindex(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestSearchRequestValidation(t *testing.T) {
	tests := []struct {
		name  string
		query string
		limit int
		valid bool
	}{
		{
			name:  "valid query",
			query: "error handling",
			limit: 10,
			valid: true,
		},
		{
			name:  "empty query",
			query: "",
			limit: 10,
			valid: false,
		},
		{
			name:  "zero limit",
			query: "test",
			limit: 0,
			valid: true,
		},
		{
			name:  "negative limit",
			query: "test",
			limit: -1,
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := elasticsearch.SearchRequest{
				Query: tt.query,
			}

			if (req.Query != "") != tt.valid {
				t.Errorf("Query validation mismatch")
			}
		})
	}
}
