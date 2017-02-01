package asana

import (
	"fmt"
)

// Workspace is the highest-level organizational unit in Asanc. All projects
// and tasks have an associated workspace.
//
// An organization is a special kind of workspace that represents a company.
// In an organization, you can group your projects into teams. You can read
// more about how organizations work on the Asana Guide. To tell if your
// workspace is an organization or not, check its is_organization property.
//
// Over time, we intend to migrate most workspaces into organizations and to
// release more organization-specific functionality. We may eventually
// deprecate using workspace-based APIs for organizations. Currently, and
// until after some reasonable grace period following any further
// announcements, you can still reference organizations in any workspace
// parameter.
type Workspace struct {
	Expandable
	WithName

	// Whether the workspace is an organization.
	IsOrganization bool `json:"is_organization,omitempty" dynamo:"is_organization"`

	// Undocumented in API docs
	EmailDomains []string `json:"email_domains,omitempty" dynamo:"email_domains"`

	cachedTags map[string]*Tag
}

// Workspace creates an unexpanded Workspace object with the given ID
func (c *Client) Workspace(id int64) *Workspace {
	result := &Workspace{}
	result.init(id, c)
	return result
}

// Expand loads the full details for this Workspace
func (w *Workspace) Expand() error {
	w.trace("Loading workspace details for %q\n", w.Name)

	if w.expanded {
		return nil
	}

	return w.client.get(fmt.Sprintf("/workspaces/%d", w.ID), nil, w)
}

// Workspaces returns workspaces and organizations accessible to the currently
// authorized account
func (c *Client) Workspaces() ([]*Workspace, error) {
	c.trace("Listing workspaces...\n")
	var result []*Workspace

	// Force all fields to be shown
	opts := Fields(Workspace{})

	// Make the request
	err := c.get("/workspaces", nil, &result, opts)
	return result, err
}
