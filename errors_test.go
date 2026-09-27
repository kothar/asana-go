package asana

import (
	"fmt"
	"testing"
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
