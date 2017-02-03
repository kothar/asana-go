package asana

import (
	"fmt"
)

// TagBase contains the modifiable fields for a Tag
type TagBase struct {
	WithName
	WithNotes
	WithColor
}

// Tag is a label that can be attached to any task in Asana. It exists in a
// single workspace or organization.
//
// Tags have some metadata associated with them, but it is possible that we
// will simplify them in the future so it is not encouraged to rely too
// heavily on it. Unlike projects, tags do not provide any ordering on the
// tasks they are associated with.
type Tag struct {
	Expandable
	TagBase

	WithCreated
	WithWorkspace
	WithFollowers
}

// Tag retrieves a tag record by ID
func (c *Client) Tag(id int64) *Tag {
	result := &Tag{}
	result.init(id, c)
	return result
}

// Expand loads the full details for this Tag
func (t *Tag) Expand() error {
	t.trace("Loading details for tag %q", t.Name)

	if t.expanded {
		return nil
	}

	_, err := t.client.get(fmt.Sprintf("/tags/%d", t.ID), nil, t)
	return err
}

// Tags returns a list of tags in this workspace
func (w *Workspace) Tags() ([]*Tag, *NextPage, error) {
	w.trace("Listing tags in %q", w.Name)

	var result []*Tag

	// Make the request
	nextPage, err := w.client.get(fmt.Sprintf("/workspaces/%d/tags", w.ID), nil, &result)
	return result, nextPage, err
}

// CreateTag adds a new tag to a workspace
func (w *Workspace) CreateTag(tag *TagBase) (*Tag, error) {
	w.info("Creating tag %q in %q\n", tag.Name, w.Name)

	result := &Tag{}
	result.expanded = true

	err := w.client.post(fmt.Sprintf("/workspaces/%d/tags", w.ID), tag, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
