package integration

import (
	"testing"

	"github.com/kothar/asana-go"
)

// Custom fields are a paid feature. On a free workspace every custom field
// endpoint answers 402 Payment Required, which these tests treat as a skip.

func (f *fixture) newCustomField(t *testing.T, req *asana.CreateCustomFieldRequest) *asana.CustomField {
	t.Helper()

	req.Workspace = f.workspace.ID
	field, err := f.client.CreateCustomField(req)
	skipIfPremiumOnly(t, err)
	if err != nil {
		t.Fatalf("create custom field %q: %v", req.Name, err)
	}
	cleanup(t, "custom field "+field.ID, func() error { return field.Delete(f.client) })
	return field
}

func TestCustomFieldValues(t *testing.T) {
	f := setup(t)

	text := f.newCustomField(t, &asana.CreateCustomFieldRequest{
		CustomFieldBase: asana.CustomFieldBase{
			Name:            f.name(t, "text"),
			ResourceSubtype: asana.FieldTypeText,
		},
	})
	enum := f.newCustomField(t, &asana.CreateCustomFieldRequest{
		CustomFieldBase: asana.CustomFieldBase{
			Name:            f.name(t, "enum"),
			ResourceSubtype: asana.FieldTypeEnum,
		},
		EnumOptions: []*asana.EnumValueBase{{Name: "Low"}, {Name: "High"}},
	})
	if len(enum.EnumOptions) != 2 {
		t.Fatalf("expected the enum field to have 2 options, got %d", len(enum.EnumOptions))
	}
	high := enum.EnumOptions[1]

	fetched := &asana.CustomField{ID: text.ID}
	must(t, fetched.Fetch(f.client))
	if fetched.Name != text.Name || fetched.ResourceSubtype != asana.FieldTypeText {
		t.Errorf("expected text field %q, got %q of type %q", text.Name, fetched.Name, fetched.ResourceSubtype)
	}
	if fetched.IsAsanaCreated() {
		t.Error("expected a user-created field not to be reported as Asana-created")
	}

	fields := mustReturn(f.workspace.AllCustomFields(f.client))(t)
	if !contains(fields, text.ID) || !contains(fields, enum.ID) {
		t.Error("expected AllCustomFields to include both fields")
	}

	p := f.newProject(t, "project")
	for _, field := range []*asana.CustomField{text, enum} {
		setting := mustReturn(p.AddCustomFieldSetting(f.client, &asana.AddCustomFieldSettingRequest{CustomField: field.ID}))(t)
		if setting.CustomField == nil || setting.CustomField.ID != field.ID {
			t.Errorf("expected a setting for field %s, got %+v", field.ID, setting.CustomField)
		}
	}

	task := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})
	must(t, task.Update(f.client, &asana.UpdateTaskRequest{
		CustomFields: map[string]interface{}{
			text.ID: "Some text",
			enum.ID: high.ID,
		},
	}))

	fetchedTask := &asana.Task{ID: task.ID}
	must(t, fetchedTask.Fetch(f.client))
	values := map[string]*asana.CustomFieldValue{}
	for _, value := range fetchedTask.CustomFields {
		values[value.ID] = value
	}
	if v := values[text.ID]; v == nil || v.TextValue == nil || *v.TextValue != "Some text" {
		t.Errorf("expected the text field to hold %q, got %+v", "Some text", v)
	}
	if v := values[enum.ID]; v == nil || v.EnumValue == nil || v.EnumValue.ID != high.ID {
		t.Errorf("expected the enum field to hold option %s, got %+v", high.ID, v)
	}

	must(t, p.RemoveCustomFieldSetting(f.client, text.ID))
	fetchedProject := &asana.Project{ID: p.ID}
	must(t, fetchedProject.Fetch(f.client))
	if contains(settingFields(fetchedProject.CustomFieldSettings), text.ID) {
		t.Error("expected the text field to be removed from the project")
	}
	if !contains(settingFields(fetchedProject.CustomFieldSettings), enum.ID) {
		t.Error("expected the enum field to stay on the project")
	}

	must(t, text.Delete(f.client))
	if err := fetched.Fetch(f.client); !asana.IsNotFoundError(err) {
		t.Errorf("expected a not found error fetching a deleted custom field, got %v", err)
	}
}

func TestProjectLocalCustomField(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")

	setting, err := p.AddProjectLocalCustomField(f.client, &asana.AddProjectLocalCustomFieldRequest{
		CustomField: asana.ProjectLocalCustomField{
			CustomFieldBase: asana.CustomFieldBase{
				Name:            f.name(t, "number"),
				ResourceSubtype: asana.FieldTypeNumber,
				Precision:       new(2),
			},
		},
	})
	skipIfPremiumOnly(t, err)
	must(t, err)
	if setting.CustomField == nil || setting.CustomField.ID == "" {
		t.Fatalf("expected the setting to carry the new field, got %+v", setting.CustomField)
	}
	field := setting.CustomField
	cleanup(t, "custom field "+field.ID, func() error { return field.Delete(f.client) })

	if asana.IsTrue(field.IsGlobalToWorkspace) {
		t.Error("expected a project-local field not to be global to the workspace")
	}

	task := f.newTask(t, &asana.CreateTaskRequest{
		Projects:     []string{p.ID},
		CustomFields: map[string]interface{}{field.ID: 12.5},
	})
	fetched := &asana.Task{ID: task.ID}
	must(t, fetched.Fetch(f.client))
	i := indexOf(fetched.CustomFields, field.ID)
	if i < 0 {
		t.Fatal("expected the task to carry the project-local field")
	}
	if v := fetched.CustomFields[i].NumberValue; v == nil || *v != 12.5 {
		t.Errorf("expected the number field to hold 12.5, got %v", v)
	}
}
