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
	RunIdempotent(ctx context.Context, record domain.IdempotencyRecord, execute func(context.Context) (int, []byte, error)) (domain.IdempotencyRunResult, error)
}

type idempotentJSONHandler func(context.Context) (int, any, error)

func (s *Server) decodeIdempotentJSON(w http.ResponseWriter, r *http.Request, principal domain.Principal, out any, execute idempotentJSONHandler) {
	raw, ok := decodeJSONBody(w, r, out)
	if !ok {
		return
	}
	s.writeIdempotentJSON(w, r, principal, raw, execute)
}

func (s *Server) writeIdempotentNoBodyJSON(w http.ResponseWriter, r *http.Request, principal domain.Principal, execute idempotentJSONHandler) {
	s.writeIdempotentJSON(w, r, principal, []byte("{}"), execute)
}

func (s *Server) writeIdempotentJSON(w http.ResponseWriter, r *http.Request, principal domain.Principal, rawBody []byte, execute idempotentJSONHandler) {
	key := strings.TrimSpace(r.Header.Get(idempotencyKeyHeader))
	if key == "" || s.idem == nil {
		writeJSONResult(w, r.Context(), execute)
		return
	}
	if len(key) > 160 {
		writeError(w, domain.ErrInvalidInput)
		return
	}

	hashBytes := sha256.Sum256(rawBody)
	requestHash := hex.EncodeToString(hashBytes[:])
	result, err := s.idem.RunIdempotent(r.Context(), domain.IdempotencyRecord{
		ActorUserID: principal.UserID,
		Method:      r.Method,
		Path:        r.URL.Path,
		Key:         key,
		RequestHash: requestHash,
		ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
	}, func(ctx context.Context) (int, []byte, error) {
		status, body, err := execute(ctx)
		if err != nil {
			return 0, nil, err
		}
		payload, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}

		return status, payload, nil
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeRawJSON(w, result.ResponseStatus, result.ResponseBody)
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, out any) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer func() { _ = r.Body.Close() }()

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

func writeJSONResult(w http.ResponseWriter, ctx context.Context, execute idempotentJSONHandler) {
	status, body, err := execute(ctx)
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
