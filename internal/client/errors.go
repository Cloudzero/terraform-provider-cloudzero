package client

// NotFoundError is returned when the API returns 404.
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

// IsNotFound checks whether an error is a 404 response.
func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}
