package asana

import (
	"net/http"
	"testing"

	"github.com/h2non/gock"
)

func TestWorkspace_MembershipFor(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/workspaces/1234/workspace_memberships").
		MatchParam("user", "me").
		Reply(200).
		JSON(o{"data": []o{{
			"gid":           "5678",
			"resource_type": "workspace_membership",
			"user":          o{"gid": "42", "resource_type": "user", "name": "Test User"},
			"workspace":     o{"gid": "1234", "resource_type": "workspace", "name": "Test"},
			"is_active":     true,
			"is_admin":      false,
			"is_guest":      true,
		}}})

	client := NewClient(http.DefaultClient)
	workspace := &Workspace{ID: "1234"}
	membership, err := workspace.MembershipFor(client, "me")
	if err != nil {
		t.Fatal(err)
	}
	if membership == nil {
		t.Fatal("Expected a membership")
	}
	if membership.ID != "5678" {
		t.Errorf("Expected membership ID 5678 but saw %s", membership.ID)
	}
	if !IsTrue(membership.IsGuest) {
		t.Error("Expected membership to be a guest")
	}
	if membership.User == nil || membership.User.ID != "42" {
		t.Errorf("Expected membership user 42 but saw %+v", membership.User)
	}
}

func TestWorkspace_MembershipFor_NotMember(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/workspaces/1234/workspace_memberships").
		MatchParam("user", "99").
		Reply(200).
		JSON(o{"data": []o{}})

	client := NewClient(http.DefaultClient)
	workspace := &Workspace{ID: "1234"}
	membership, err := workspace.MembershipFor(client, "99")
	if err != nil {
		t.Fatal(err)
	}
	if membership != nil {
		t.Errorf("Expected no membership but saw %+v", membership)
	}
}

func TestUser_WorkspaceMemberships(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/users/me/workspace_memberships").
		Reply(200).
		JSON(o{"data": []o{
			{"gid": "1", "resource_type": "workspace_membership"},
			{"gid": "2", "resource_type": "workspace_membership"},
		}})

	client := NewClient(http.DefaultClient)
	user := &User{ID: "me"}
	memberships, _, err := user.WorkspaceMemberships(client)
	if err != nil {
		t.Fatal(err)
	}
	if len(memberships) != 2 {
		t.Errorf("Expected 2 memberships but found %d", len(memberships))
	}
}

func TestWorkspace_MembershipFor_RequiresUser(t *testing.T) {
	defer gock.Off()
	gock.New("https://app.asana.com").
		Get("/api/1.0/workspaces/1234/workspace_memberships").
		Reply(200).
		JSON(o{"data": []o{{"gid": "5678"}}})

	client := NewClient(http.DefaultClient)
	workspace := &Workspace{ID: "1234"}
	if _, err := workspace.MembershipFor(client, ""); err == nil {
		t.Fatal("Expected an error for an empty user ID")
	}
	if gock.IsDone() {
		t.Error("Expected no request to be made")
	}
}
