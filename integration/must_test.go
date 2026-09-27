package integration

import "testing"

// These helpers take the test explicitly rather than holding it in package
// state, so they stay correct under t.Parallel. Go does not allow
// must(t, f()) when f returns more than one value, so the helpers that return
// a value take f's results first and the test in a second call:
//
//	tag := mustReturn(f.workspace.CreateTag(f.client, req))(t)

// must stops the test if err is not nil
func must(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// mustReturn returns v once the test has checked that err is nil
func mustReturn[T any](v T, err error) func(testing.TB) T {
	return func(t testing.TB) T {
		t.Helper()
		must(t, err)
		return v
	}
}

// mustPage returns a single page of results, dropping the pointer to the next
// page, once the test has checked that err is nil
func mustPage[T, P any](v T, _ P, err error) func(testing.TB) T {
	return mustReturn(v, err)
}
