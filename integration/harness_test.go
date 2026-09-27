package integration

// These tests check the suite's own plumbing and run without credentials

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	resp, err := client.Post(server.URL, "text/plain", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
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
