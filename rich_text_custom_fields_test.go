package asana

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/h2non/gock"
)

func TestCustomFieldValue_HTMLTextValue(t *testing.T) {
	if !slices.Contains(Fields(CustomFieldValue{}).Fields, "html_text_value") {
		t.Errorf("Expected Fields(CustomFieldValue{}) to include %q", "html_text_value")
	}

	// Nested custom field values are not expanded by Fields(Task{}), so the
	// task opt_fields list keeps requesting custom_fields as a whole and
	// never gains the request-only html_custom_fields.
	taskFields := Fields(Task{}).Fields
	if !slices.Contains(taskFields, "custom_fields") {
		t.Errorf("Expected Fields(Task{}) to include %q, got %v", "custom_fields", taskFields)
	}
	for _, name := range []string{"html_text_value", "html_custom_fields"} {
		if slices.Contains(taskFields, name) {
			t.Errorf("Fields(Task{}) should not include %q, got %v", name, taskFields)
		}
	}
	task := &Task{}
	if err := json.Unmarshal([]byte(`{
		"gid": "1",
		"custom_fields": [{
			"gid": "100",
			"type": "text",
			"text_value": "Bold",
			"html_text_value": "<body><strong>Bold</strong></body>"
		}]
	}`), task); err != nil {
		t.Fatal(err)
	}
	if len(task.CustomFields) != 1 {
		t.Fatalf("CustomFields = %+v, want 1 value", task.CustomFields)
	}
	value := task.CustomFields[0]
	if value.HTMLTextValue == nil || *value.HTMLTextValue != "<body><strong>Bold</strong></body>" {
		t.Errorf("HTMLTextValue = %v, want <body><strong>Bold</strong></body>", value.HTMLTextValue)
	}
	if value.TextValue == nil || *value.TextValue != "Bold" {
		t.Errorf("TextValue = %v, want Bold", value.TextValue)
	}
}

func TestHTMLCustomFieldsSentOnCreateAndUpdate(t *testing.T) {
	htmlFields := map[string]string{"100": "<body><em>Draft</em></body>"}

	tests := []struct {
		name   string
		method string
		path   string
		call   func(*Client) error
	}{
		{
			name:   "create task",
			method: http.MethodPost,
			path:   "/api/1.0/tasks",
			call: func(c *Client) error {
				_, err := c.CreateTask(&CreateTaskRequest{HTMLCustomFields: htmlFields})
				return err
			},
		},
		{
			name:   "update task",
			method: http.MethodPut,
			path:   "/api/1.0/tasks/1",
			call: func(c *Client) error {
				return (&Task{ID: "1"}).Update(c, &UpdateTaskRequest{HTMLCustomFields: htmlFields})
			},
		},
		{
			name:   "create project",
			method: http.MethodPost,
			path:   "/api/1.0/projects",
			call: func(c *Client) error {
				_, err := c.CreateProject(&CreateProjectRequest{HTMLCustomFields: htmlFields})
				return err
			},
		},
		{
			name:   "update project",
			method: http.MethodPut,
			path:   "/api/1.0/projects/10",
			call: func(c *Client) error {
				return (&Project{ID: "10"}).Update(c, &UpdateProjectRequest{HTMLCustomFields: htmlFields})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer gock.Off()

			var sent map[string]any
			request := gock.New("https://app.asana.com")
			request.Method = tc.method
			request.Path(tc.path).
				AddMatcher(func(req *http.Request, _ *gock.Request) (bool, error) {
					var body struct {
						Data map[string]any `json:"data"`
					}
					err := json.NewDecoder(req.Body).Decode(&body)
					sent = body.Data
					return err == nil, err
				}).
				Reply(200).
				JSON(o{"data": o{"gid": "1"}})

			if err := tc.call(NewClient(http.DefaultClient)); err != nil {
				t.Fatal(err)
			}
			if !gock.IsDone() {
				t.Fatalf("Expected %s %s", tc.method, tc.path)
			}

			want := map[string]any{"html_custom_fields": map[string]any{"100": "<body><em>Draft</em></body>"}}
			if !reflect.DeepEqual(sent, want) {
				t.Errorf("%s %s data = %v, want %v", tc.method, tc.path, sent, want)
			}
		})
	}
}
