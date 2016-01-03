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
	expandable
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

	return t.Client.get(fmt.Sprintf("/tags/%d", t.ID), nil, t)
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
