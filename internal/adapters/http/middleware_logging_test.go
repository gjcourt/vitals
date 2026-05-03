package adapthttp

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	s := &Server{logger: logger}
	// Create a dummy handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap it
	wrapped := s.loggingMiddleware(nextHandler)

	req := httptest.NewRequest("GET", "/test-path", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusTeapot {
		t.Errorf("Expected status %d, got %d", http.StatusTeapot, w.Code)
	}

	// Check log
	logOutput := buf.String()
	if !strings.Contains(logOutput, "method=GET") || !strings.Contains(logOutput, "path=/test-path") || !strings.Contains(logOutput, "status=418") {
		t.Errorf("Log output missing expected fields. Got: %s", logOutput)
	}
}
