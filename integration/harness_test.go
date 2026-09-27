package integration

// These tests check the suite's own plumbing and run without credentials

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kothar/asana-go"
	"github.com/rs/xid"
)

func TestRetryTransport(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		if len(bodies) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Transport: &retryTransport{base: http.DefaultTransport}}
	resp := mustReturn(client.Post(server.URL, "text/plain", strings.NewReader("payload")))(t)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected the retry to succeed, got %d", resp.StatusCode)
	}
	if len(bodies) != 2 || bodies[1] != "payload" {
		t.Errorf("expected the body to be replayed on retry, got %q", bodies)
	}
}

func TestRunTime(t *testing.T) {
	f := &fixture{runID: namePrefix + " " + xid.New().String()}
	created, ok := runTime(f.name(t, "task"))
	if !ok {
		t.Fatal("expected a name made by the suite to be recognised")
	}
	if age := time.Since(created); age < 0 || age > time.Minute {
		t.Errorf("expected the run time to be now, got %s ago", age)
	}

	for _, name := range []string{"Groceries", namePrefix, namePrefix + " not-an-id task", namePrefix + "x " + xid.New().String()} {
		if _, ok := runTime(name); ok {
			t.Errorf("expected %q not to be recognised", name)
		}
	}
}

func TestRetryTransportConnectionReset(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method)
		if len(requests) == 1 {
			// Drop the connection without answering, as a reset would
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = conn.Close()
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Transport: &retryTransport{base: http.DefaultTransport}}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("expected the GET to be retried, got %v", err)
	}
	_ = resp.Body.Close()
	if len(requests) != 2 {
		t.Errorf("expected 2 attempts, got %d", len(requests))
	}

	// A POST is not retried, since it may already have taken effect
	requests = nil
	if _, err := client.Post(server.URL, "text/plain", strings.NewReader("payload")); err == nil {
		t.Error("expected the POST to fail without a retry")
	}
	if len(requests) != 1 {
		t.Errorf("expected 1 attempt, got %d", len(requests))
	}
}

func TestRetryable(t *testing.T) {
	cases := []struct {
		method string
		status int
		want   bool
	}{
		{http.MethodPost, http.StatusTooManyRequests, true},
		{http.MethodPost, http.StatusServiceUnavailable, false},
		{http.MethodGet, http.StatusServiceUnavailable, true},
		{http.MethodPut, http.StatusServiceUnavailable, true},
		{http.MethodGet, http.StatusInternalServerError, false},
		{http.MethodGet, http.StatusOK, false},
	}
	for _, c := range cases {
		if got := retryable(c.method, c.status); got != c.want {
			t.Errorf("retryable(%s, %d) = %v, want %v", c.method, c.status, got, c.want)
		}
	}
}

// fatalRecorder stands in for a test so the must helpers can be checked
// without failing the real one
type fatalRecorder struct {
	testing.TB
	fatal []any
}

func (r *fatalRecorder) Helper() {}

func (r *fatalRecorder) Fatal(args ...any) { r.fatal = args }

func TestMust(t *testing.T) {
	boom := errors.New("boom")

	r := &fatalRecorder{}
	if got := mustReturn("value", nil)(r); got != "value" || r.fatal != nil {
		t.Errorf("expected mustReturn to pass the value through, got %q and %v", got, r.fatal)
	}
	r = &fatalRecorder{}
	if got := mustPage("page", &asana.NextPage{}, nil)(r); got != "page" || r.fatal != nil {
		t.Errorf("expected mustPage to pass the page through, got %q and %v", got, r.fatal)
	}

	for name, fail := range map[string]func(testing.TB){
		"must":       func(t testing.TB) { must(t, boom) },
		"mustReturn": func(t testing.TB) { mustReturn("value", boom)(t) },
		"mustPage":   func(t testing.TB) { mustPage("page", &asana.NextPage{}, boom)(t) },
	} {
		r := &fatalRecorder{}
		fail(r)
		if len(r.fatal) != 1 || r.fatal[0] != boom {
			t.Errorf("expected %s to stop the test with the error, got %v", name, r.fatal)
		}
	}
}
