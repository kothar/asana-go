package asana

import (
	"fmt"
)

// ProjectStatus is a description of the project’s status containing a color
// (must be either null or one of: green, yellow, red) and a short
// description.
type ProjectStatus struct {
	Color  string `json:"color,omitempty"`
	Text   string `json:"text,omitempty"`
	Author *User  `json:"author,omitempty"`
}

// ProjectBase contains the parts of Project which are not related to a specific instance
type ProjectBase struct {
	WithName
	WithNotes
	WithColor

	// The current owner of the project, may be null.
	Owner *User `json:"owner,omitempty"`

	// A description of the project’s status containing a color (must be
	// either null or one of: green, yellow, red) and a short description.
	CurrentStatus *ProjectStatus `json:"current_status,omitempty"`

	// The day on which this project is due. This takes a date with format
	// YYYY-MM-DD.
	DueDate *Date `json:"due_date,omitempty"`

	// True if the project is archived, false if not. Archived projects do not
	// show in the UI by default and may be treated differently for queries.
	Archived bool `json:"archived,omitempty"`

	// True if the project is public to the organization. If false, do not
	// share this project with other users in this organization without
	// explicitly checking to see if they have access.
	Public bool `json:"public,omitempty"`

	// Create-only. The team that this project is shared with. This field only
	// exists for projects in organizations.
	Team *Team `json:"team,omitempty"`
}

// NewProject represents a request to create a new project
type NewProject struct {
	ProjectBase

	Workspace int64 `json:"workspace,omitempty"`
	Team      int64 `json:"team,omitempty"`
}

// Project represents a prioritized list of tasks in Asana. It exists in a
// single workspace or organization and is accessible to a subset of users in
// that workspace or organization, depending on its permissions.
//
// Projects in organizations are shared with a single team. You cannot
// currently change the team of a project via the API. Non-organization
// workspaces do not have teams and so you should not specify the team of
// project in a regular workspace.
//
// Followers of a project are a subset of the members of that project.
// Followers of a project will receive all updates including tasks created,
// added and removed from that project. Members of the project have access to
// and will receive status updates of the project. Adding followers to a
// project will add them as members if they are not already, removing
// followers from a project will not affect membership.
type Project struct {
	expandable
	ProjectBase

	WithDates
	WithWorkspace
	WithFollowers

	// Read-only. Array of users who are members of this project.
	Members []*User `json:"members,omitempty"`
}

// Project creates an unexpaned Project object with the given ID
func (c *Client) Project(id int64) *Project {
	result := &Project{}
	result.init(id, c)
	return result
}

// Expand loads the full details for this Project
func (p *Project) Expand() error {
	p.trace("Loading project details for %q", p.Name)

	if p.expanded {
		return nil
	}

	return p.Client.get(fmt.Sprintf("/projects/%d", p.ID), nil, p)
}

// Projects returns a list of projects in this workspace
func (w *Workspace) Projects() ([]*Project, error) {
	w.trace("Listing projects in %q", w.Name)

	var result []*Project

	// Make the request
	err := w.Client.get(fmt.Sprintf("/workspaces/%d/projects", w.ID), nil, &result)
	return result, err
}

// CreateProject adds a new project to a workspace
func (c *Client) CreateProject(project *NewProject) (*Project, error) {
	c.info("Creating project %q\n", project.Name)

	result := &Project{}
	result.expanded = true

	err := c.post("/projects", project, result)
	return result, err
}
