package store

import "kingsway/backend/internal/domain"

func normalizePage(page domain.PageRequest) domain.PageRequest {
	if page.Limit <= 0 || page.Limit > 500 {
		page.Limit = 100
	}
	if page.Offset < 0 {
		page.Offset = 0
	}
	return page
}

func pageSlice[T any](items []T, page domain.PageRequest) ([]T, int) {
	page = normalizePage(page)
	total := len(items)
	start := page.Offset
	if start > total {
		start = total
	}
	end := start + page.Limit
	if end > total {
		end = total
	}
	return items[start:end], total
}
