package asana

import "fmt"

// ResourceSubtypeCustom is the task resource_subtype Asana uses for tasks
// with a custom type
const ResourceSubtypeCustom = "custom"

// CustomTypeStatusOption is one of the statuses a task with a custom type can
// have. Statuses replace the complete checkbox for these tasks.
type CustomTypeStatusOption struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	// The name of the status, as shown in the Asana UI
	Name string `json:"name,omitempty"`

	// Whether a task with this status is complete. Asana's docs give
	// "Incomplete" as an example but don't list the values.
	CompletionState string `json:"completion_state,omitempty"`

	// Whether the status can currently be chosen
	Enabled *bool `json:"enabled,omitempty"`

	// The color shown for the status
	Color string `json:"color,omitempty"`
}

func (o *CustomTypeStatusOption) GetID() string {
	return o.ID
}

// CustomType is a task type defined in Asana, such as 'Bug' or 'Request'. Each
// type has its own set of statuses.
//
// Custom types are added to projects in the Asana UI. The API can list the
// types available in a project and set the type of an existing task, but
// can't create a task with a custom type (POST /tasks rejects it).
type CustomType struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	// The name of the type
	Name string `json:"name,omitempty"`

	// Read-only. Identifies the type when Asana created it, rather than a
	// user. Some of these types can't be assigned through the API.
	AsanaCreatedTypeIdentifier string `json:"asana_created_type_identifier,omitempty"`

	// The statuses a task of this type can have
	StatusOptions []*CustomTypeStatusOption `json:"status_options,omitempty"`
}

func (c *CustomType) GetID() string {
	return c.ID
}

// Fetch loads the full details for this CustomType
func (c *CustomType) Fetch(client *Client, opts ...*Options) error {
	client.trace("Loading custom type details for %q", c.Name)

	_, err := client.get(fmt.Sprintf("/custom_types/%s", c.ID), nil, c, opts...)
	return err
}

// CustomTypes returns the custom types available in this project
func (p *Project) CustomTypes(client *Client, opts ...*Options) ([]*CustomType, *NextPage, error) {
	client.trace("Listing custom types in %q", p.Name)
	var result []*CustomType

	query := &struct {
		Project string `url:"project"`
	}{
		Project: p.ID,
	}

	// Make the request
	nextPage, err := client.get("/custom_types", query, &result, opts...)
	return result, nextPage, err
}
