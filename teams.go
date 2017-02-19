package asana

import (
	"fmt"
)

// Team is used to group related projects and people together within an
// organization. Each project in an organization is associated with a team.
type Team struct {
	Expandable
	WithName

	Organization *Workspace `json:"organization,omitempty"`
}

// Team creates an unexpanded Team object with the given ID
func (c *Client) Team(id int64) *Team {
	result := &Team{}
	result.init(id, c)
	return result
}

// Expand loads the full details for this Team
func (t *Team) Expand() error {
	t.trace("Loading team details for %q\n", t.Name)

	if t.expanded {
		return nil
	}

	// Use fields options to request Organization field which is not returned by default
	_, err := t.client.get(fmt.Sprintf("/teams/%d", t.ID), nil, t, Fields(*t))

	// This should be set by injection, not sure why it isn't
	if t.Organization != nil {
		t.Organization.client = t.client
	}
	return err
}

// Teams returns the compact records for all teams in the organization visible to the authorized user
func (w *Workspace) Teams(options ...*Options) ([]*Team, *NextPage, error) {
	w.trace("Listing teams in workspace %d...\n", w.ID)
	var result []*Team

	// Make the request
	nextPage, err := w.client.get(fmt.Sprintf("/organizations/%d/teams", w.ID), nil, &result, options...)
	return result, nextPage, err
}
