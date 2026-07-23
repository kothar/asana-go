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
