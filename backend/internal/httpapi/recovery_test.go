package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestRecoveryMiddlewareReturnsGenericInternalError(t *testing.T) {
	t.Parallel()

	server := &Server{logger: zap.NewNop(), options: normalizeOptions(Options{RateLimitEnabled: false})}
	handler := server.withRecovery(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("sensitive panic detail")
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
	if body := recorder.Body.String(); body != "{\"error\":\"internal error\"}\n" {
		t.Fatalf("unexpected response body: %s", body)
	}
}
