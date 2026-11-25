package asana

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/h2non/gock"
)

func TestTimeTracking_ScopeSafeFields(t *testing.T) {
	if slices.Contains(Fields(Task{}).Fields, "actual_time_minutes") {
		t.Error("Expected Fields(Task{}) to exclude actual_time_minutes by default for scope safety")
	}

	task := &Task{}
	if err := json.Unmarshal([]byte(`{"gid":"10","actual_time_minutes":75.5}`), task); err != nil {
		t.Fatal(err)
	}
	if task.ActualTimeMinutes == nil || *task.ActualTimeMinutes != 75.5 {
		t.Errorf("ActualTimeMinutes = %v, want 75.5", task.ActualTimeMinutes)
	}

	nullTask := &Task{}
	if err := json.Unmarshal([]byte(`{"gid":"11","actual_time_minutes":null}`), nullTask); err != nil {
		t.Fatal(err)
	}
	if nullTask.ActualTimeMinutes != nil {
		t.Errorf("ActualTimeMinutes = %v, want nil", nullTask.ActualTimeMinutes)
	}

	entryFields := Fields(TimeTrackingEntry{}).Fields
	for _, want := range []string{
		"gid", "duration_minutes", "entered_on", "attributable_to", "assignee",
		"created_by", "created_at", "task", "approval_status", "billable_status", "description",
	} {
		if !slices.Contains(entryFields, want) {
			t.Errorf("Expected Fields(TimeTrackingEntry{}) to include %q, got %v", want, entryFields)
		}
	}
	if slices.Contains(entryFields, "categories") {
		t.Error("Expected Fields(TimeTrackingEntry{}) to exclude categories by default for scope safety")
	}

	entry := &TimeTrackingEntry{}
	if err := json.Unmarshal([]byte(`{
		"gid": "50",
		"categories": [{"gid": "800", "name": "Engineering", "color": "blue"}]
	}`), entry); err != nil {
		t.Fatal(err)
	}
	if len(entry.Categories) != 1 || entry.Categories[0].GetID() != "800" || entry.Categories[0].Name != "Engineering" {
		t.Errorf("Categories = %+v, want gid 800 Engineering", entry.Categories)
	}
}

func TestTask_TimeTrackingEntries(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/tasks/10/time_tracking_entries").
		Reply(200).
		JSON(o{
			"data": []o{{
				"gid":              "501",
				"duration_minutes": 45,
				"entered_on":       "2026-11-24",
				"attributable_to":  o{"gid": "20", "name": "Roadmap"},
				"assignee":         o{"gid": "7", "name": "Alice"},
				"created_by":       o{"gid": "8", "name": "Bob"},
				"created_at":       "2026-11-24T10:30:00Z",
				"task":             o{"gid": "10", "name": "Implement SDK"},
				"approval_status":  "APPROVED",
				"billable_status":  "billable",
				"description":      "API integration",
			}},
			"next_page": o{"offset": "page2"},
		})

	client := NewClient(http.DefaultClient)
	task := &Task{ID: "10"}
	entries, nextPage, err := task.TimeTrackingEntries(client)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	e := entries[0]
	wantDate := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)
	wantCreated := time.Date(2026, 11, 24, 10, 30, 0, 0, time.UTC)
	if e.GetID() != "501" || e.DurationMinutes != 45 ||
		e.EnteredOn == nil || !time.Time(*e.EnteredOn).Equal(wantDate) ||
		e.AttributableTo == nil || e.AttributableTo.ID != "20" ||
		e.Assignee == nil || e.Assignee.ID != "7" ||
		e.CreatedBy == nil || e.CreatedBy.ID != "8" ||
		e.CreatedAt == nil || !e.CreatedAt.Equal(wantCreated) ||
		e.Task == nil || e.Task.ID != "10" ||
		e.ApprovalStatus != TimeTrackingApprovalApproved ||
		e.BillableStatus != BillableStatusBillable ||
		e.Description != "API integration" {
		t.Errorf("Unexpected time tracking entry: %+v", e)
	}
	if nextPage == nil || nextPage.Offset != "page2" {
		t.Errorf("nextPage = %+v, want offset page2", nextPage)
	}
}

func TestTask_CreateTimeTrackingEntry(t *testing.T) {
	defer gock.Off()

	client := NewClient(http.DefaultClient)
	task := &Task{ID: "10"}

	if _, err := task.CreateTimeTrackingEntry(client, &CreateTimeTrackingEntryRequest{DurationMinutes: 0}); err == nil {
		t.Fatal("Expected validation error for DurationMinutes == 0")
	}

	enteredOn := Date(time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC))
	var sent map[string]any
	gock.New("https://app.asana.com").
		Post("/api/1.0/tasks/10/time_tracking_entries").
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
			"gid":              "502",
			"duration_minutes": 30,
			"entered_on":       "2026-11-24",
			"attributable_to":  o{"gid": "20"},
			"assignee":         o{"gid": "7"},
			"billable_status":  "nonBillable",
			"description":      "Pair debugging",
		}})

	entry, err := task.CreateTimeTrackingEntry(client, &CreateTimeTrackingEntryRequest{
		DurationMinutes: 30,
		EnteredOn:       &enteredOn,
		AttributableTo:  "20",
		Assignee:        "7",
		BillableStatus:  BillableStatusNonBillable,
		Description:     "Pair debugging",
		Categories:      []string{"800"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if sent["duration_minutes"] != float64(30) || sent["entered_on"] != "2026-11-24" ||
		sent["attributable_to"] != "20" || sent["assignee"] != "7" ||
		sent["billable_status"] != "nonBillable" || sent["description"] != "Pair debugging" {
		t.Errorf("Unexpected POST time_tracking_entries data: %v", sent)
	}
	cats, ok := sent["categories"].([]any)
	if !ok || len(cats) != 1 || cats[0] != "800" {
		t.Errorf("categories payload = %#v, want [800]", sent["categories"])
	}
	if entry.ID != "502" || entry.DurationMinutes != 30 || entry.Assignee == nil || entry.Assignee.ID != "7" {
		t.Errorf("Unexpected created entry: %+v", entry)
	}
}

func TestTimeTrackingEntry_FetchUpdateDelete(t *testing.T) {
	defer gock.Off()

	client := NewClient(http.DefaultClient)
	entry := &TimeTrackingEntry{ID: "502"}

	if err := entry.Update(client, &UpdateTimeTrackingEntryRequest{DurationMinutes: -5}); err == nil {
		t.Fatal("Expected validation error for negative DurationMinutes")
	}

	gock.New("https://app.asana.com").
		Get("/api/1.0/time_tracking_entries/502").
		Reply(200).
		JSON(o{"data": o{
			"gid":              "502",
			"duration_minutes": 30,
			"assignee":         o{"gid": "7"},
		}})

	var sent map[string]any
	gock.New("https://app.asana.com").
		Put("/api/1.0/time_tracking_entries/502").
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
			"gid":              "502",
			"duration_minutes": 60,
			"assignee":         o{"gid": "9"},
		}})

	gock.New("https://app.asana.com").
		Delete("/api/1.0/time_tracking_entries/502").
		Reply(200).
		JSON(o{"data": o{}})

	if err := entry.Fetch(client); err != nil {
		t.Fatal(err)
	}
	if entry.DurationMinutes != 30 || entry.Assignee == nil || entry.Assignee.ID != "7" {
		t.Errorf("Unexpected fetched entry: %+v", entry)
	}

	if err := entry.Update(client, &UpdateTimeTrackingEntryRequest{
		DurationMinutes: 60,
		Assignee:        "9",
	}); err != nil {
		t.Fatal(err)
	}
	if sent["duration_minutes"] != float64(60) || sent["assignee"] != "9" {
		t.Errorf("Unexpected PUT data: %v", sent)
	}
	if entry.DurationMinutes != 60 || entry.Assignee == nil || entry.Assignee.ID != "9" {
		t.Errorf("Unexpected updated entry: %+v", entry)
	}

	if err := entry.Delete(client); err != nil {
		t.Fatal(err)
	}
	if !gock.IsDone() {
		t.Error("Expected GET, PUT, and DELETE requests to be made")
	}
}
