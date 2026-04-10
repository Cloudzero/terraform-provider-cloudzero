package client

import "context"

// Paginate fetches all pages from a paginated CloudZero API endpoint.
// The fetchPage function should return the items and next cursor (empty string if done).
func Paginate[T any](ctx context.Context, fetchPage func(ctx context.Context, cursor string) ([]T, string, error)) ([]T, error) {
	var all []T
	cursor := ""

	for {
		items, nextCursor, err := fetchPage(ctx, cursor)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return all, nil
}
