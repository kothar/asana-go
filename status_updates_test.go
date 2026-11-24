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

func TestStatusUpdate_Fields(t *testing.T) {
	if !slices.Contains(Fields(Project{}).Fields, "current_status_update") {
		t.Errorf("Expected Fields(Project{}) to include %q", "current_status_update")
	}

	fields := Fields(StatusUpdate{}).Fields
	for _, name := range []string{
		"gid", "resource_subtype", "title", "text", "html_text", "status_type",
		"author", "created_at", "created_by", "modified_at", "liked", "num_likes", "parent",
	} {
		if !slices.Contains(fields, name) {
			t.Errorf("Expected Fields(StatusUpdate{}) to include %q, got %v", name, fields)
		}
	}

	project := &Project{}
	if err := json.Unmarshal([]byte(`{
		"gid": "10",
		"name": "Launch",
		"current_status_update": {
			"gid": "90",
			"resource_type": "status_update",
			"resource_subtype": "project_status_update",
			"title": "Week 42"
		}
	}`), project); err != nil {
		t.Fatal(err)
	}
	if project.CurrentStatusUpdate == nil || project.CurrentStatusUpdate.GetID() != "90" ||
		project.CurrentStatusUpdate.Title != "Week 42" ||
		project.CurrentStatusUpdate.ResourceSubtype != "project_status_update" {
		t.Errorf("CurrentStatusUpdate = %+v, want gid 90 Week 42", project.CurrentStatusUpdate)
	}
}

func TestProject_StatusUpdates(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/status_updates").
		MatchParam("parent", "10").
		Reply(200).
		JSON(o{
			"data": []o{{
				"gid":              "90",
				"resource_type":    "status_update",
				"resource_subtype": "project_status_update",
				"title":            "Week 42",
				"text":             "Everything on track",
				"html_text":        "<body>Everything on track</body>",
				"status_type":      "on_track",
				"author":           o{"gid": "7", "name": "Alice"},
				"created_by":       o{"gid": "7", "name": "Alice"},
				"created_at":       "2026-11-20T12:00:00Z",
				"parent":           o{"gid": "10", "resource_type": "project", "name": "Launch"},
			}},
			"next_page": o{"offset": "tok123", "path": "/status_updates?offset=tok123", "uri": "https://app.asana.com/api/1.0/status_updates?offset=tok123"},
		})

	client := NewClient(http.DefaultClient)
	project := &Project{ID: "10"}
	updates, nextPage, err := project.StatusUpdates(client)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 {
		t.Fatalf("len(updates) = %d, want 1", len(updates))
	}
	u := updates[0]
	wantCreated := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	if u.ID != "90" || u.StatusType != StatusTypeOnTrack || u.Title != "Week 42" ||
		u.Author == nil || u.Author.ID != "7" || u.CreatedAt == nil || !u.CreatedAt.Equal(wantCreated) ||
		u.Parent == nil || u.Parent.ID != "10" || u.Parent.ResourceType != "project" {
		t.Errorf("Unexpected status update: %+v", u)
	}
	if nextPage == nil || nextPage.Offset != "tok123" {
		t.Errorf("nextPage = %+v, want offset tok123", nextPage)
	}
}

func TestClient_StatusUpdates_WithCreatedSinceAndValidation(t *testing.T) {
	defer gock.Off()

	client := NewClient(http.DefaultClient)
	if _, _, err := client.StatusUpdates(nil); err == nil {
		t.Fatal("Expected error for nil query")
	}
	if _, _, err := client.StatusUpdates(&StatusUpdateQuery{}); err == nil {
		t.Fatal("Expected error for missing parent")
	}

	gock.New("https://app.asana.com").
		Get("/api/1.0/status_updates").
		MatchParam("parent", "10").
		MatchParam("created_since", "2026-11-01T00:00:00Z").
		Reply(200).
		JSON(o{"data": []o{{"gid": "91", "status_type": "at_risk"}}})

	updates, _, err := client.StatusUpdates(&StatusUpdateQuery{
		Parent:       "10",
		CreatedSince: "2026-11-01T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 || updates[0].StatusType != StatusTypeAtRisk {
		t.Errorf("Unexpected updates: %+v", updates)
	}
}

func TestClient_CreateStatusUpdate(t *testing.T) {
	defer gock.Off()

	client := NewClient(http.DefaultClient)
	if _, err := client.CreateStatusUpdate(&CreateStatusUpdateRequest{
		StatusUpdateBase: StatusUpdateBase{Title: "Missing parent"},
	}); err == nil {
		t.Fatal("Expected validation error when Parent is empty")
	}

	var sent map[string]any
	gock.New("https://app.asana.com").
		Post("/api/1.0/status_updates").
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
			"gid":              "92",
			"resource_subtype": "project_status_update",
			"title":            "Week 43",
			"html_text":        "<body>Delayed by 2 days</body>",
			"status_type":      "off_track",
			"parent":           o{"gid": "10", "resource_type": "project", "name": "Launch"},
		}})

	created, err := client.CreateStatusUpdate(&CreateStatusUpdateRequest{
		StatusUpdateBase: StatusUpdateBase{
			Title:      "Week 43",
			HTMLText:   "<body>Delayed by 2 days</body>",
			StatusType: StatusTypeOffTrack,
		},
		Parent: "10",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"title":       "Week 43",
		"html_text":   "<body>Delayed by 2 days</body>",
		"status_type": "off_track",
		"parent":      "10",
	}
	if !maps.Equal(sent, want) {
		t.Errorf("POST /status_updates data = %v, want %v", sent, want)
	}
	if created.ID != "92" || created.StatusType != StatusTypeOffTrack || created.Parent == nil || created.Parent.ID != "10" {
		t.Errorf("Unexpected created status update: %+v", created)
	}
}

func TestStatusUpdate_FetchAndDelete(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/status_updates/92").
		Reply(200).
		JSON(o{"data": o{
			"gid":         "92",
			"title":       "Week 43",
			"status_type": "complete",
			"liked":       true,
			"num_likes":   3,
		}})

	gock.New("https://app.asana.com").
		Delete("/api/1.0/status_updates/92").
		Reply(200).
		JSON(o{"data": o{}})

	client := NewClient(http.DefaultClient)
	update := &StatusUpdate{ID: "92"}
	if err := update.Fetch(client); err != nil {
		t.Fatal(err)
	}
	if update.StatusType != StatusTypeComplete || !update.Liked || update.NumLikes != 3 {
		t.Errorf("Unexpected fetched update: %+v", update)
	}
	if err := update.Delete(client); err != nil {
		t.Fatal(err)
	}
	if !gock.IsDone() {
		t.Error("Expected GET and DELETE requests to be made")
	}
}
