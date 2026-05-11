package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/files"
)

const multipartFormMemoryBytes int64 = 1 << 20
const maxUploadRequestBytes int64 = maxUploadFileBytes + multipartFormMemoryBytes

func (s *Server) registerFile(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input files.RegisterFileInput
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		file, err := s.files.RegisterFile(ctx, principal, input)
		if err == nil {
			setAuditEntity(ctx, "files", file.ID, file.BranchID)
		}
		return http.StatusCreated, file, err
	})
}

func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadRequestBytes)
	if err := r.ParseMultipartForm(multipartFormMemoryBytes); err != nil {
		writeError(w, domain.ErrInvalidInput)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	content, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, domain.ErrInvalidInput)
		return
	}
	defer func() { _ = content.Close() }()
	if header.Size > maxUploadFileBytes {
		writeError(w, domain.ErrInvalidInput)
		return
	}
	contentBytes, err := io.ReadAll(io.LimitReader(content, maxUploadFileBytes+1))
	if err != nil || int64(len(contentBytes)) > maxUploadFileBytes {
		writeError(w, domain.ErrInvalidInput)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if strings.TrimSpace(mimeType) == "" {
		mimeType = "application/octet-stream"
	}
	size := header.Size
	if size <= 0 {
		size = int64(len(contentBytes))
	}

	contentHash := sha256.Sum256(contentBytes)
	raw, _ := json.Marshal(map[string]any{
		"branch_id":         strings.TrimSpace(r.FormValue("branch_id")),
		"owner_type":        strings.TrimSpace(r.FormValue("owner_type")),
		"owner_id":          strings.TrimSpace(r.FormValue("owner_id")),
		"category":          strings.TrimSpace(r.FormValue("category")),
		"policy":            strings.TrimSpace(r.FormValue("policy")),
		"purpose":           strings.TrimSpace(r.FormValue("purpose")),
		"original_filename": header.Filename,
		"mime_type":         strings.TrimSpace(mimeType),
		"size":              size,
		"sha256":            hex.EncodeToString(contentHash[:]),
	})
	s.writeIdempotentJSON(w, r, principal, raw, func(ctx context.Context) (int, any, error) {
		file, err := s.files.UploadFile(ctx, principal, files.UploadFileInput{
			BranchID:         r.FormValue("branch_id"),
			OwnerType:        r.FormValue("owner_type"),
			OwnerID:          r.FormValue("owner_id"),
			Category:         domain.FileCategory(r.FormValue("category")),
			Policy:           domain.FilePolicy(r.FormValue("policy")),
			Purpose:          domain.FilePurpose(r.FormValue("purpose")),
			OriginalFilename: header.Filename,
			MimeType:         mimeType,
			Size:             size,
			Content:          bytes.NewReader(contentBytes),
		})
		if err == nil {
			setAuditEntity(ctx, "files", file.ID, file.BranchID)
		}
		return http.StatusCreated, file, err
	})
}

func (s *Server) fileAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/files/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if r.Method == http.MethodGet && len(parts) == 1 && parts[0] != "" {
		file, err := s.files.GetFile(r.Context(), principal, parts[0])
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, file)
		return
	}
	if r.Method == http.MethodDelete && len(parts) == 1 && parts[0] != "" {
		s.writeIdempotentNoBodyJSON(w, r, principal, func(ctx context.Context) (int, any, error) {
			beforeFile, err := s.files.GetFile(ctx, principal, parts[0])
			if err != nil {
				return 0, nil, err
			}
			setAuditBefore(ctx, beforeFile)
			setAuditEntity(ctx, "files", parts[0], beforeFile.BranchID)
			file, err := s.files.DeleteFile(ctx, principal, parts[0])
			return http.StatusOK, file, err
		})
		return
	}
	if len(parts) != 2 || parts[1] != "download-url" {
		writeError(w, domain.ErrNotFound)
		return
	}

	result, err := s.files.CreateDownloadURL(r.Context(), principal, parts[0])
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) cleanupFileRetention(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	limit := 100
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeError(w, domain.ErrInvalidInput)
			return
		}
		limit = parsed
	}

	raw, _ := json.Marshal(map[string]any{"limit": limit})
	s.writeIdempotentJSON(w, r, principal, raw, func(ctx context.Context) (int, any, error) {
		result, err := s.files.CleanupExpiredFiles(ctx, principal, limit)
		return http.StatusOK, result, err
	})
}

func (s *Server) listFiles(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	ownerType := strings.TrimSpace(r.URL.Query().Get("owner_type"))
	ownerID := strings.TrimSpace(r.URL.Query().Get("owner_id"))
	purpose := domain.FilePurpose(strings.TrimSpace(r.URL.Query().Get("purpose")))
	files, total, err := s.files.ListFilesPage(r.Context(), principal, r.URL.Query().Get("branch_id"), ownerType, ownerID, purpose, page.domainPage())
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, files)
}
