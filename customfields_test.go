package asana

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/h2non/gock"
)

func TestCustomFieldBase_Precision_ParseZero(t *testing.T) {
	cf := &CustomFieldBase{}
	if err := json.Unmarshal([]byte(`
{
	"precision": 0
}
`), cf); err != nil {
		t.Fatal(err)
	}

	if cf.Precision == nil || *cf.Precision != 0 {
		t.Errorf("Expected Precision to be a pointer to the integer zero, but saw %v", cf.Precision)
	}
}

func TestCustomFieldBase_Precision_ParseMissing(t *testing.T) {
	cf := &CustomFieldBase{}
	if err := json.Unmarshal([]byte(`
{
	"name": "name"
}
`), cf); err != nil {
		t.Fatal(err)
	}

	if cf.Precision != nil {
		t.Errorf("Expected Precision to be a nil, but saw %v", cf.Precision)
	}
}

func TestCustomFieldBase_Precision_SerializeZero(t *testing.T) {
	val := 0
	cf := &CustomFieldBase{Precision: &val}
	if bs, err := json.Marshal(cf); err != nil {
		t.Fatal(err)
	} else {
		if string(bs) != `{"precision":0,"resource_subtype":""}` {
			t.Errorf("Expected Precision to be a zero, but saw %v", string(bs))
		}
	}

}

func TestCustomField_AsanaCreatedField_Parse(t *testing.T) {
	cf := &CustomField{}
	if err := json.Unmarshal([]byte(`
{
	"gid": "123",
	"name": "Deal size",
	"resource_subtype": "number",
	"asana_created_field": "deal_size"
}
`), cf); err != nil {
		t.Fatal(err)
	}

	if cf.AsanaCreatedField != "deal_size" {
		t.Errorf("Expected AsanaCreatedField to be %q, but saw %q", "deal_size", cf.AsanaCreatedField)
	}
	if !cf.IsAsanaCreated() {
		t.Errorf("Expected IsAsanaCreated to be true for an Asana-created field")
	}
}

func TestCustomField_AsanaCreatedField_Absent(t *testing.T) {
	cf := &CustomField{}
	if err := json.Unmarshal([]byte(`
{
	"gid": "123",
	"name": "Effort",
	"resource_subtype": "number"
}
`), cf); err != nil {
		t.Fatal(err)
	}

	if cf.AsanaCreatedField != "" {
		t.Errorf("Expected AsanaCreatedField to be empty, but saw %q", cf.AsanaCreatedField)
	}
	if cf.IsAsanaCreated() {
		t.Errorf("Expected IsAsanaCreated to be false for a user-created field")
	}
}

// TestCustomField_AsanaCreatedField_RequestedInFields ensures the read-only
// asana_created_field property is included in opt_fields when requesting all
// fields for a CustomField, so it is populated on reads.
func TestCustomField_AsanaCreatedField_RequestedInFields(t *testing.T) {
	options := Fields(CustomField{})
	for _, name := range options.Fields {
		if name == "asana_created_field" {
			return
		}
	}
	t.Errorf("Expected Fields(CustomField{}) to include %q, got %v", "asana_created_field", options.Fields)
}

func TestCustomField_Delete(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Delete("/api/1.0/custom_fields/1234").
		Reply(200).
		JSON(map[string]any{"data": map[string]any{}})

	client := NewClient(http.DefaultClient)
	field := &CustomField{ID: "1234"}
	if err := field.Delete(client); err != nil {
		t.Fatal(err)
	}
	if !gock.IsDone() {
		t.Error("Expected DELETE request to be made")
	}
}

func TestCustomField_ReferenceFields(t *testing.T) {
	if !slices.Contains(Fields(CustomField{}).Fields, "input_restrictions") {
		t.Errorf("Expected Fields(CustomField{}) to include %q", "input_restrictions")
	}
	valueFields := Fields(CustomFieldValue{}).Fields
	for _, name := range []string{"input_restrictions", "reference_value"} {
		if !slices.Contains(valueFields, name) {
			t.Errorf("Expected Fields(CustomFieldValue{}) to include %q", name)
		}
	}

	cfv := &CustomFieldValue{}
	err := json.Unmarshal([]byte(`{
		"gid": "500",
		"name": "Related Work",
		"resource_subtype": "reference",
		"input_restrictions": ["task", "project", "portfolio"],
		"reference_value": [
			{"gid": "101", "resource_type": "task", "name": "Build API"},
			{"gid": "202", "resource_type": "project", "name": "Q4 Launch"},
			{"gid": "303", "resource_type": "portfolio", "name": "Platform"},
			{"gid": "404", "resource_type": "goal", "name": "Ship v2"},
			{"gid": "505", "resource_type": "team", "name": "Core"},
			{"gid": "606", "resource_type": "record", "name": "Account A"}
		]
	}`), cfv)
	if err != nil {
		t.Fatal(err)
	}

	if cfv.ResourceSubtype != FieldTypeReference {
		t.Errorf("ResourceSubtype = %q, want %q", cfv.ResourceSubtype, FieldTypeReference)
	}
	wantRestrictions := []string{"task", "project", "portfolio"}
	if !slices.Equal(cfv.InputRestrictions, wantRestrictions) {
		t.Errorf("InputRestrictions = %v, want %v", cfv.InputRestrictions, wantRestrictions)
	}
	if len(cfv.ReferenceValue) != 6 {
		t.Fatalf("len(ReferenceValue) = %d, want 6", len(cfv.ReferenceValue))
	}
	wantRefs := []CompactResource{
		{ID: "101", ResourceType: "task", Name: "Build API"},
		{ID: "202", ResourceType: "project", Name: "Q4 Launch"},
		{ID: "303", ResourceType: "portfolio", Name: "Platform"},
		{ID: "404", ResourceType: "goal", Name: "Ship v2"},
		{ID: "505", ResourceType: "team", Name: "Core"},
		{ID: "606", ResourceType: "record", Name: "Account A"},
	}
	for i, want := range wantRefs {
		got := cfv.ReferenceValue[i]
		if got == nil || got.GetID() != want.ID || got.ResourceType != want.ResourceType || got.Name != want.Name {
			t.Errorf("ReferenceValue[%d] = %+v, want %+v", i, got, want)
		}
	}
}

func TestClient_CreateCustomField_Reference(t *testing.T) {
	defer gock.Off()

	var sent map[string]any
	gock.New("https://app.asana.com").
		Post("/api/1.0/custom_fields").
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
			"gid":                "500",
			"name":               "Linked Tasks",
			"resource_subtype":   "reference",
			"input_restrictions": []string{"task", "project"},
		}})

	client := NewClient(http.DefaultClient)
	field, err := client.CreateCustomField(&CreateCustomFieldRequest{
		CustomFieldBase: CustomFieldBase{
			Name:              "Linked Tasks",
			ResourceSubtype:   FieldTypeReference,
			InputRestrictions: []string{"task", "project"},
		},
		Workspace: "999",
	})
	if err != nil {
		t.Fatal(err)
	}

	if sent["resource_subtype"] != "reference" || sent["name"] != "Linked Tasks" || sent["workspace"] != "999" {
		t.Errorf("Unexpected POST /custom_fields payload: %v", sent)
	}
	restrictions, ok := sent["input_restrictions"].([]any)
	if !ok || len(restrictions) != 2 || restrictions[0] != "task" || restrictions[1] != "project" {
		t.Errorf("input_restrictions payload = %#v, want [task project]", sent["input_restrictions"])
	}
	if field.ID != "500" || field.ResourceSubtype != FieldTypeReference || !slices.Equal(field.InputRestrictions, []string{"task", "project"}) {
		t.Errorf("Unexpected created field: %+v", field)
	}
}

func TestProject_AddProjectLocalCustomField_Reference(t *testing.T) {
	defer gock.Off()

	var sent map[string]any
	gock.New("https://app.asana.com").
		Post("/api/1.0/projects/123/addCustomFieldSetting").
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
			"gid": "777",
			"custom_field": o{
				"gid":                "501",
				"name":               "Related Projects",
				"resource_subtype":   "reference",
				"input_restrictions": []string{"project"},
			},
		}})

	client := NewClient(http.DefaultClient)
	project := &Project{ID: "123"}
	setting, err := project.AddProjectLocalCustomField(client, &AddProjectLocalCustomFieldRequest{
		CustomField: ProjectLocalCustomField{
			CustomFieldBase: CustomFieldBase{
				Name:              "Related Projects",
				ResourceSubtype:   FieldTypeReference,
				InputRestrictions: []string{"project"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	cfPayload, ok := sent["custom_field"].(map[string]any)
	if !ok || cfPayload["resource_subtype"] != "reference" {
		t.Errorf("Unexpected custom_field payload: %#v", sent["custom_field"])
	}
	restrictions, ok := cfPayload["input_restrictions"].([]any)
	if !ok || len(restrictions) != 1 || restrictions[0] != "project" {
		t.Errorf("Unexpected input_restrictions payload: %#v", cfPayload["input_restrictions"])
	}
	if setting.CustomField == nil || setting.CustomField.ID != "501" || setting.CustomField.ResourceSubtype != FieldTypeReference {
		t.Errorf("Unexpected setting response: %+v", setting)
	}
}

func TestCustomField_ReadOnlyFlagsAndDurationFormat(t *testing.T) {
	wantFields := []string{
		"id_prefix", "representation_type", "is_formula_field", "is_lookup_field", "is_value_read_only",
	}
	cfFields := Fields(CustomField{}).Fields
	cfvFields := Fields(CustomFieldValue{}).Fields
	for _, name := range wantFields {
		if !slices.Contains(cfFields, name) {
			t.Errorf("Expected Fields(CustomField{}) to include %q", name)
		}
		if !slices.Contains(cfvFields, name) {
			t.Errorf("Expected Fields(CustomFieldValue{}) to include %q", name)
		}
	}

	tests := []struct {
		name               string
		payload            string
		wantSubtype        FieldType
		wantFormat         Format
		wantRepresentation string
		wantIDPrefix       string
		wantFormula        bool
		wantLookup         bool
		wantReadOnly       bool
	}{
		{
			name: "formula field",
			payload: `{
				"gid": "1",
				"name": "Total Cost",
				"resource_subtype": "number",
				"representation_type": "formula",
				"is_formula_field": true,
				"is_lookup_field": false,
				"is_value_read_only": true
			}`,
			wantSubtype:        FieldTypeNumber,
			wantRepresentation: RepresentationTypeFormula,
			wantFormula:        true,
			wantReadOnly:       true,
		},
		{
			name: "lookup field",
			payload: `{
				"gid": "2",
				"name": "Parent Status",
				"resource_subtype": "text",
				"representation_type": "text",
				"is_lookup_field": true,
				"is_value_read_only": true
			}`,
			wantSubtype:        FieldTypeText,
			wantRepresentation: RepresentationTypeText,
			wantLookup:         true,
			wantReadOnly:       true,
		},
		{
			name: "custom id field",
			payload: `{
				"gid": "3",
				"name": "ID",
				"resource_subtype": "text",
				"representation_type": "custom_id",
				"format": "identifier",
				"id_prefix": "BUG",
				"is_value_read_only": true
			}`,
			wantSubtype:        FieldTypeText,
			wantFormat:         FormatIdentifier,
			wantRepresentation: RepresentationTypeCustomID,
			wantIDPrefix:       "BUG",
			wantReadOnly:       true,
		},
		{
			name: "duration number field",
			payload: `{
				"gid": "4",
				"name": "Estimated Effort",
				"resource_subtype": "number",
				"representation_type": "number",
				"format": "duration",
				"precision": 0,
				"is_value_read_only": false
			}`,
			wantSubtype:        FieldTypeNumber,
			wantFormat:         FormatDuration,
			wantRepresentation: RepresentationTypeNumber,
			wantReadOnly:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := &CustomField{}
			if err := json.Unmarshal([]byte(tt.payload), cf); err != nil {
				t.Fatal(err)
			}
			if cf.ResourceSubtype != tt.wantSubtype {
				t.Errorf("ResourceSubtype = %q, want %q", cf.ResourceSubtype, tt.wantSubtype)
			}
			if cf.Format != tt.wantFormat {
				t.Errorf("Format = %q, want %q", cf.Format, tt.wantFormat)
			}
			if cf.RepresentationType != tt.wantRepresentation {
				t.Errorf("RepresentationType = %q, want %q", cf.RepresentationType, tt.wantRepresentation)
			}
			if cf.IDPrefix != tt.wantIDPrefix {
				t.Errorf("IDPrefix = %q, want %q", cf.IDPrefix, tt.wantIDPrefix)
			}
			if IsTrue(cf.IsFormulaField) != tt.wantFormula {
				t.Errorf("IsFormulaField = %v, want %v", IsTrue(cf.IsFormulaField), tt.wantFormula)
			}
			if IsTrue(cf.IsLookupField) != tt.wantLookup {
				t.Errorf("IsLookupField = %v, want %v", IsTrue(cf.IsLookupField), tt.wantLookup)
			}
			if IsTrue(cf.IsValueReadOnly) != tt.wantReadOnly {
				t.Errorf("IsValueReadOnly = %v, want %v", IsTrue(cf.IsValueReadOnly), tt.wantReadOnly)
			}
		})
	}
}

func TestClient_CreateCustomField_DurationFormat(t *testing.T) {
	defer gock.Off()

	precision := 0
	var sent map[string]any
	gock.New("https://app.asana.com").
		Post("/api/1.0/custom_fields").
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
			"gid":                 "600",
			"name":                "Time Estimate",
			"resource_subtype":    "number",
			"representation_type": "number",
			"format":              "duration",
			"precision":           0,
			"is_value_read_only":  false,
		}})

	client := NewClient(http.DefaultClient)
	field, err := client.CreateCustomField(&CreateCustomFieldRequest{
		CustomFieldBase: CustomFieldBase{
			Name:            "Time Estimate",
			ResourceSubtype: FieldTypeNumber,
			Format:          FormatDuration,
			Precision:       &precision,
		},
		Workspace: "999",
	})
	if err != nil {
		t.Fatal(err)
	}

	if sent["format"] != Duration || sent["resource_subtype"] != "number" || sent["precision"] != float64(0) {
		t.Errorf("Unexpected POST /custom_fields data: %v", sent)
	}
	for _, roKey := range []string{"representation_type", "is_formula_field", "is_lookup_field", "is_value_read_only"} {
		if _, exists := sent[roKey]; exists {
			t.Errorf("Read-only field %q should not be sent in CreateCustomFieldRequest", roKey)
		}
	}
	if field.ID != "600" || field.Format != FormatDuration || field.RepresentationType != RepresentationTypeNumber {
		t.Errorf("Unexpected created field: %+v", field)
	}
}
