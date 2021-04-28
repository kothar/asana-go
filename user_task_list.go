package asana

import "fmt"

// UserTaskList represents the tasks assigned to a particular user.
type UserTaskList struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	// Read-only. The name of the object.
	Name string `json:"name,omitempty"`

	// The owner of the user task list.
	Owner *User `json:"owner,omitempty"`

	// Read-only. Workspaces and organizations this user may access.
	//
	// Note: The API will only return workspaces and organizations that also
	// contain the authenticated user.
	Workspaces []*Workspace `json:"workspaces,omitempty"`
}

// Fetch loads the full details for this UserTaskList
func (u *UserTaskList) Fetch(client *Client, options ...*Options) error {
	client.trace("Loading details for user task list %q", u.ID)

	_, err := client.get(fmt.Sprintf("/user_task_lists/%s", u.ID), nil, u, options...)
	return err
}
