package asana

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/h2non/gock"
)

func TestProjectBrief_Fields(t *testing.T) {
	if !slices.Contains(Fields(Project{}).Fields, "project_brief") {
		t.Errorf("Expected Fields(Project{}) to include %q", "project_brief")
	}

	briefFields := Fields(ProjectBrief{}).Fields
	for _, name := range []string{"gid", "title", "html_text", "text", "permalink_url", "project"} {
		if !slices.Contains(briefFields, name) {
			t.Errorf("Expected Fields(ProjectBrief{}) to include %q, got %v", name, briefFields)
		}
	}

	project := &Project{}
	if err := json.Unmarshal([]byte(`{
		"gid": "10",
		"name": "Roadmap",
		"project_brief": {"gid": "20", "resource_type": "project_brief"}
	}`), project); err != nil {
		t.Fatal(err)
	}
	if project.ProjectBrief == nil || project.ProjectBrief.GetID() != "20" {
		t.Errorf("ProjectBrief = %+v, want gid 20", project.ProjectBrief)
	}
}

func TestProjectBrief_Fetch(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/project_briefs/20").
		Reply(200).
		JSON(o{"data": o{
			"gid":           "20",
			"resource_type": "project_brief",
			"title":         "Q4 Goals",
			"html_text":     "<body>Ship v2</body>",
			"text":          "Ship v2",
			"permalink_url": "https://app.asana.com/0/10/20",
			"project":       o{"gid": "10", "resource_type": "project", "name": "Roadmap"},
		}})

	client := NewClient(http.DefaultClient)
	brief := &ProjectBrief{ID: "20"}
	if err := brief.Fetch(client); err != nil {
		t.Fatal(err)
	}

	if brief.Title != "Q4 Goals" || brief.HTMLText != "<body>Ship v2</body>" || brief.Text != "Ship v2" {
		t.Errorf("Unexpected brief content: %+v", brief)
	}
	if brief.PermalinkURL != "https://app.asana.com/0/10/20" {
		t.Errorf("PermalinkURL = %q, want %q", brief.PermalinkURL, "https://app.asana.com/0/10/20")
	}
	if brief.Project == nil || brief.Project.ID != "10" || brief.Project.Name != "Roadmap" {
		t.Errorf("Project = %+v, want gid 10 Roadmap", brief.Project)
	}
}

func TestProject_CreateProjectBrief(t *testing.T) {
	defer gock.Off()

	var sent map[string]any
	gock.New("https://app.asana.com").
		Post("/api/1.0/projects/10/project_briefs").
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
			"gid":           "21",
			"title":         "Overview",
			"html_text":     "<body>Hello</body>",
			"permalink_url": "https://app.asana.com/0/10/21",
			"project":       o{"gid": "10", "name": "Roadmap"},
		}})

	client := NewClient(http.DefaultClient)
	project := &Project{ID: "10"}
	brief, err := project.CreateProjectBrief(client, &ProjectBriefRequest{
		ProjectBriefBase: ProjectBriefBase{
			Title:    "Overview",
			HTMLText: "<body>Hello</body>",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"title":     "Overview",
		"html_text": "<body>Hello</body>",
	}
	if !maps.Equal(sent, want) {
		t.Errorf("POST project_briefs data = %v, want %v", sent, want)
	}
	if brief.ID != "21" || brief.Title != "Overview" || brief.Project == nil || brief.Project.ID != "10" {
		t.Errorf("Unexpected created brief: %+v", brief)
	}
}

func TestProjectBrief_Update(t *testing.T) {
	defer gock.Off()

	var sent map[string]any
	gock.New("https://app.asana.com").
		Put("/api/1.0/project_briefs/21").
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
			"gid":       "21",
			"title":     "Updated Overview",
			"html_text": "<body>Updated</body>",
		}})

	client := NewClient(http.DefaultClient)
	brief := &ProjectBrief{ID: "21"}
	err := brief.Update(client, &ProjectBriefRequest{
		ProjectBriefBase: ProjectBriefBase{
			Title:    "Updated Overview",
			HTMLText: "<body>Updated</body>",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]any{
		"title":     "Updated Overview",
		"html_text": "<body>Updated</body>",
	}
	if !maps.Equal(sent, want) {
		t.Errorf("PUT project_briefs/21 data = %v, want %v", sent, want)
	}
	if brief.Title != "Updated Overview" || brief.HTMLText != "<body>Updated</body>" {
		t.Errorf("Unexpected updated brief: %+v", brief)
	}
}

func TestProjectBrief_Delete(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Delete("/api/1.0/project_briefs/21").
		Reply(200).
		JSON(o{"data": o{}})

	client := NewClient(http.DefaultClient)
	brief := &ProjectBrief{ID: "21"}
	if err := brief.Delete(client); err != nil {
		t.Fatal(err)
	}
	if !gock.IsDone() {
		t.Error("Expected DELETE request to be made")
	}
}

func TestProjectBriefRejectsAmbiguousContent(t *testing.T) {
	client := NewClient(http.DefaultClient)
	request := &ProjectBriefRequest{ProjectBriefBase: ProjectBriefBase{Text: "plain", HTMLText: "<body>formatted</body>"}}
	if _, err := (&Project{ID: "p"}).CreateProjectBrief(client, request); err == nil {
		t.Fatal("create accepted both text representations")
	}
	if err := (&ProjectBrief{ID: "b"}).Update(client, request); err == nil {
		t.Fatal("update accepted both text representations")
	}
	if _, err := (&Project{ID: "p"}).CreateProjectBrief(client, nil); err == nil {
		t.Fatal("create accepted nil request")
	}
}
