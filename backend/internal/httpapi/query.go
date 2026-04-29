package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"kingsway/backend/internal/domain"
)

type pageParams struct {
	Limit  int
	Offset int
}

func parsePage(r *http.Request) (pageParams, error) {
	limit := 100
	offset := 0

	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 500 {
			return pageParams{}, domain.ErrInvalidInput
		}
		limit = parsed
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			return pageParams{}, domain.ErrInvalidInput
		}
		offset = parsed
	}

	return pageParams{Limit: limit, Offset: offset}, nil
}

func writePageHeaders(w http.ResponseWriter, page pageParams, total int) {
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Limit", strconv.Itoa(page.Limit))
	w.Header().Set("X-Offset", strconv.Itoa(page.Offset))
}

func writePagedJSON[T any](w http.ResponseWriter, r *http.Request, items []T) bool {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return false
	}

	total := len(items)
	start := page.Offset
	if start > total {
		start = total
	}
	end := start + page.Limit
	if end > total {
		end = total
	}

	writePageHeaders(w, page, total)
	writeJSON(w, http.StatusOK, items[start:end])
	return true
}

func parseBoolQuery(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	default:
		return false, false
	}
}
