package asana

import (
	"fmt"
)

// TagBase contains the modifiable fields for a Tag
type TagBase struct {
	HasName
	HasNotes
	HasColor
}

// Tag is a label that can be attached to any task in Asana. It exists in a
// single workspace or organization.
//
// Tags have some metadata associated with them, but it is possible that we
// will simplify them in the future so it is not encouraged to rely too
// heavily on it. Unlike projects, tags do not provide any ordering on the
// tasks they are associated with.
type Tag struct {
	TagBase

	HasID
	HasCreated
	HasWorkspace
	HasFollowers

	expandable
}

// Tag retrieves a tag record by ID
func (c *Client) Tag(id int64) (*Tag, error) {
	c.trace("Loading tag %d", id)

	result := &Tag{}
	result.expanded = true

	// Make the request
	err := c.get(fmt.Sprintf("/tags/%d", id), nil, result)
	return result, err
}

// Expand loads the full details for this Tag
func (t *Tag) Expand() error {
	t.trace("Loading details for tag %q", t.Name)

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
func (w *Workspace) Tags() (map[string]*Tag, error) {
	w.trace("Listing tags in %q", w.Name)

	if w.cachedTags != nil {
		return w.cachedTags, nil
	}

	var result []*Tag

	// Make the request
	err := w.Client.get(fmt.Sprintf("/workspaces/%d/tags", w.ID), nil, &result)
	if err != nil {
		return nil, err
	}

	// Index tags
	tagMap := make(map[string]*Tag)
	for _, tag := range result {
		tagMap[tag.Name] = tag
	}

	// Cache result
	w.cachedTags = tagMap
	return tagMap, nil
}

// CreateTag adds a new tag to a workspace
func (w *Workspace) CreateTag(tag *TagBase) (*Tag, error) {
	w.info("Creating tag %q in %q\n", tag.Name, w.Name)

	result := &Tag{}
	result.expanded = true

	err := w.Client.post(fmt.Sprintf("/workspaces/%d/tags", w.ID), tag, result)
	if err != nil {
		return nil, err
	}

	// Update tag cache
	if w.cachedTags != nil {
		w.cachedTags[result.Name] = result
	}

	return result, nil
}
