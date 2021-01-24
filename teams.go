package asana

import (
	"fmt"
)

// Team is used to group related projects and people together within an
// organization. Each project in an organization is associated with a team.
type Team struct {
	client *Client

	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	// Read-only. The name of the object.
	Name string `json:"name,omitempty"`

	// The description of the team with formatting as HTML.
	Description string `json:"description,omitempty"`

	// The description of the team with formatting as HTML.
	HTMLDescription string `json:"html_description,omitempty"`

	Organization *Workspace `json:"organization,omitempty"`
}

// Fetch loads the full details for this Team
func (t *Team) Fetch(client *Client) error {
	client.trace("Loading team details for %q\n", t.Name)

	// Use fields options to request Organization field which is not returned by default
	_, err := client.get(fmt.Sprintf("/teams/%s", t.ID), nil, t, Fields(*t))
	return err
}

// Teams returns the compact records for all teams in the organization visible to the authorized user
func (w *Workspace) Teams(options ...*Options) ([]*Team, *NextPage, error) {
	w.client.trace("Listing teams in workspace %s...\n", w.ID)
	var result []*Team

	// Make the request
	nextPage, err := w.client.get(fmt.Sprintf("/organizations/%s/teams", w.ID), nil, &result, options...)
	for _, r := range result {
		r.client = w.client
	}
	return result, nextPage, err
}

// AllTeams repeatedly pages through all available teams in a workspace
func (w *Workspace) AllTeams(options ...*Options) ([]*Team, error) {
	var allTeams []*Team
	nextPage := &NextPage{}

	var teams []*Team
	var err error

	for nextPage != nil {
		page := &Options{
			Limit:  100,
			Offset: nextPage.Offset,
		}

		allOptions := append([]*Options{page}, options...)
		teams, nextPage, err = w.Teams(allOptions...)
		if err != nil {
			return nil, err
		}

		allTeams = append(allTeams, teams...)
	}
	for _, r := range allTeams {
		r.client = w.client
	}
	return allTeams, nil
}
