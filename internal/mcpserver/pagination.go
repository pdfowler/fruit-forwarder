package mcpserver

import "fmt"

const (
	defaultPageSize = 100
	maxPageSize     = 100
	maxPageOffset   = 1_000_000
)

func pageBounds(offset, limit, total int) (int, int, *int, error) {
	if offset < 0 || offset > maxPageOffset {
		return 0, 0, nil, fmt.Errorf("offset must be between 0 and %d", maxPageOffset)
	}
	if limit == 0 {
		limit = defaultPageSize
	}
	if limit < 1 || limit > maxPageSize {
		return 0, 0, nil, fmt.Errorf("limit must be between 1 and %d", maxPageSize)
	}
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	var next *int
	if end < total {
		next = &end
	}
	return start, end, next, nil
}
