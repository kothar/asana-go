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
	if err := fetched.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
	if fetched.Name != text.Name || fetched.ResourceSubtype != asana.FieldTypeText {
		t.Errorf("expected text field %q, got %q of type %q", text.Name, fetched.Name, fetched.ResourceSubtype)
	}
	if fetched.IsAsanaCreated() {
		t.Error("expected a user-created field not to be reported as Asana-created")
	}

	fields, err := f.workspace.AllCustomFields(f.client)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(fields, text.ID, customFieldID) || !contains(fields, enum.ID, customFieldID) {
		t.Error("expected AllCustomFields to include both fields")
	}

	p := f.newProject(t, "project")
	for _, field := range []*asana.CustomField{text, enum} {
		setting, err := p.AddCustomFieldSetting(f.client, &asana.AddCustomFieldSettingRequest{CustomField: field.ID})
		if err != nil {
			t.Fatal(err)
		}
		if setting.CustomField == nil || setting.CustomField.ID != field.ID {
			t.Errorf("expected a setting for field %s, got %+v", field.ID, setting.CustomField)
		}
	}

	task := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})
	if err := task.Update(f.client, &asana.UpdateTaskRequest{
		CustomFields: map[string]interface{}{
			text.ID: "Some text",
			enum.ID: high.ID,
		},
	}); err != nil {
		t.Fatal(err)
	}

	fetchedTask := &asana.Task{ID: task.ID}
	if err := fetchedTask.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
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

	if err := p.RemoveCustomFieldSetting(f.client, text.ID); err != nil {
		t.Fatal(err)
	}
	fetchedProject := &asana.Project{ID: p.ID}
	if err := fetchedProject.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
	if contains(fetchedProject.CustomFieldSettings, text.ID, settingFieldID) {
		t.Error("expected the text field to be removed from the project")
	}
	if !contains(fetchedProject.CustomFieldSettings, enum.ID, settingFieldID) {
		t.Error("expected the enum field to stay on the project")
	}

	if err := text.Delete(f.client); err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
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
	if err := fetched.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
	i := indexOf(fetched.CustomFields, field.ID, fieldValueID)
	if i < 0 {
		t.Fatal("expected the task to carry the project-local field")
	}
	if v := fetched.CustomFields[i].NumberValue; v == nil || *v != 12.5 {
		t.Errorf("expected the number field to hold 12.5, got %v", v)
	}
}
