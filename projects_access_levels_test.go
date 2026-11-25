package asana

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/h2non/gock"
)

func TestProject_AccessLevelFields(t *testing.T) {
	fields := Fields(Project{}).Fields
	for _, name := range []string{
		"default_access_level",
		"minimum_access_level_for_customization",
		"minimum_access_level_for_sharing",
	} {
		if !slices.Contains(fields, name) {
			t.Errorf("Expected Fields(Project{}) to include %q, got %v", name, fields)
		}
	}

	project := &Project{}
	if err := json.Unmarshal([]byte(`{
		"gid": "10",
		"default_access_level": "commenter",
		"minimum_access_level_for_customization": "admin",
		"minimum_access_level_for_sharing": "editor"
	}`), project); err != nil {
		t.Fatal(err)
	}

	if project.DefaultAccessLevel != AccessLevelCommenter {
		t.Errorf("DefaultAccessLevel = %q, want %q", project.DefaultAccessLevel, AccessLevelCommenter)
	}
	if project.MinimumAccessLevelForCustomization != AccessLevelAdmin {
		t.Errorf("MinimumAccessLevelForCustomization = %q, want %q", project.MinimumAccessLevelForCustomization, AccessLevelAdmin)
	}
	if project.MinimumAccessLevelForSharing != AccessLevelEditor {
		t.Errorf("MinimumAccessLevelForSharing = %q, want %q", project.MinimumAccessLevelForSharing, AccessLevelEditor)
	}
}

func TestProject_AccessLevelsSentOnCreateAndUpdate(t *testing.T) {
	defer gock.Off()

	// capture records the "data" object of a JSON request body.
	capture := func(sent *map[string]any) gock.MatchFunc {
		return func(req *http.Request, _ *gock.Request) (bool, error) {
			var body struct {
				Data map[string]any `json:"data"`
			}
			err := json.NewDecoder(req.Body).Decode(&body)
			*sent = body.Data
			return err == nil, err
		}
	}

	base := ProjectBase{
		Name:                               "Roadmap",
		DefaultAccessLevel:                 AccessLevelViewer,
		MinimumAccessLevelForCustomization: AccessLevelEditor,
		MinimumAccessLevelForSharing:       AccessLevelAdmin,
	}
	want := map[string]any{
		"name":                                   "Roadmap",
		"default_access_level":                   "viewer",
		"minimum_access_level_for_customization": "editor",
		"minimum_access_level_for_sharing":       "admin",
	}

	var created map[string]any
	gock.New("https://app.asana.com").
		Post("/api/1.0/projects").
		AddMatcher(capture(&created)).
		Reply(201).
		JSON(o{"data": o{"gid": "10", "name": "Roadmap", "default_access_level": "viewer"}})

	var updated map[string]any
	gock.New("https://app.asana.com").
		Put("/api/1.0/projects/10").
		AddMatcher(capture(&updated)).
		Reply(200).
		JSON(o{"data": o{"gid": "10", "name": "Roadmap", "minimum_access_level_for_sharing": "admin"}})

	client := NewClient(http.DefaultClient)
	project, err := client.CreateProject(&CreateProjectRequest{ProjectBase: base})
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(created, want) {
		t.Errorf("POST /projects data = %v, want %v", created, want)
	}
	if project.DefaultAccessLevel != AccessLevelViewer {
		t.Errorf("created DefaultAccessLevel = %q, want %q", project.DefaultAccessLevel, AccessLevelViewer)
	}

	if err := project.Update(client, &UpdateProjectRequest{ProjectBase: base}); err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(updated, want) {
		t.Errorf("PUT /projects/10 data = %v, want %v", updated, want)
	}
	if project.MinimumAccessLevelForSharing != AccessLevelAdmin {
		t.Errorf("updated MinimumAccessLevelForSharing = %q, want %q", project.MinimumAccessLevelForSharing, AccessLevelAdmin)
	}
}
