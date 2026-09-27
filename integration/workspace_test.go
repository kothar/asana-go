package integration

import (
	"testing"

	"github.com/kothar/asana-go"
)

func TestCurrentUser(t *testing.T) {
	f := setup(t)

	me := mustReturn(f.client.CurrentUser())(t)
	if me.ID == "" || me.Name == "" {
		t.Errorf("expected the current user to have a gid and name, got %+v", me)
	}
	if !contains(me.Workspaces, f.workspace.ID) {
		t.Errorf("expected the current user's workspaces to include %s", f.workspace.ID)
	}

	user := &asana.User{ID: me.ID}
	must(t, user.Fetch(f.client))
	if user.Name != me.Name {
		t.Errorf("expected fetched user name %q, got %q", me.Name, user.Name)
	}
}

func TestWorkspaces(t *testing.T) {
	f := setup(t)

	workspaces := mustReturn(f.client.AllWorkspaces())(t)
	if !contains(workspaces, f.workspace.ID) {
		t.Errorf("expected AllWorkspaces to include %s", f.workspace.ID)
	}

	page := mustPage(f.client.Workspaces(&asana.Options{Limit: 1}))(t)
	if len(page) != 1 {
		t.Errorf("expected a page of one workspace, got %d", len(page))
	}

	w := &asana.Workspace{ID: f.workspace.ID}
	must(t, w.Fetch(f.client))
	if w.Name == "" {
		t.Error("expected the workspace to have a name")
	}
}

func TestWorkspaceUsers(t *testing.T) {
	f := setup(t)

	users := mustReturn(f.workspace.AllUsers(f.client))(t)
	if !contains(users, f.me.ID) {
		t.Errorf("expected the workspace users to include the current user %s", f.me.ID)
	}
}

func TestWorkspaceMemberships(t *testing.T) {
	f := setup(t)

	membership := mustReturn(f.workspace.MembershipFor(f.client, "me"))(t)
	if membership == nil {
		t.Fatalf("expected the current user to have a membership of workspace %s", f.workspace.ID)
	}
	if membership.User == nil || membership.User.ID != f.me.ID {
		t.Errorf("expected the membership to belong to %s, got %+v", f.me.ID, membership.User)
	}
	if membership.Workspace == nil || membership.Workspace.ID != f.workspace.ID {
		t.Errorf("expected the membership to be for workspace %s, got %+v", f.workspace.ID, membership.Workspace)
	}
	if !asana.IsTrue(membership.IsActive) {
		t.Error("expected the current user's membership to be active")
	}
	if membership.IsGuest == nil || *membership.IsGuest {
		t.Errorf("expected the current user to be a full member, not a guest (is_guest: %v)", membership.IsGuest)
	}

	byID := mustReturn(f.workspace.MembershipFor(f.client, f.me.ID))(t)
	if byID == nil || byID.ID != membership.ID {
		t.Errorf("expected looking up the user by gid to find membership %s, got %+v", membership.ID, byID)
	}

	fetched := &asana.WorkspaceMembership{ID: membership.ID}
	must(t, fetched.Fetch(f.client))
	if fetched.Workspace == nil || fetched.Workspace.ID != f.workspace.ID {
		t.Errorf("expected the fetched membership to be for workspace %s, got %+v", f.workspace.ID, fetched.Workspace)
	}

	mine := mustPage(f.me.WorkspaceMemberships(f.client, &asana.Options{Limit: 100}))(t)
	if !contains(mine, membership.ID) {
		t.Errorf("expected the current user's memberships to include %s", membership.ID)
	}

	opts := &asana.Options{Limit: 100}
	for {
		memberships, next, err := f.workspace.WorkspaceMemberships(f.client, opts)
		must(t, err)
		if contains(memberships, membership.ID) {
			return
		}
		if next == nil {
			t.Fatalf("expected the workspace memberships to include %s", membership.ID)
		}
		opts = &asana.Options{Limit: 100, Offset: next.Offset}
	}
}

func TestTeams(t *testing.T) {
	f := setup(t)
	if !f.workspace.IsOrganization {
		t.Skip("the workspace is not an organization, so it has no teams")
	}

	teams := mustReturn(f.workspace.AllTeams(f.client))(t)
	if len(teams) == 0 {
		t.Fatal("expected at least one team")
	}

	team := &asana.Team{ID: f.team.ID}
	must(t, team.Fetch(f.client))
	if team.Organization == nil || team.Organization.ID != f.workspace.ID {
		t.Errorf("expected the team to belong to %s, got %+v", f.workspace.ID, team.Organization)
	}
}

func TestUserTaskList(t *testing.T) {
	f := setup(t)

	list := mustReturn(f.me.TaskList(f.client, f.workspace))(t)
	if list.ID == "" {
		t.Fatal("expected My Tasks to have a gid")
	}

	fetched := &asana.UserTaskList{ID: list.ID}
	must(t, fetched.Fetch(f.client))
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
			if err != nil || contains(tasks, task.ID) {
				return err == nil, err
			}
			if next == nil {
				return false, nil
			}
			opts = &asana.Options{Limit: 100, Offset: next.Offset}
		}
	})

	tasks := mustPage(f.client.QueryTasks(&asana.TaskQuery{
		Assignee:       "me",
		Workspace:      f.workspace.ID,
		CompletedSince: "now",
	}, &asana.Options{Limit: 100}))(t)
	if !contains(tasks, task.ID) {
		t.Errorf("expected QueryTasks by assignee to include %s", task.ID)
	}
}
