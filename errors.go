package asana

func (r *response) Error(statusCode int, errorType string) error {
	if r.Errors != nil {
		return r.Errors[0].withType(statusCode, errorType)
	}

	return &Error{
		StatusCode: statusCode,
		Type:       errorType,
		Message:    "Unknown error",
	}
}

// Error is an error message returned by the API
type Error struct {
	StatusCode int
	Type       string
	Message    string `json:"message"`
	Phrase     string `json:"phrase"`
	Help       string `json:"help"`
}

func (err Error) Error() string {
	return err.Type + ": " + err.Message
}

func (err *Error) withType(statusCode int, errorType string) *Error {
	err.StatusCode = statusCode
	err.Type = errorType
	return err
}

// IsNotFoundError checks if the provided error represents a 404 not found response from the API
func IsNotFoundError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.StatusCode == 404
	}
	return false
}

// IsAuthError checks if the provided error represents a 401 Authorization error response from the API
func IsAuthError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.StatusCode == 401
	}
	return false
}
