package integration

import (
	"os"
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
	if len(types) != 0 {
		t.Errorf("expected a new project to have no custom types, got %d", len(types))
	}
}

// TestCustomTypeProbe records what the API allows for tasks with a custom
// type. Custom types can only be made and added to projects in the Asana UI,
// and need a paid plan, so it runs against a project set up by hand:
//
//	ASANA_TEST_CUSTOM_TYPE_PROJECT  gid of a project that has at least one
//	                                custom type, on a plan that includes them
//
// The test account must be able to create projects in that project's team.
// The probe's tasks and projects are deleted when it finishes, but they are
// outside the scratch workspace, so the sweeper won't find any it leaves
// behind.
func TestCustomTypeProbe(t *testing.T) {
	f := setup(t)
	projectID := os.Getenv("ASANA_TEST_CUSTOM_TYPE_PROJECT")
	if projectID == "" {
		t.Skip("set ASANA_TEST_CUSTOM_TYPE_PROJECT to probe custom types")
	}

	source := &asana.Project{ID: projectID}
	must(t, source.Fetch(f.client, &asana.Options{Fields: []string{"name", "team", "workspace"}}))

	types := mustPage(source.CustomTypes(f.client, asana.Fields(asana.CustomType{})))(t)
	var customType *asana.CustomType
	for _, ct := range types {
		if ct.AsanaCreatedTypeIdentifier == "" && len(ct.StatusOptions) > 0 {
			customType = ct
			break
		}
	}
	if customType == nil {
		t.Fatalf("project %s has no user-created custom type with status options (found %d types)", projectID, len(types))
	}
	status := customType.StatusOptions[len(customType.StatusOptions)-1]
	t.Logf("using custom type %q with status %q (%s)", customType.Name, status.Name, status.CompletionState)

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
		if source.Team == nil {
			t.Skip("the project has no team to create a second project in")
		}
		team := &asana.Team{ID: source.Team.ID}
		other, err := team.CreateProject(f.client, &asana.CreateProjectRequest{
			ProjectBase: asana.ProjectBase{Name: f.name(t, "project")},
		})
		must(t, err)
		cleanup(t, "project "+other.ID, func() error { return other.Delete(f.client) })

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
