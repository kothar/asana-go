package asana

import (
	"encoding/json"
	"fmt"
	"time"
)

type EnumValue struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	EnumValueBase

	Enabled bool `json:"enabled"`
}

type EnumValueBase struct {
	// Read-only. The name of the object.
	Name string `json:"name,omitempty"`

	Color string `json:"color"`
}

type FieldType string

// FieldTypes for CustomField.Type field
const (
	Text   FieldType = "text"
	Enum   FieldType = "enum"
	Number FieldType = "number"
)

// Custom Fields store the metadata that is used in order to add user-
// specified information to tasks in Asana. Be sure to reference the Custom
// Fields developer documentation for more information about how custom fields
// relate to various resources in Asana.
type CustomField struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	// Read-only. The name of the object.
	Name string `json:"name,omitempty"`

	// The description of the custom field.
	Description string `json:"description,omitempty"`

	// Read-only. The time at which this object was created.
	CreatedAt *time.Time `json:"created_at,omitempty"`

	// The type of the custom field. Must be one of the given values:
	// 'text', 'enum', 'number'
	ResourceSubtype FieldType `json:"resource_subtype"`

	// Only relevant for custom fields of type ‘Enum’. This array specifies
	// the possible values which an enum custom field can adopt.
	EnumOptions []*EnumValue `json:"enum_options,omitempty"`

	// Only relevant for custom fields of type ‘Number’. This field dictates
	// the number of places after the decimal to round to, i.e. 0 is integer
	// values, 1 rounds to the nearest tenth, and so on.
	Precision int `json:"precision,omitempty"`
}

type CustomFieldSetting struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	CustomField *CustomField `json:"custom_field"`

	Project *Project `json:"project,omitempty"`

	Important bool `json:"is_important,omitempty"`
}

type AddCustomFieldSettingRequest struct {
	CustomField  string `json:"custom_field"`
	Important    bool   `json:"is_important,omitempty"`
	InsertBefore string `json:"insert_before,omitempty"`
	InsertAfter  string `json:"insert_after,omitempty"`
}

func (p *Project) AddCustomFieldSetting(client *Client, request *AddCustomFieldSettingRequest) (*CustomFieldSetting, error) {
	client.trace("Attach custom field %q to project %q", request.CustomField, p.ID)

	// Custom request encoding
	m := map[string]interface{}{}
	m["custom_field"] = request.CustomField
	m["is_important"] = request.Important

	if request.InsertAfter == "-" {
		m["insert_after"] = nil
	} else if request.InsertAfter != "" {
		m["insert_after"] = request.InsertAfter
	}

	if request.InsertBefore == "-" {
		m["insert_before"] = nil
	} else if request.InsertBefore != "" {
		m["insert_before"] = request.InsertBefore
	}

	result := &CustomFieldSetting{}
	err := client.post(fmt.Sprintf("/projects/%s/addCustomFieldSetting", p.ID), m, result)
	return result, err
}

func (p *Project) RemoveCustomFieldSetting(client *Client, customFieldID string) error {
	client.trace("Remove custom field %q from project %q", customFieldID, p.ID)

	// Custom request encoding
	m := map[string]interface{}{
		"custom_field": customFieldID,
	}

	err := client.post(fmt.Sprintf("/projects/%s/removeCustomFieldSetting", p.ID), m, &json.RawMessage{})
	return err
}

type CreateCustomFieldRequest struct {
	// Required: The workspace to create a custom field in.
	Workspace string `json:"workspace"`

	// Required: The type of the custom field. Must be one of the given values:
	// 'text', 'enum', 'number'
	ResourceSubtype FieldType `json:"resource_subtype"`

	// Required. The name of the object.
	Name string `json:"name"`

	// The description of the custom field.
	Description string `json:"description,omitempty"`

	// The number of decimal places for the numerical values.
	// Required if the custom field is of type ‘number’.
	Precision int `json:"precision"`

	// The discrete values the custom field can assume.
	// Required if the custom field is of type ‘enum’.
	EnumOptions []*EnumValueBase `json:"enum_options,omitempty"`
}

func (c *Client) CreateCustomField(request *CreateCustomFieldRequest) (*CustomField, error) {
	c.trace("Create custom field %q in workspace %s", request.Name, request.Workspace)

	result := &CustomField{}
	err := c.post("/custom_fields", request, result)
	return result, err
}

// When a custom field is associated with a project, tasks in that project can
// carry additional custom field values which represent the value of the field
// on that particular task - for instance, the selected item from an enum type
// custom field. These custom fields will appear as an array in a
// custom_fields property of the task, along with some basic information which
// can be used to associate the custom field value with the custom field
// metadata.
type CustomFieldValue struct {
	CustomField

	// Custom fields of type text will return a text_value property containing
	// the string of text for the field.
	TextValue *string `json:"text_value,omitempty"`

	// Custom fields of type number will return a number_value property
	// containing the number for the field.
	NumberValue *float64 `json:"number_value,omitempty"`

	// Custom fields of type enum will return an enum_value property
	// containing an object that represents the selection of the enum value.
	EnumValue *EnumValue `json:"enum_value,omitempty"`
}

// Fetch loads the full details for this CustomField
func (f *CustomField) Fetch(client *Client, options ...*Options) error {
	client.trace("Loading details for custom field %q", f.ID)

	_, err := client.get(fmt.Sprintf("/custom_fields/%s", f.ID), nil, f, options...)
	return err
}

// CustomFields returns the compact records for all custom fields in the workspace
func (w *Workspace) CustomFields(client *Client, options ...*Options) ([]*CustomField, *NextPage, error) {
	client.trace("Listing custom fields in workspace %s...\n", w.ID)
	var result []*CustomField

	// Make the request
	nextPage, err := client.get(fmt.Sprintf("/workspaces/%s/custom_fields", w.ID), nil, &result, options...)
	return result, nextPage, err
}

// AllCustomFields repeatedly pages through all available custom fields in a workspace
func (w *Workspace) AllCustomFields(client *Client, options ...*Options) ([]*CustomField, error) {
	var allCustomFields []*CustomField
	nextPage := &NextPage{}

	var customFields []*CustomField
	var err error

	for nextPage != nil {
		page := &Options{
			Limit:  100,
			Offset: nextPage.Offset,
		}

		allOptions := append([]*Options{page}, options...)
		customFields, nextPage, err = w.CustomFields(client, allOptions...)
		if err != nil {
			return nil, err
		}

		allCustomFields = append(allCustomFields, customFields...)
	}
	return allCustomFields, nil
}
