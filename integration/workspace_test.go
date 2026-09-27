package integration

import (
	"testing"

	"github.com/kothar/asana-go"
)

func TestCurrentUser(t *testing.T) {
	f := setup(t)

	me, err := f.client.CurrentUser()
	if err != nil {
		t.Fatal(err)
	}
	if me.ID == "" || me.Name == "" {
		t.Errorf("expected the current user to have a gid and name, got %+v", me)
	}
	if !contains(me.Workspaces, f.workspace.ID, workspaceID) {
		t.Errorf("expected the current user's workspaces to include %s", f.workspace.ID)
	}

	user := &asana.User{ID: me.ID}
	if err := user.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
	if user.Name != me.Name {
		t.Errorf("expected fetched user name %q, got %q", me.Name, user.Name)
	}
}

func TestWorkspaces(t *testing.T) {
	f := setup(t)

	workspaces, err := f.client.AllWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	if !contains(workspaces, f.workspace.ID, workspaceID) {
		t.Errorf("expected AllWorkspaces to include %s", f.workspace.ID)
	}

	page, _, err := f.client.Workspaces(&asana.Options{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 {
		t.Errorf("expected a page of one workspace, got %d", len(page))
	}

	w := &asana.Workspace{ID: f.workspace.ID}
	if err := w.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
	if w.Name == "" {
		t.Error("expected the workspace to have a name")
	}
}

func TestWorkspaceUsers(t *testing.T) {
	f := setup(t)

	users, err := f.workspace.AllUsers(f.client)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(users, f.me.ID, userID) {
		t.Errorf("expected the workspace users to include the current user %s", f.me.ID)
	}
}

func TestTeams(t *testing.T) {
	f := setup(t)
	if !f.workspace.IsOrganization {
		t.Skip("the workspace is not an organization, so it has no teams")
	}

	teams, err := f.workspace.AllTeams(f.client)
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) == 0 {
		t.Fatal("expected at least one team")
	}

	team := &asana.Team{ID: f.team.ID}
	if err := team.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
	if team.Organization == nil || team.Organization.ID != f.workspace.ID {
		t.Errorf("expected the team to belong to %s, got %+v", f.workspace.ID, team.Organization)
	}
}

func TestUserTaskList(t *testing.T) {
	f := setup(t)

	list, err := f.me.TaskList(f.client, f.workspace)
	if err != nil {
		t.Fatal(err)
	}
	if list.ID == "" {
		t.Fatal("expected My Tasks to have a gid")
	}

	fetched := &asana.UserTaskList{ID: list.ID}
	if err := fetched.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
	if fetched.Owner == nil || fetched.Owner.ID != f.me.ID {
		t.Errorf("expected My Tasks to be owned by %s, got %+v", f.me.ID, fetched.Owner)
	}
	if fetched.Workspace == nil || fetched.Workspace.ID != f.workspace.ID {
		t.Errorf("expected My Tasks to be in workspace %s, got %+v", f.workspace.ID, fetched.Workspace)
	}

	task := f.newTask(t, &asana.CreateTaskRequest{
		Workspace: f.workspace.ID,
		Assignee:  f.me.ID,
	})

	eventually(t, "the assigned task to appear in My Tasks", func() (bool, error) {
		opts := &asana.Options{Limit: 100}
		for {
			tasks, next, err := list.Tasks(f.client, opts)
			if err != nil || contains(tasks, task.ID, taskID) {
				return err == nil, err
			}
			if next == nil {
				return false, nil
			}
			opts = &asana.Options{Limit: 100, Offset: next.Offset}
		}
	})

	tasks, _, err := f.client.QueryTasks(&asana.TaskQuery{
		Assignee:       "me",
		Workspace:      f.workspace.ID,
		CompletedSince: "now",
	}, &asana.Options{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(tasks, task.ID, taskID) {
		t.Errorf("expected QueryTasks by assignee to include %s", task.ID)
	}
}
