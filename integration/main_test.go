// Package integration exercises the client against the live Asana API.
//
// The tests need a personal access token for a scratch workspace, where they
// create and delete their own projects, tasks, tags and custom fields:
//
//	ASANA_TEST_PAT        personal access token for the test account
//	ASANA_TEST_WORKSPACE  gid of the scratch workspace to write to
//	ASANA_TEST_TEAM       optional gid of the team to create projects in, when
//	                      the workspace is an organization (defaults to the
//	                      first team the account can see)
//	ASANA_TEST_CUSTOM_TYPE_PROJECT
//	                      optional gid of a project with a custom task type,
//	                      for TestCustomTypeProbe (which otherwise looks for
//	                      one in the scratch workspace)
//
// When ASANA_TEST_PAT or ASANA_TEST_WORKSPACE is unset every test is skipped,
// so `go test ./...` stays offline.
//
// Everything a run creates is named with a shared prefix and deleted when its
// test finishes. Anything left behind by an interrupted run is swept up by the
// next run once it is more than an hour old.
package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kothar/asana-go"
	"github.com/rs/xid"
	"golang.org/x/oauth2"
)

const (
	// namePrefix marks every object the suite creates, so that the sweeper
	// only ever touches its own leftovers
	namePrefix = "asana-go-it"

	// staleAfter is how old a leftover object must be before the sweeper
	// removes it. It is long enough that a concurrent run never loses the
	// objects it is still using.
	staleAfter = time.Hour
)

type fixture struct {
	client    *asana.Client
	token     string
	workspace *asana.Workspace
	team      *asana.Team
	me        *asana.User
	runID     string
}

var (
	fx         *fixture
	skipReason string
)

func TestMain(m *testing.M) {
	token := os.Getenv("ASANA_TEST_PAT")
	workspaceID := os.Getenv("ASANA_TEST_WORKSPACE")
	if token == "" || workspaceID == "" {
		skipReason = "set ASANA_TEST_PAT and ASANA_TEST_WORKSPACE to run the Asana integration tests"
		os.Exit(m.Run())
	}

	f, err := newFixture(token, workspaceID, os.Getenv("ASANA_TEST_TEAM"))
	if err != nil {
		log.Fatalf("integration setup: %v", err)
	}
	fx = f

	f.sweep()

	os.Exit(m.Run())
}

func newFixture(token, workspaceID, teamID string) (*fixture, error) {
	f := &fixture{
		client: newClient(token),
		token:  token,
		runID:  fmt.Sprintf("%s %s", namePrefix, xid.New()),
	}

	me, err := f.client.CurrentUser()
	if err != nil {
		return nil, fmt.Errorf("load current user: %w", err)
	}
	f.me = me

	// Refuse to run against a workspace the token can't see, rather than
	// finding out halfway through a test
	for _, w := range me.Workspaces {
		if w.ID == workspaceID {
			f.workspace = w
		}
	}
	if f.workspace == nil {
		return nil, fmt.Errorf("workspace %s is not visible to the test account", workspaceID)
	}
	if err := f.workspace.Fetch(f.client); err != nil {
		return nil, fmt.Errorf("load workspace: %w", err)
	}

	// Projects in an organization have to belong to a team
	if f.workspace.IsOrganization {
		if teamID == "" {
			teams, _, err := f.workspace.Teams(f.client, &asana.Options{Limit: 1})
			if err != nil {
				return nil, fmt.Errorf("list teams: %w", err)
			}
			if len(teams) == 0 {
				return nil, errors.New("the workspace is an organization with no visible teams; set ASANA_TEST_TEAM")
			}
			teamID = teams[0].ID
		}
		f.team = &asana.Team{ID: teamID}
		if err := f.team.Fetch(f.client); err != nil {
			return nil, fmt.Errorf("load team %s: %w", teamID, err)
		}
	}

	return f, nil
}

// responseTimeout bounds how long one attempt waits for Asana to start
// answering. A request that hangs then fails on its own and can be retried,
// rather than using up the deadline for every attempt.
const responseTimeout = 20 * time.Second

// newClient builds a client that authenticates with the token and waits out
// rate limits, so that a burst of test requests doesn't fail the run
func newClient(token string) *asana.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.ResponseHeaderTimeout = responseTimeout

	return asana.NewClient(&http.Client{
		// The client's deadline covers every attempt, so it leaves room for
		// retryTransport to try a hung request again
		Timeout: 3 * time.Minute,
		Transport: &oauth2.Transport{
			Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
			Base:   &retryTransport{base: base},
		},
	})
}

// retryTransport retries requests that Asana rejects as rate limited or
// temporarily unavailable, honouring the Retry-After header. It also retries
// idempotent requests whose connection fails, such as by being reset, or that
// get no response in time: a test run makes enough requests that such network
// blips come up now and then.
type retryTransport struct {
	base http.RoundTripper
}

func (r *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	const attempts = 5

	for attempt := 1; ; attempt++ {
		resp, err := r.base.RoundTrip(req)
		if err == nil && attempt > 1 && req.Method == http.MethodDelete && resp.StatusCode == http.StatusNotFound {
			// An earlier attempt deleted the object before failing, such as
			// with a 503 from a proxy, so the delete has done its job
			_ = resp.Body.Close()
			log.Printf("%s %s returned 404 on retry, so an earlier attempt deleted it", req.Method, req.URL.Path)
			return deleted(req), nil
		}
		if attempt == attempts {
			return resp, err
		}
		// A body that can't be replayed (a streamed upload) can't be retried
		if req.Body != nil && req.GetBody == nil {
			return resp, err
		}

		if err != nil {
			// A POST may have taken effect before the connection failed, so
			// retrying it could create a duplicate
			if !idempotent(req.Method) || req.Context().Err() != nil {
				return resp, err
			}
			log.Printf("%s %s failed, retrying: %v", req.Method, req.URL.Path, err)
		} else if !retryable(req.Method, resp.StatusCode) {
			return resp, nil
		}

		wait := time.Duration(attempt) * 2 * time.Second
		if resp != nil {
			if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
				wait = time.Duration(seconds) * time.Second
			}
			_ = resp.Body.Close()
			log.Printf("%s %s returned %d, retrying in %s", req.Method, req.URL.Path, resp.StatusCode, wait)
		}

		select {
		case <-time.After(wait):
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}

		next := req.Clone(req.Context())
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			next.Body = body
		}
		req = next
	}
}

// deleted is the response Asana gives for a successful delete
func deleted(req *http.Request) *http.Response {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"data":{}}`)),
		Request:    req,
	}
}

// retryable reports whether a response means the request can safely be sent
// again. A 429 means Asana refused the request, so even a POST can be retried.
// A 503 may come from a proxy after Asana acted on the request, so only
// idempotent requests are retried.
func retryable(method string, status int) bool {
	switch status {
	case http.StatusTooManyRequests:
		return true
	case http.StatusServiceUnavailable:
		return idempotent(method)
	}
	return false
}

// idempotent reports whether repeating a request with this method has the
// same effect as making it once
func idempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete:
		return true
	}
	return false
}

// setup returns the shared fixture, or skips the test when no credentials
// were provided
func setup(t *testing.T) *fixture {
	t.Helper()
	if fx == nil {
		t.Skip(skipReason)
	}
	return fx
}

// name returns a unique name for an object created by the test
func (f *fixture) name(t *testing.T, kind string) string {
	return fmt.Sprintf("%s %s %s", f.runID, t.Name(), kind)
}

// cleanup registers a deletion to run when the test finishes. An object that
// is already gone (because the test deleted it) is not an error.
func cleanup(t *testing.T, what string, del func() error) {
	t.Helper()
	t.Cleanup(func() {
		if err := del(); err != nil && !asana.IsNotFoundError(err) {
			t.Errorf("clean up %s: %v", what, err)
		}
	})
}

func (f *fixture) newProject(t *testing.T, kind string) *asana.Project {
	t.Helper()

	req := &asana.CreateProjectRequest{
		ProjectBase: asana.ProjectBase{
			Name:  f.name(t, kind),
			Notes: "Created by the asana-go integration tests",
		},
	}

	var p *asana.Project
	var err error
	if f.team != nil {
		p, err = f.team.CreateProject(f.client, req)
	} else {
		req.Workspace = f.workspace.ID
		p, err = f.client.CreateProject(req)
	}
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	cleanup(t, "project "+p.ID, func() error { return p.Delete(f.client) })
	return p
}

func (f *fixture) newTask(t *testing.T, req *asana.CreateTaskRequest) *asana.Task {
	t.Helper()

	if req.Name == "" {
		req.Name = f.name(t, "task")
	}
	task, err := f.client.CreateTask(req)
	if err != nil {
		t.Fatalf("create task %q: %v", req.Name, err)
	}
	cleanup(t, "task "+task.ID, func() error { return task.Delete(f.client) })
	return task
}

// skipIfPremiumOnly skips the test when Asana refuses a feature because the
// workspace is on the free plan
func skipIfPremiumOnly(t *testing.T, err error) {
	t.Helper()
	if isPaymentRequired(err) {
		t.Skipf("the workspace's plan doesn't include this feature: %v", err)
	}
}

// sweep deletes objects left behind by earlier runs that were interrupted
// before their cleanups ran. Leftovers are recognised by the timestamp in the
// run ID that starts their names. It is best effort: failures are only logged.
func (f *fixture) sweep() {
	cutoff := time.Now().Add(-staleAfter)
	stale := func(name string) bool {
		created, ok := runTime(name)
		return ok && created.Before(cutoff)
	}
	names := &asana.Options{Fields: []string{"name"}}

	projects, err := f.workspace.AllProjects(f.client, names)
	if err != nil {
		log.Printf("sweep: list projects: %v", err)
	}
	for _, p := range projects {
		if stale(p.Name) {
			log.Printf("sweep: deleting project %s %q", p.ID, p.Name)
			if err := p.Delete(f.client); err != nil {
				log.Printf("sweep: %v", err)
			}
		}
	}

	tags, err := f.workspace.AllTags(f.client, names)
	if err != nil {
		log.Printf("sweep: list tags: %v", err)
	}
	for _, tag := range tags {
		if stale(tag.Name) {
			log.Printf("sweep: deleting tag %s %q", tag.ID, tag.Name)
			if err := tag.Delete(f.client); err != nil {
				log.Printf("sweep: %v", err)
			}
		}
	}

	// Free workspaces refuse to list custom fields at all
	fields, err := f.workspace.AllCustomFields(f.client, names)
	if err != nil && !isPaymentRequired(err) {
		log.Printf("sweep: list custom fields: %v", err)
	}
	for _, field := range fields {
		if stale(field.Name) {
			log.Printf("sweep: deleting custom field %s %q", field.ID, field.Name)
			if err := field.Delete(f.client); err != nil {
				log.Printf("sweep: %v", err)
			}
		}
	}
}

// runTime recovers the creation time of the run that named an object
func runTime(name string) (time.Time, bool) {
	rest, ok := strings.CutPrefix(name, namePrefix+" ")
	if !ok {
		return time.Time{}, false
	}
	id, _, _ := strings.Cut(rest, " ")
	parsed, err := xid.FromString(id)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.Time(), true
}

func isPaymentRequired(err error) bool {
	e, ok := asana.IsAsanaError(err)
	return ok && e.StatusCode == http.StatusPaymentRequired
}

// contains reports whether a list holds the object with the given gid
func contains[T asana.Identifiable](items []T, id string) bool {
	return slices.ContainsFunc(items, hasID[T](id))
}

// indexOf returns the position of the object with the given gid in a list,
// or -1 if it is missing
func indexOf[T asana.Identifiable](items []T, id string) int {
	return slices.IndexFunc(items, hasID[T](id))
}

func hasID[T asana.Identifiable](id string) func(T) bool {
	return func(item T) bool { return item.GetID() == id }
}

// settingFields returns the custom fields that a project's settings attach
func settingFields(settings []*asana.CustomFieldSetting) []*asana.CustomField {
	var fields []*asana.CustomField
	for _, s := range settings {
		if s.CustomField != nil {
			fields = append(fields, s.CustomField)
		}
	}
	return fields
}

// members returns the users and teams that hold a set of memberships
func members(memberships []*asana.ProjectMembership) []*asana.ProjectMember {
	var result []*asana.ProjectMember
	for _, m := range memberships {
		if m.Member != nil {
			result = append(result, m.Member)
		}
	}
	return result
}

// eventually retries a check for a short while, for reads that lag behind a
// write
func eventually(t *testing.T, what string, check func() (bool, error)) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		ok, err := check()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		if ok {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s", what)
		case <-time.After(2 * time.Second):
		}
	}
}
