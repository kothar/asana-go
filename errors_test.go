package asana

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/h2non/gock"
)

func TestCauseWrappedError(t *testing.T) {
	cause := &Error{StatusCode: 500}

	wrap1 := fmt.Errorf("Wrapping 1: %w", cause)
	wrap2 := fmt.Errorf("Wrapping 2: %w", wrap1)

	if !IsRecoverableError(cause) {
		t.Error("Expected original error to be recoverable")
	}
	if !IsRecoverableError(wrap1) {
		t.Error("Expected wrapped error to be recoverable")
	}
	if !IsRecoverableError(wrap2) {
		t.Error("Expected double-wrapped error to be recoverable")
	}
}

func TestIsPaymentRequired(t *testing.T) {
	cause := &Error{StatusCode: 402}
	wrapped := fmt.Errorf("Wrapping: %w", cause)

	if !IsPaymentRequired(cause) {
		t.Error("Expected 402 error to be payment required")
	}
	if !IsPaymentRequired(wrapped) {
		t.Error("Expected wrapped 402 error to be payment required")
	}
	if IsPaymentRequired(&Error{StatusCode: 403}) {
		t.Error("Expected 403 error not to be payment required")
	}
	if IsPaymentRequired(fmt.Errorf("not an asana error")) {
		t.Error("Expected non-Asana error not to be payment required")
	}
}

func TestRateLimitedResponseCarriesRetryAfter(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/workspaces/1234").
		Reply(429).
		SetHeader("Retry-After", "42").
		JSON(o{"errors": []o{{"message": "You have made too many requests recently."}}})

	client := NewClient(http.DefaultClient)
	err := (&Workspace{ID: "1234"}).Fetch(client)

	if !IsRateLimited(err) {
		t.Fatalf("Expected a rate limit error but saw %v", err)
	}
	if got := RetryAfter(err); got != 42*time.Second {
		t.Errorf("Expected RetryAfter of 42s but saw %s", got)
	}
}

func TestRateLimitedResponseWithUnreadableRetryAfter(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/workspaces/1234").
		Reply(429).
		SetHeader("Retry-After", "soon").
		JSON(o{"errors": []o{{"message": "You have made too many requests recently."}}})

	client := NewClient(http.DefaultClient)
	err := (&Workspace{ID: "1234"}).Fetch(client)

	if got := RetryAfter(err); got != 0 {
		t.Errorf("Expected no RetryAfter for an unreadable header but saw %s", got)
	}
}

func TestErrorClassification(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		recoverable bool
		fatal       bool
		rateLimited bool
		notFound    bool
		auth        bool
	}{
		{name: "server error", err: &Error{StatusCode: 503}, recoverable: true},
		{name: "rate limited", err: &Error{StatusCode: 429}, fatal: true, rateLimited: true},
		{name: "not found", err: &Error{StatusCode: 404}, fatal: true, notFound: true},
		{name: "unauthorized", err: &Error{StatusCode: 401}, fatal: true, auth: true},
		{name: "bad request", err: &Error{StatusCode: 400}, fatal: true},
		{name: "wrapped server error", err: fmt.Errorf("wrap: %w", &Error{StatusCode: 500}), recoverable: true},
		{name: "not an asana error", err: errors.New("connection reset")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRecoverableError(tt.err); got != tt.recoverable {
				t.Errorf("IsRecoverableError = %v, want %v", got, tt.recoverable)
			}
			if got := IsFatalError(tt.err); got != tt.fatal {
				t.Errorf("IsFatalError = %v, want %v", got, tt.fatal)
			}
			if got := IsRateLimited(tt.err); got != tt.rateLimited {
				t.Errorf("IsRateLimited = %v, want %v", got, tt.rateLimited)
			}
			if got := IsNotFoundError(tt.err); got != tt.notFound {
				t.Errorf("IsNotFoundError = %v, want %v", got, tt.notFound)
			}
			if got := IsAuthError(tt.err); got != tt.auth {
				t.Errorf("IsAuthError = %v, want %v", got, tt.auth)
			}
		})
	}
}

func TestRetryAfterDefaultsForOtherErrors(t *testing.T) {
	if got := RetryAfter(&Error{StatusCode: 500, RetryAfter: 5 * time.Second}); got != time.Minute {
		t.Errorf("Expected a one minute default for a non-429 error but saw %s", got)
	}
	if got := RetryAfter(errors.New("not an asana error")); got != time.Minute {
		t.Errorf("Expected a one minute default for a non-Asana error but saw %s", got)
	}
}
