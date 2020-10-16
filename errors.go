package asana

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/xid"
)

func (r *Response) Error(resp *http.Response, requestID xid.ID) error {
	var asanaError *Error
	if r.Errors != nil {
		asanaError = r.Errors[0].withType(resp.StatusCode, resp.Status)
	} else {
		asanaError = &Error{
			StatusCode: resp.StatusCode,
			Type:       resp.Status,
			Message:    "Unknown error",
			RequestID:  requestID.String(),
		}
	}

	retryHeader := resp.Header.Get("Retry-After")
	if retryHeader != "" {
		retryAfter, err := strconv.ParseInt(retryHeader, 10, 64)
		if err != nil {
			asanaError.RetryAfter = time.Duration(retryAfter) * time.Second
		}
	}

	return asanaError
}

// Error is an error message returned by the API
type Error struct {
	StatusCode int
	Type       string
	Message    string        `json:"message"`
	Phrase     string        `json:"phrase"`
	Help       string        `json:"help"`
	RetryAfter time.Duration `json:"-"`
	RequestID  string        `json:"-"`
}

func (err Error) Error() string {
	return fmt.Sprintf("%s %d: %s", err.RequestID, err.StatusCode, err.Message)
}

func IsAsanaError(err error) (*Error, bool) {
	cause := errors.Cause(err)
	if e, ok := cause.(*Error); ok {
		return e, true
	}
	return nil, false
}

func (err *Error) withType(statusCode int, errorType string) *Error {
	err.StatusCode = statusCode
	err.Type = errorType
	return err
}

func IsRecoverableError(err error) bool {
	if e, ok := IsAsanaError(err); ok {
		return e.StatusCode >= 500 && e.StatusCode < 600
	}
	return false
}

// IsNotFoundError checks if the provided error represents a 404 not found response from the API
func IsNotFoundError(err error) bool {
	if e, ok := IsAsanaError(err); ok {
		return e.StatusCode == 404
	}
	return false
}

// IsAuthError checks if the provided error represents a 401 Authorization error response from the API
func IsAuthError(err error) bool {
	if e, ok := IsAsanaError(err); ok {
		return e.StatusCode == 401
	}
	return false
}

// IsRateLimited returns true if the error was a rate limit error
func IsRateLimited(err error) bool {
	if e, ok := IsAsanaError(err); ok {
		return e.StatusCode == 429
	}
	return false
}

// IsRateLimited returns true if the error was a rate limit error
func IsPayloadTooLarge(err error) bool {
	if e, ok := IsAsanaError(err); ok {
		return e.StatusCode == 413
	}
	return false
}

// RetryAfter returns a Duration indicating after how many seconds a rate-limited requests may be retried
// or nil if the error was not a rate limit error
func RetryAfter(err error) time.Duration {
	if e, ok := IsAsanaError(err); ok {
		if e.StatusCode == 429 {
			return e.RetryAfter
		}
	}
	return time.Minute
}
