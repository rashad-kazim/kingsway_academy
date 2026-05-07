package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"kingsway/backend/internal/academic"
	"kingsway/backend/internal/admin"
	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/files"
	"kingsway/backend/internal/finance"
	"kingsway/backend/internal/notification"
	"kingsway/backend/internal/store"
)

func TestHTTPIdempotentCreateBranchReplaysCompletedResponse(t *testing.T) {
	t.Parallel()

	handler, token := newTestServer(t)
	stamp := time.Now().UTC().UnixNano()
	payload := fmt.Sprintf(`{"name":"Idempotent %d","slug":"idempotent-%d","address":"Main"}`, stamp, stamp)
	key := fmt.Sprintf("idem-%d", stamp)

	first := authedJSON(t, handler, token, http.MethodPost, "/v1/branches", payload, key)
	if first.Code != http.StatusCreated {
		t.Fatalf("expected first create status 201, got %d: %s", first.Code, first.Body.String())
	}
	var firstBranch domain.Branch
	decodeRecorder(t, first, &firstBranch)

	second := authedJSON(t, handler, token, http.MethodPost, "/v1/branches", payload, key)
	if second.Code != http.StatusCreated {
		t.Fatalf("expected replay status 201, got %d: %s", second.Code, second.Body.String())
	}
	var secondBranch domain.Branch
	decodeRecorder(t, second, &secondBranch)
	if secondBranch.ID != firstBranch.ID {
		t.Fatalf("expected replayed response id %q, got %q", firstBranch.ID, secondBranch.ID)
	}

	conflict := authedJSON(t, handler, token, http.MethodPost, "/v1/branches", strings.Replace(payload, "Main", "Other", 1), key)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("expected same key/different body conflict, got %d: %s", conflict.Code, conflict.Body.String())
	}

	list := authedJSON(t, handler, token, http.MethodGet, "/v1/branches", "", "")
	var branches []domain.Branch
	decodeRecorder(t, list, &branches)
	matches := 0
	for _, branch := range branches {
		if branch.Name == firstBranch.Name {
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("expected one persisted branch, got %d", matches)
	}
}

func TestHTTPConcurrentIdempotentCreateBranchDoesNotDuplicate(t *testing.T) {
	t.Parallel()

	handler, token := newTestServer(t)
	stamp := time.Now().UTC().UnixNano()
	name := fmt.Sprintf("Concurrent %d", stamp)
	payload := fmt.Sprintf(`{"name":%q,"slug":"concurrent-%d","address":"Main"}`, name, stamp)
	key := fmt.Sprintf("concurrent-%d", stamp)

	const attempts = 50
	statuses := make([]int, attempts)
	var wg sync.WaitGroup
	wg.Add(attempts)
	for i := 0; i < attempts; i++ {
		go func(index int) {
			defer wg.Done()
			recorder := authedJSON(t, handler, token, http.MethodPost, "/v1/branches", payload, key)
			statuses[index] = recorder.Code
		}(i)
	}
	wg.Wait()

	for _, status := range statuses {
		if status != http.StatusCreated && status != http.StatusConflict {
			t.Fatalf("expected concurrent statuses to be 201 or 409, got %#v", statuses)
		}
	}

	list := authedJSON(t, handler, token, http.MethodGet, "/v1/branches", "", "")
	var branches []domain.Branch
	decodeRecorder(t, list, &branches)
	matches := 0
	for _, branch := range branches {
		if branch.Name == name {
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("expected one persisted branch after %d attempts, got %d", attempts, matches)
	}
}

func newTestServer(t *testing.T) (http.Handler, string) {
	t.Helper()

	repo := store.NewMemory()
	authService := auth.NewService(repo, "test-secret")
	academicService := academic.NewService(repo, authService)
	financeService := finance.NewService(repo)
	filesService := files.NewService(repo)
	notificationService := notification.NewService(repo)
	adminService := admin.NewService(repo)
	server := New(
		authService,
		academicService,
		financeService,
		filesService,
		notificationService,
		adminService,
		repo,
		zap.NewNop(),
		Options{RateLimitEnabled: false},
	)

	body := `{"email":"owner@test.local","password":"Kingsway123!","first_name":"Owner","last_name":"User"}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/bootstrap-owner", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("bootstrap owner failed: %d %s", recorder.Code, recorder.Body.String())
	}
	var result auth.LoginResult
	decodeRecorder(t, recorder, &result)
	if result.Token == "" {
		t.Fatal("expected bootstrap token")
	}

	return server.Handler(), result.Token
}

func authedJSON(t *testing.T, handler http.Handler, token string, method string, path string, body string, idempotencyKey string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Buffer
	if body == "" {
		reader = bytes.NewBuffer(nil)
	} else {
		reader = bytes.NewBufferString(body)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		request.Header.Set(idempotencyKeyHeader, idempotencyKey)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeRecorder(t *testing.T, recorder *httptest.ResponseRecorder, out any) {
	t.Helper()

	if err := json.Unmarshal(recorder.Body.Bytes(), out); err != nil {
		t.Fatalf("decode response %d %q: %v", recorder.Code, recorder.Body.String(), err)
	}
}
