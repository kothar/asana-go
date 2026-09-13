package asana

import (
	"net/http"
	"testing"

	"github.com/h2non/gock"
)

func TestUserTaskList_Fetch(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/user_task_lists/12345").
		Reply(200).
		JSON(o{"data": o{
			"gid":           "12345",
			"resource_type": "user_task_list",
			"name":          "My tasks in My Workspace",
			"owner":         o{"gid": "54321", "resource_type": "user", "name": "Greg Sanchez"},
			"workspace":     o{"gid": "31337", "resource_type": "workspace", "name": "My Workspace"},
		}})

	taskList := &UserTaskList{ID: "12345"}

	client := NewClient(http.DefaultClient)
	if err := taskList.Fetch(client); err != nil {
		t.Fatal(err)
	}

	if taskList.Name != "My tasks in My Workspace" {
		t.Errorf("Expected name %q but saw %q", "My tasks in My Workspace", taskList.Name)
	}
	if taskList.Owner == nil {
		t.Fatal("Expected an owner to be returned")
	}
	if taskList.Owner.ID != "54321" {
		t.Errorf("Expected owner ID 54321 but saw %s", taskList.Owner.ID)
	}
	if taskList.Workspace == nil {
		t.Fatal("Expected a workspace to be returned")
	}
	if taskList.Workspace.ID != "31337" {
		t.Errorf("Expected workspace ID 31337 but saw %s", taskList.Workspace.ID)
	}
	if taskList.Workspace.Name != "My Workspace" {
		t.Errorf("Expected workspace name %q but saw %q", "My Workspace", taskList.Workspace.Name)
	}
}

func TestUser_TaskList(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/users/54321/user_task_list").
		MatchParam("workspace", "31337").
		Reply(200).
		JSON(o{"data": o{
			"gid":           "12345",
			"resource_type": "user_task_list",
			"name":          "My tasks in My Workspace",
			"owner":         o{"gid": "54321", "resource_type": "user", "name": "Greg Sanchez"},
			"workspace":     o{"gid": "31337", "resource_type": "workspace", "name": "My Workspace"},
		}})

	user := &User{ID: "54321"}
	workspace := &Workspace{ID: "31337"}

	client := NewClient(http.DefaultClient)
	taskList, err := user.TaskList(client, workspace)
	if err != nil {
		t.Fatal(err)
	}

	if taskList.ID != "12345" {
		t.Errorf("Expected user task list ID 12345 but saw %s", taskList.ID)
	}
	if taskList.Workspace == nil || taskList.Workspace.ID != "31337" {
		t.Errorf("Expected the task list to reference workspace 31337")
	}
}

func TestUserTaskList_Tasks(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/user_task_lists/12345/tasks").
		Reply(200).
		JSON(o{"data": []o{{
			"gid":           "67890",
			"resource_type": "task",
			"name":          "Buy catnip",
		}}})

	taskList := &UserTaskList{ID: "12345"}

	client := NewClient(http.DefaultClient)
	tasks, _, err := taskList.Tasks(client)
	if err != nil {
		t.Fatal(err)
	}

	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task but found %d", len(tasks))
	}
	if tasks[0].ID != "67890" {
		t.Errorf("Expected task ID 67890 but saw %s", tasks[0].ID)
	}
}
