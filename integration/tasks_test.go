package integration

import (
	"testing"
	"time"

	"github.com/kothar/asana-go"
)

func TestTaskLifecycle(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")

	section := mustReturn(p.CreateSection(f.client, &asana.SectionBase{Name: "Section"}))(t)

	due := asana.Date(time.Date(2030, time.January, 2, 0, 0, 0, 0, time.UTC))
	task := f.newTask(t, &asana.CreateTaskRequest{
		TaskBase: asana.TaskBase{
			Notes: "Created by the asana-go integration tests",
			DueOn: &due,
		},
		// Asana needs a workspace, parent or projects even when memberships
		// already name the project
		Workspace:   f.workspace.ID,
		Memberships: []*asana.CreateMembership{{Project: p.ID, Section: section.ID}},
	})

	fetched := &asana.Task{ID: task.ID}
	must(t, fetched.Fetch(f.client))
	if fetched.Name != task.Name || fetched.Notes != task.Notes {
		t.Errorf("expected name %q and notes %q, got %q and %q", task.Name, task.Notes, fetched.Name, fetched.Notes)
	}
	if fetched.DueOn == nil || time.Time(*fetched.DueOn).Format(time.DateOnly) != "2030-01-02" {
		t.Errorf("expected due date 2030-01-02, got %v", fetched.DueOn)
	}
	if len(fetched.Memberships) != 1 || fetched.Memberships[0].Section == nil || fetched.Memberships[0].Section.ID != section.ID {
		t.Errorf("expected the task to be in section %s, got %+v", section.ID, fetched.Memberships)
	}

	tasks := mustPage(p.Tasks(f.client))(t)
	if !contains(tasks, task.ID) {
		t.Error("expected the project's tasks to include the task")
	}

	tasks = mustPage(section.Tasks(f.client))(t)
	if !contains(tasks, task.ID) {
		t.Error("expected the section's tasks to include the task")
	}

	tasks = mustPage(f.client.QueryTasks(&asana.TaskQuery{Project: p.ID}))(t)
	if !contains(tasks, task.ID) {
		t.Error("expected QueryTasks by project to include the task")
	}

	newName := f.name(t, "renamed")
	must(t, task.Update(f.client, &asana.UpdateTaskRequest{
		TaskBase: asana.TaskBase{Name: newName, Completed: asana.Bool(true)},
	}))
	if task.Name != newName || !asana.IsTrue(task.Completed) {
		t.Errorf("expected Update to refresh the task, got name %q and completed %v", task.Name, task.Completed)
	}
	must(t, fetched.Fetch(f.client))
	if fetched.Name != newName || !asana.IsTrue(fetched.Completed) || fetched.CompletedAt == nil {
		t.Errorf("expected the task to be renamed and completed, got %q, %v, %v", fetched.Name, fetched.Completed, fetched.CompletedAt)
	}

	must(t, task.Delete(f.client))
	if err := fetched.Fetch(f.client); !asana.IsNotFoundError(err) {
		t.Errorf("expected a not found error fetching a deleted task, got %v", err)
	}
}

func TestTaskPagination(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")

	want := map[string]bool{}
	for range 3 {
		task := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})
		want[task.ID] = true
	}

	seen := map[string]bool{}
	pages := 0
	opts := &asana.Options{Limit: 1}
	for {
		tasks, next, err := p.Tasks(f.client, opts)
		must(t, err)
		pages++
		if len(tasks) > 1 {
			t.Fatalf("expected at most one task per page, got %d", len(tasks))
		}
		for _, task := range tasks {
			seen[task.ID] = true
		}
		if next == nil {
			break
		}
		if next.Offset == "" {
			t.Fatal("expected the next page to carry an offset")
		}
		opts = &asana.Options{Limit: 1, Offset: next.Offset}
	}

	if pages < 3 {
		t.Errorf("expected at least 3 pages, got %d", pages)
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("expected paging to return task %s", id)
		}
	}
}

func TestSubtasks(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")

	parent := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})

	sub := mustReturn(parent.CreateSubtask(f.client, &asana.Task{TaskBase: asana.TaskBase{Name: f.name(t, "subtask")}}))(t)
	cleanup(t, "subtask "+sub.ID, func() error { return sub.Delete(f.client) })

	fetched := &asana.Task{ID: sub.ID}
	must(t, fetched.Fetch(f.client))
	if fetched.Parent == nil || fetched.Parent.ID != parent.ID {
		t.Errorf("expected the subtask's parent to be %s, got %+v", parent.ID, fetched.Parent)
	}

	// Reparent a top-level task under the parent, after the existing subtask
	moved := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})
	must(t, moved.SetParent(f.client, &asana.SetParentRequest{
		Parent:      parent.ID,
		InsertAfter: sub.ID,
	}))

	subtasks := mustPage(parent.Subtasks(f.client))(t)
	if i, j := indexOf(subtasks, sub.ID), indexOf(subtasks, moved.ID); i < 0 || j < 0 || j < i {
		t.Errorf("expected both subtasks with the moved one last, found them at %d and %d", i, j)
	}
}

func TestTaskProjects(t *testing.T) {
	f := setup(t)
	home := f.newProject(t, "home")
	other := f.newProject(t, "other")

	task := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{home.ID}})

	must(t, task.AddProject(f.client, &asana.AddProjectRequest{Project: other.ID}))
	fetched := &asana.Task{ID: task.ID}
	must(t, fetched.Fetch(f.client))
	if !contains(fetched.Projects, home.ID) || !contains(fetched.Projects, other.ID) {
		t.Errorf("expected the task to be in both projects, got %d projects", len(fetched.Projects))
	}

	must(t, task.RemoveProject(f.client, other.ID))
	fetched = &asana.Task{ID: task.ID}
	must(t, fetched.Fetch(f.client))
	if contains(fetched.Projects, other.ID) {
		t.Error("expected the task to be removed from the other project")
	}
}

func TestTaskDependencies(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")

	blocked := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})
	blocker := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})
	follower := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})

	err := blocked.AddDependencies(f.client, &asana.AddDependenciesRequest{Dependencies: []string{blocker.ID}})
	skipIfPremiumOnly(t, err)
	must(t, err)
	must(t, blocker.AddDependents(f.client, &asana.AddDependentsRequest{Dependents: []string{follower.ID}}))

	fields := &asana.Options{Fields: []string{"dependencies", "dependents"}}
	fetched := &asana.Task{ID: blocker.ID}
	must(t, fetched.Fetch(f.client, fields))
	if !contains(fetched.Dependents, blocked.ID) || !contains(fetched.Dependents, follower.ID) {
		t.Errorf("expected the blocker to have both dependents, got %d", len(fetched.Dependents))
	}

	fetched = &asana.Task{ID: blocked.ID}
	must(t, fetched.Fetch(f.client, fields))
	if !contains(fetched.Dependencies, blocker.ID) {
		t.Errorf("expected the blocked task to depend on %s, got %d dependencies", blocker.ID, len(fetched.Dependencies))
	}
}
