package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/kothar/asana-go"
)

// TestCustomTypeFieldsOnStandardTask checks that requesting the custom type
// fields works on any plan, since callers like Ditto ask for every Task field
func TestCustomTypeFieldsOnStandardTask(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")
	task := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})

	fetched := &asana.Task{ID: task.ID}
	must(t, fetched.Fetch(f.client, asana.Fields(asana.Task{})))
	if fetched.CustomType != nil || fetched.CustomTypeStatusOption != nil {
		t.Errorf("expected a standard task to have no custom type, got %+v and %+v", fetched.CustomType, fetched.CustomTypeStatusOption)
	}

	types, _, err := p.CustomTypes(f.client)
	skipIfPremiumOnly(t, err)
	must(t, err)
	t.Logf("a new project has %d custom types", len(types))
}

// TestCustomTypeProbe records what the API allows for tasks with a custom
// type. Custom types can only be made and added to projects in the Asana UI,
// and need a paid plan, so it uses a project set up by hand. It looks for one
// in the scratch workspace, or uses the project named by:
//
//	ASANA_TEST_CUSTOM_TYPE_PROJECT  optional gid of a project that has at
//	                                least one custom type
//
// The test is skipped when no project with a custom type is found.
func TestCustomTypeProbe(t *testing.T) {
	f := setup(t)

	source, customType := f.findCustomType(t)
	if source == nil {
		t.Skip("no project with a user-created custom type; add one in the Asana UI or set ASANA_TEST_CUSTOM_TYPE_PROJECT")
	}
	status := customType.StatusOptions[len(customType.StatusOptions)-1]
	t.Logf("probing custom type %q in project %s %q with status %q (%s)",
		customType.Name, source.ID, source.Name, status.Name, status.CompletionState)

	setType := &asana.UpdateTaskRequest{
		TaskBase:               asana.TaskBase{ResourceSubtype: asana.ResourceSubtypeCustom},
		CustomType:             customType.ID,
		CustomTypeStatusOption: status.ID,
	}

	t.Run("create with the custom subtype", func(t *testing.T) {
		// This is what Ditto sent before it learned about custom types
		task, err := f.client.CreateTask(&asana.CreateTaskRequest{
			TaskBase: asana.TaskBase{Name: f.name(t, "task"), ResourceSubtype: asana.ResourceSubtypeCustom},
			Projects: []string{source.ID},
		})
		if err == nil {
			cleanup(t, "task "+task.ID, func() error { return task.Delete(f.client) })
			t.Errorf("expected Asana to refuse a new task with the custom subtype, but it made %s with type %+v", task.ID, task.CustomType)
			return
		}
		t.Logf("Asana refused the new task: %v", err)
	})

	t.Run("create with a custom type", func(t *testing.T) {
		// The refusal above asks for a custom_type, which CreateTaskRequest
		// doesn't carry, so send the request by hand
		body := map[string]any{"data": map[string]any{
			"name":                      f.name(t, "task"),
			"projects":                  []string{source.ID},
			"resource_subtype":          asana.ResourceSubtypeCustom,
			"custom_type":               customType.ID,
			"custom_type_status_option": status.ID,
		}}
		id, err := f.postTask(body)
		if err != nil {
			t.Logf("Asana refused a new task with a custom type: %v", err)
			return
		}
		task := &asana.Task{ID: id}
		cleanup(t, "task "+id, func() error { return task.Delete(f.client) })
		must(t, task.Fetch(f.client, asana.Fields(asana.Task{})))
		t.Logf("Asana created %s with type %+v and status %+v", id, task.CustomType, task.CustomTypeStatusOption)
	})

	t.Run("set the type in its project", func(t *testing.T) {
		task := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{source.ID}})
		must(t, task.Update(f.client, setType))

		fetched := &asana.Task{ID: task.ID}
		must(t, fetched.Fetch(f.client, asana.Fields(asana.Task{})))
		if fetched.ResourceSubtype != asana.ResourceSubtypeCustom || fetched.CustomType == nil || fetched.CustomType.ID != customType.ID {
			t.Errorf("expected the task to have custom type %s, got subtype %q and type %+v", customType.ID, fetched.ResourceSubtype, fetched.CustomType)
		}
		if fetched.CustomTypeStatusOption == nil || fetched.CustomTypeStatusOption.ID != status.ID {
			t.Errorf("expected the task to have status %s, got %+v", status.ID, fetched.CustomTypeStatusOption)
		}
		t.Logf("status %q (%s) left the task with completed=%v", status.Name, status.CompletionState, asana.IsTrue(fetched.Completed))
	})

	t.Run("set the type in a project without it", func(t *testing.T) {
		// A transfer creates new projects, which don't have the source
		// project's custom types. Whether Asana still accepts the type
		// decides whether Ditto can copy it.
		other := f.newProject(t, "project")
		otherTypes := mustPage(other.CustomTypes(f.client))(t)
		t.Logf("a new project has %d custom types", len(otherTypes))

		task := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{other.ID}})
		if err := task.Update(f.client, setType); err != nil {
			t.Logf("Asana refused the custom type in a project without it: %v", err)
			return
		}
		t.Logf("Asana accepted the custom type in a project without it: type %+v, status %+v", task.CustomType, task.CustomTypeStatusOption)
	})
}

// customTypeFields asks for the status options' details, which a plain
// "status_options" field leaves out
var customTypeFields = &asana.Options{Fields: []string{
	"name", "asana_created_type_identifier",
	"status_options.name", "status_options.completion_state", "status_options.enabled", "status_options.color",
}}

// findCustomType returns a project in the scratch workspace with a custom
// type that the API can assign, or the project named by
// ASANA_TEST_CUSTOM_TYPE_PROJECT. Projects made by the suite are ignored.
func (f *fixture) findCustomType(t *testing.T) (*asana.Project, *asana.CustomType) {
	t.Helper()

	var projects []*asana.Project
	if id := os.Getenv("ASANA_TEST_CUSTOM_TYPE_PROJECT"); id != "" {
		projects = []*asana.Project{{ID: id}}
	} else {
		all, err := f.workspace.AllProjects(f.client, &asana.Options{Fields: []string{"name"}})
		must(t, err)
		for _, p := range all {
			if !strings.HasPrefix(p.Name, namePrefix) {
				projects = append(projects, p)
			}
		}
	}

	for _, p := range projects {
		types, _, err := p.CustomTypes(f.client, customTypeFields)
		skipIfPremiumOnly(t, err)
		must(t, err)
		for _, ct := range types {
			// Some Asana-created types can't be assigned through the API
			if ct.AsanaCreatedTypeIdentifier == "" && len(ct.StatusOptions) > 0 {
				return p, ct
			}
		}
	}
	return nil, nil
}

// postTask creates a task from a raw request body, for fields the client
// doesn't model, and returns its gid
func (f *fixture) postTask(body any) (string, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, "https://app.asana.com/api/1.0/tasks", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("%d: %s", resp.StatusCode, content)
	}
	var result struct {
		Data struct {
			ID string `json:"gid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(content, &result); err != nil {
		return "", err
	}
	return result.Data.ID, nil
}
