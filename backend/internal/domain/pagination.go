package domain

func PageSlice[T any](items []T, page PageRequest) ([]T, int) {
	if page.Limit <= 0 || page.Limit > 500 {
		page.Limit = 100
	}
	if page.Offset < 0 {
		page.Offset = 0
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
	return items[start:end], total
}

func SinglePage[T any](item T, page PageRequest) ([]T, int) {
	if page.Offset > 0 {
		return []T{}, 1
	}
	return []T{item}, 1
}
