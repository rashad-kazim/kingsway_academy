package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"kingsway/backend/internal/domain"
)

const idempotencyKeyHeader = "Idempotency-Key"

type IdempotencyStore interface {
	BeginIdempotency(ctx context.Context, record domain.IdempotencyRecord) (domain.IdempotencyBeginResult, error)
	CompleteIdempotency(ctx context.Context, actorUserID string, method string, path string, key string, responseStatus int, responseBody []byte) error
	ClearIdempotency(ctx context.Context, actorUserID string, method string, path string, key string) error
}

type idempotentJSONHandler func() (int, any, error)

func (s *Server) decodeIdempotentJSON(w http.ResponseWriter, r *http.Request, principal domain.Principal, out any, execute idempotentJSONHandler) {
	raw, ok := decodeJSONBody(w, r, out)
	if !ok {
		return
	}
	s.writeIdempotentJSON(w, r, principal, raw, execute)
}

func (s *Server) writeIdempotentJSON(w http.ResponseWriter, r *http.Request, principal domain.Principal, rawBody []byte, execute idempotentJSONHandler) {
	key := strings.TrimSpace(r.Header.Get(idempotencyKeyHeader))
	if key == "" || s.idem == nil {
		writeJSONResult(w, execute)
		return
	}
	if len(key) > 160 {
		writeError(w, domain.ErrInvalidInput)
		return
	}

	hashBytes := sha256.Sum256(rawBody)
	requestHash := hex.EncodeToString(hashBytes[:])
	begin, err := s.idem.BeginIdempotency(r.Context(), domain.IdempotencyRecord{
		ActorUserID: principal.UserID,
		Method:      r.Method,
		Path:        r.URL.Path,
		Key:         key,
		RequestHash: requestHash,
		ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !begin.Started {
		if begin.Record.Status == domain.IdempotencyStatusCompleted && len(begin.Record.ResponseBody) > 0 {
			writeRawJSON(w, begin.Record.ResponseStatus, begin.Record.ResponseBody)
			return
		}
		writeError(w, domain.ErrConflict)
		return
	}

	status, body, err := execute()
	if err != nil {
		_ = s.idem.ClearIdempotency(r.Context(), principal.UserID, r.Method, r.URL.Path, key)
		writeError(w, err)
		return
	}
	payload, err := json.Marshal(body)
	if err != nil {
		_ = s.idem.ClearIdempotency(r.Context(), principal.UserID, r.Method, r.URL.Path, key)
		writeError(w, err)
		return
	}
	if err := s.idem.CompleteIdempotency(r.Context(), principal.UserID, r.Method, r.URL.Path, key, status, payload); err != nil {
		writeError(w, err)
		return
	}
	writeRawJSON(w, status, payload)
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, out any) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, domain.ErrInvalidInput)
		return nil, false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		writeError(w, domain.ErrInvalidInput)
		return nil, false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(w, domain.ErrInvalidInput)
		return nil, false
	}

	return raw, true
}

func writeJSONResult(w http.ResponseWriter, execute idempotentJSONHandler) {
	status, body, err := execute()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, status, body)
}

func writeRawJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
