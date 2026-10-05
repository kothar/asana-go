package asana

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/h2non/gock"
)

func TestProject_CustomTypes(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Get("/api/1.0/custom_types").
		MatchParam("project", "1234").
		Reply(200).
		JSON(o{"data": []o{{
			"gid":           "77",
			"resource_type": "custom_type",
			"name":          "Bug",
			"status_options": []o{
				{"gid": "1", "resource_type": "custom_type_status_option", "name": "Open", "completion_state": "Incomplete", "enabled": true, "color": "blue"},
				{"gid": "2", "resource_type": "custom_type_status_option", "name": "Fixed", "completion_state": "Complete", "enabled": true, "color": "green"},
			},
		}}})

	client := NewClient(http.DefaultClient)
	project := &Project{ID: "1234"}
	types, _, err := project.CustomTypes(client)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].ID != "77" || types[0].Name != "Bug" {
		t.Fatalf("Expected the Bug type but saw %+v", types)
	}
	options := types[0].StatusOptions
	if len(options) != 2 || options[1].Name != "Fixed" || options[1].CompletionState != "Complete" || !IsTrue(options[1].Enabled) {
		t.Errorf("Expected the Open and Fixed statuses but saw %+v", options)
	}
}

func TestTask_CustomTypeFields(t *testing.T) {
	fields := Fields(Task{}).Fields
	for _, name := range []string{"custom_type", "custom_type_status_option"} {
		if !slices.Contains(fields, name) {
			t.Errorf("Expected Fields(Task{}) to request %s", name)
		}
	}

	task := &Task{}
	err := json.Unmarshal([]byte(`{
		"gid": "9",
		"resource_subtype": "custom",
		"custom_type": {"gid": "77", "resource_type": "custom_type", "name": "Bug", "asana_created_type_identifier": null},
		"custom_type_status_option": {"gid": "2", "resource_type": "custom_type_status_option", "name": "Fixed"}
	}`), task)
	if err != nil {
		t.Fatal(err)
	}
	if task.ResourceSubtype != ResourceSubtypeCustom || task.CustomType == nil || task.CustomType.Name != "Bug" ||
		task.CustomTypeStatusOption == nil || task.CustomTypeStatusOption.ID != "2" {
		t.Errorf("Expected a Bug task with status 2 but saw %+v", task)
	}
}

func TestTask_UpdateCustomType(t *testing.T) {
	defer gock.Off()

	var sent map[string]any
	gock.New("https://app.asana.com").
		Put("/api/1.0/tasks/9").
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
			"gid":                       "9",
			"resource_subtype":          "custom",
			"custom_type":               o{"gid": "77", "name": "Bug"},
			"custom_type_status_option": o{"gid": "2", "name": "Fixed"},
		}})

	client := NewClient(http.DefaultClient)
	task := &Task{ID: "9"}
	err := task.Update(client, &UpdateTaskRequest{
		TaskBase:               TaskBase{ResourceSubtype: ResourceSubtypeCustom},
		CustomType:             "77",
		CustomTypeStatusOption: "2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.CustomType == nil || task.CustomType.ID != "77" {
		t.Errorf("Expected Update to load the custom type but saw %+v", task.CustomType)
	}
	want := map[string]any{"resource_subtype": "custom", "custom_type": "77", "custom_type_status_option": "2"}
	if !maps.Equal(sent, want) {
		t.Errorf("Expected the update to send %v but saw %v", want, sent)
	}
}
