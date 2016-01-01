package asana

import (
	"fmt"
)

// Tag is a label that can be attached to any task in Asana. It exists in a
// single workspace or organization.
//
// Tags have some metadata associated with them, but it is possible that we
// will simplify them in the future so it is not encouraged to rely too
// heavily on it. Unlike projects, tags do not provide any ordering on the
// tasks they are associated with.
type Tag struct {
	HasID
	HasName
	HasCreated
	HasNotes
	HasWorkspace
	HasFollowers
	HasColor

	expandable
}

// Tag retrieves a tag record by ID
func (c *Client) Tag(id int64) (*Tag, error) {
	result := &Tag{}
	result.expanded = true

	// Make the request
	err := c.get(fmt.Sprintf("/tags/%d", id), nil, result)
	return result, err
}

// Expand loads the full details for this Tag
func (t *Tag) Expand() error {
	if t.expanded {
		return nil
	}

	e, err := t.Client.Tag(t.ID)
	if err != nil {
		return err
	}

	*t = *e
	return nil
}

// Tags returns a list of tags in this workspace
func (w *Workspace) Tags() ([]*Tag, error) {
	var result []*Tag

	// Make the request
	err := w.Client.get(fmt.Sprintf("/workspaces/%d/tags", w.ID), nil, &result)
	return result, err
}

// CreateTag adds a new tag to a workspace
func (w *Workspace) CreateTag(tag *Tag) (*Tag, error) {
	result := &Tag{}
	result.expanded = true

	err := w.Client.post(fmt.Sprintf("/workspaces/%d/tags", w.ID), tag, result)
	return result, err
}
