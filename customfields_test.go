package asana

import (
    "encoding/json"
    "testing"
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
	"name": "Priority",
	"resource_subtype": "enum",
	"asana_created_field": "priority"
}
`), cf); err != nil {
        t.Fatal(err)
    }

    if cf.AsanaCreatedField != "priority" {
        t.Errorf("Expected AsanaCreatedField to be %q, but saw %q", "priority", cf.AsanaCreatedField)
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
