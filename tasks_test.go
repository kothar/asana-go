package asana

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/h2non/gock"
)

func TestTask_StartAtAndApprovalStatusFields(t *testing.T) {
	fields := Fields(Task{}).Fields
	for _, name := range []string{"start_at", "approval_status"} {
		if !slices.Contains(fields, name) {
			t.Errorf("Expected Fields(Task{}) to include %q, got %v", name, fields)
		}
	}

	task := &Task{}
	err := json.Unmarshal([]byte(`{
		"gid": "123",
		"name": "Design review",
		"resource_subtype": "approval",
		"approval_status": "changes_requested",
		"start_at": "2026-12-01T09:00:00Z",
		"due_at": "2026-12-01T17:00:00Z"
	}`), task)
	if err != nil {
		t.Fatal(err)
	}

	if task.ApprovalStatus != ApprovalStatusChangesRequested {
		t.Errorf("ApprovalStatus = %q, want %q", task.ApprovalStatus, ApprovalStatusChangesRequested)
	}
	wantStart := time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC)
	if task.StartAt == nil || !task.StartAt.Equal(wantStart) {
		t.Errorf("StartAt = %v, want %v", task.StartAt, wantStart)
	}
}

func TestCreateTaskRequest_ValidateStartAt(t *testing.T) {
	startDate := Date(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	dueDate := Date(time.Date(2026, 12, 2, 0, 0, 0, 0, time.UTC))
	startTime := time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC)
	dueTime := time.Date(2026, 12, 1, 17, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		req         CreateTaskRequest
		wantStartAt bool
		wantStartOn bool
		wantDueAt   bool
		wantDueOn   bool
	}{
		{
			name: "start_at with due_at clears start_on and due_on",
			req: CreateTaskRequest{
				TaskBase: TaskBase{
					StartAt: &startTime,
					StartOn: &startDate,
					DueAt:   &dueTime,
					DueOn:   &dueDate,
				},
			},
			wantStartAt: true,
			wantStartOn: false,
			wantDueAt:   true,
			wantDueOn:   false,
		},
		{
			name: "start_at without due_at clears start_at",
			req: CreateTaskRequest{
				TaskBase: TaskBase{
					StartAt: &startTime,
				},
			},
			wantStartAt: false,
			wantStartOn: false,
			wantDueAt:   false,
			wantDueOn:   false,
		},
		{
			name: "start_at with due_on only clears start_at and preserves start_on",
			req: CreateTaskRequest{
				TaskBase: TaskBase{
					StartAt: &startTime,
					StartOn: &startDate,
					DueOn:   &dueDate,
				},
			},
			wantStartAt: false,
			wantStartOn: true,
			wantDueAt:   false,
			wantDueOn:   true,
		},
		{
			name: "start_on with due_at keeps both",
			req: CreateTaskRequest{
				TaskBase: TaskBase{
					StartOn: &startDate,
					DueAt:   &dueTime,
				},
			},
			wantStartAt: false,
			wantStartOn: true,
			wantDueAt:   true,
			wantDueOn:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req
			if err := req.Validate(); err != nil {
				t.Fatalf("Validate() returned unexpected error: %v", err)
			}
			if (req.StartAt != nil) != tt.wantStartAt {
				t.Errorf("StartAt present = %v, want %v", req.StartAt != nil, tt.wantStartAt)
			}
			if (req.StartOn != nil) != tt.wantStartOn {
				t.Errorf("StartOn present = %v, want %v", req.StartOn != nil, tt.wantStartOn)
			}
			if (req.DueAt != nil) != tt.wantDueAt {
				t.Errorf("DueAt present = %v, want %v", req.DueAt != nil, tt.wantDueAt)
			}
			if (req.DueOn != nil) != tt.wantDueOn {
				t.Errorf("DueOn present = %v, want %v", req.DueOn != nil, tt.wantDueOn)
			}
		})
	}
}

func TestClient_CreateTask_StartAtAndApprovalStatus(t *testing.T) {
	defer gock.Off()

	startDate := Date(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	dueDate := Date(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	startTime := time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC)
	dueTime := time.Date(2026, 12, 1, 17, 0, 0, 0, time.UTC)

	var sent map[string]any
	gock.New("https://app.asana.com").
		Post("/api/1.0/tasks").
		AddMatcher(func(req *http.Request, _ *gock.Request) (bool, error) {
			var body struct {
				Data map[string]any `json:"data"`
			}
			err := json.NewDecoder(req.Body).Decode(&body)
			sent = body.Data
			return err == nil, err
		}).
		Reply(201).
		JSON(o{"data": o{
			"gid":              "42",
			"name":             "Approve launch",
			"resource_subtype": "approval",
			"approval_status":  "rejected",
			"start_at":         "2026-12-01T09:00:00Z",
			"due_at":           "2026-12-01T17:00:00Z",
		}})

	client := NewClient(http.DefaultClient)
	task, err := client.CreateTask(&CreateTaskRequest{
		TaskBase: TaskBase{
			Name:            "Approve launch",
			ResourceSubtype: "approval",
			ApprovalStatus:  ApprovalStatusRejected,
			StartOn:         &startDate,
			StartAt:         &startTime,
			DueOn:           &dueDate,
			DueAt:           &dueTime,
		},
		Workspace: "999",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"name":             "Approve launch",
		"resource_subtype": "approval",
		"approval_status":  "rejected",
		"start_at":         "2026-12-01T09:00:00Z",
		"due_at":           "2026-12-01T17:00:00Z",
		"workspace":        "999",
	}
	if !maps.Equal(sent, want) {
		t.Errorf("POST /tasks data = %v, want %v", sent, want)
	}
	if task.ID != "42" || task.ApprovalStatus != ApprovalStatusRejected || task.StartAt == nil || !task.StartAt.Equal(startTime) {
		t.Errorf("Unexpected created task: %+v", task)
	}
}

func TestTask_Update_StartAtAndApprovalStatus(t *testing.T) {
	defer gock.Off()

	startTime := time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC)
	dueTime := time.Date(2026, 12, 1, 18, 0, 0, 0, time.UTC)

	var sent map[string]any
	gock.New("https://app.asana.com").
		Put("/api/1.0/tasks/42").
		AddMatcher(func(req *http.Request, _ *gock.Request) (bool, error) {
			var body struct {
				Data map[string]any `json:"data"`
			}
			err := json.NewDecoder(req.Body).Decode(&body)
			sent = body.Data
			return err == nil, err
		}).
		Reply(200).
		JSON(o{"data": o{
			"gid":             "42",
			"approval_status": "approved",
			"start_at":        "2026-12-01T10:00:00Z",
			"due_at":          "2026-12-01T18:00:00Z",
		}})

	client := NewClient(http.DefaultClient)
	task := &Task{ID: "42"}
	err := task.Update(client, &UpdateTaskRequest{
		TaskBase: TaskBase{
			ApprovalStatus: ApprovalStatusApproved,
			StartAt:        &startTime,
			DueAt:          &dueTime,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"approval_status": "approved",
		"start_at":        "2026-12-01T10:00:00Z",
		"due_at":          "2026-12-01T18:00:00Z",
	}
	if !maps.Equal(sent, want) {
		t.Errorf("PUT /tasks/42 data = %v, want %v", sent, want)
	}
	if task.ApprovalStatus != ApprovalStatusApproved || task.StartAt == nil || !task.StartAt.Equal(startTime) {
		t.Errorf("Unexpected updated task: %+v", task)
	}
}
