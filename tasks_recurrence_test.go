package asana

import (
	"encoding/json"
	"testing"
)

func TestTask_UnmarshalRecurrence(t *testing.T) {
	task := &Task{}
	if err := json.Unmarshal([]byte(`
{
	"gid": "123",
	"name": "Weekly review",
	"recurrence": {
		"type": "weekly",
		"data": {
			"days_of_week": ["monday", "friday"],
			"ends_on": "2026-12-31"
		}
	}
}
`), task); err != nil {
		t.Fatal(err)
	}

	if task.Recurrence == nil {
		t.Fatal("expected recurrence to be populated")
	}

	if task.Recurrence.Type != "weekly" {
		t.Fatalf("expected recurrence type to be weekly, got %q", task.Recurrence.Type)
	}

	daysOfWeek, ok := task.Recurrence.Data["days_of_week"].([]interface{})
	if !ok {
		t.Fatalf("expected days_of_week array, got %T", task.Recurrence.Data["days_of_week"])
	}

	if len(daysOfWeek) != 2 || daysOfWeek[0] != "monday" || daysOfWeek[1] != "friday" {
		t.Fatalf("unexpected recurrence days_of_week payload: %#v", daysOfWeek)
	}

	if endsOn, ok := task.Recurrence.Data["ends_on"].(string); !ok || endsOn != "2026-12-31" {
		t.Fatalf("unexpected recurrence ends_on payload: %#v", task.Recurrence.Data["ends_on"])
	}
}

func TestCreateTaskRequest_MarshalRecurrence(t *testing.T) {
	request := &CreateTaskRequest{
		TaskBase: TaskBase{
			Name: "Weekly review",
			Recurrence: &Recurrence{
				Type: "weekly",
				Data: map[string]interface{}{
					"days_of_week": []string{"monday", "friday"},
					"ends_on":      "2026-12-31",
				},
			},
		},
		Workspace: "999",
	}

	bs, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(bs, &payload); err != nil {
		t.Fatal(err)
	}

	recurrence, ok := payload["recurrence"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected recurrence object, got %T", payload["recurrence"])
	}

	if recurrence["type"] != "weekly" {
		t.Fatalf("expected recurrence type to be weekly, got %#v", recurrence["type"])
	}

	data, ok := recurrence["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected recurrence data object, got %T", recurrence["data"])
	}

	daysOfWeek, ok := data["days_of_week"].([]interface{})
	if !ok {
		t.Fatalf("expected days_of_week array, got %T", data["days_of_week"])
	}

	if len(daysOfWeek) != 2 || daysOfWeek[0] != "monday" || daysOfWeek[1] != "friday" {
		t.Fatalf("unexpected recurrence days_of_week payload: %#v", daysOfWeek)
	}
}

func TestUpdateTaskRequest_MarshalRecurrence(t *testing.T) {
	request := &UpdateTaskRequest{
		TaskBase: TaskBase{
			Recurrence: &Recurrence{
				Type: "custom",
				Data: map[string]interface{}{
					"interval": 2,
				},
			},
		},
	}

	bs, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(bs, &payload); err != nil {
		t.Fatal(err)
	}

	recurrence, ok := payload["recurrence"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected recurrence object, got %T", payload["recurrence"])
	}

	if recurrence["type"] != "custom" {
		t.Fatalf("expected recurrence type to be custom, got %#v", recurrence["type"])
	}

	data, ok := recurrence["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected recurrence data object, got %T", recurrence["data"])
	}

	if interval, ok := data["interval"].(float64); !ok || interval != 2 {
		t.Fatalf("unexpected recurrence interval payload: %#v", data["interval"])
	}
}
