package asana

type EnumValue struct {
	ID      int    `json:"id" dynamo:"id"`
	Name    string `json:"name" dynamo:"name"`
	Enabled bool   `json:"enabled" dynamo:"enabled"`
	Color   string `json:"color" dynamo:"color"`
}

// Custom Fields store the metadata that is used in order to add user-
// specified information to tasks in Asana. Be sure to reference the Custom
// Fields developer documentation for more information about how custom fields
// relate to various resources in Asana.
type CustomField struct {
	Expandable

	WithName
	WithCreated

	// The type of the custom field. Must be one of the given values:
	// 'text', 'enum', 'number'
	Type string `json:"type" dynamo:"type"`

	// Only relevant for custom fields of type ‘Enum’. This array specifies
	// the possible values which an enum custom field can adopt.
	EnumOptions []*EnumValue `json:"enum_options,omitempty" dynamo:"enum_options"`

	// Only relevant for custom fields of type ‘Number’. This field dictates
	// the number of places after the decimal to round to, i.e. 0 is integer
	// values, 1 rounds to the nearest tenth, and so on.
	Precision int `json:"precision,omitempty" dynamo:"precision"`
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
	TextValue string

	// Custom fields of type number will return a number_value property
	// containing the number for the field.
	NumberValue string

	// Custom fields of type enum will return an enum_value property
	// containing an object that represents the selection of the enum value.
	EnumValue *EnumValue
}
