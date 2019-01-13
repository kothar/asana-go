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

// NewTag creates a tag record stub with the given ID
func NewTag(id string) *Tag {
	result := &Tag{}
	result.ID = id
	return result
}

// Expand loads the full details for this Tag
func (t *Tag) Expand(client *Client) error {
	client.trace("Loading details for tag %q", t.Name)

	if t.expanded {
		return nil
	}

	_, err := client.get(fmt.Sprintf("/tags/%s", t.ID), nil, t)
	return err
}

// Tags returns a list of tags in this workspace
func (w *Workspace) Tags(client *Client, options ...*Options) ([]*Tag, *NextPage, error) {
	client.trace("Listing tags in %q", w.Name)

	var result []*Tag

	// Make the request
	nextPage, err := client.get(fmt.Sprintf("/workspaces/%s/tags", w.ID), nil, &result, options...)
	return result, nextPage, err
}

// AllTags repeatedly pages through all available tags in a workspace
func (w *Workspace) AllTags(client *Client, options ...*Options) ([]*Tag, error) {
	allTags := []*Tag{}
	nextPage := &NextPage{}

	var tags []*Tag
	var err error

	for nextPage != nil {
		page := &Options{
			Limit:  100,
			Offset: nextPage.Offset,
		}

		allOptions := append([]*Options{page}, options...)
		tags, nextPage, err = w.Tags(client, allOptions...)
		if err != nil {
			return nil, err
		}

		allTags = append(allTags, tags...)
	}
	return allTags, nil
}

// CreateTag adds a new tag to a workspace
func (w *Workspace) CreateTag(client *Client, tag *TagBase) (*Tag, error) {
	client.info("Creating tag %q in %q\n", tag.Name, w.Name)

	result := &Tag{}
	result.expanded = true

	err := client.post(fmt.Sprintf("/workspaces/%s/tags", w.ID), tag, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
