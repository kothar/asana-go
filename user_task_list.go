package asana

import "fmt"

// UserTaskList represents the tasks assigned to a particular user. It provides
// API access to a user's My Tasks view in Asana.
type UserTaskList struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	// Read-only. The name of the object.
	Name string `json:"name,omitempty"`

	// Read-only. The owner of the user task list, i.e. the person whose My
	// Tasks is represented by this resource.
	Owner *User `json:"owner,omitempty"`

	// Read-only. The workspace in which the user task list is located.
	Workspace *Workspace `json:"workspace,omitempty"`
}

// Fetch loads the full details for this UserTaskList
func (u *UserTaskList) Fetch(client *Client, options ...*Options) error {
	client.trace("Loading details for user task list %q", u.ID)

	_, err := client.get(fmt.Sprintf("/user_task_lists/%s", u.ID), nil, u, options...)
	return err
}

// TaskList returns the user task list (My Tasks) for this user in the given
// workspace
func (u *User) TaskList(client *Client, workspace *Workspace, options ...*Options) (*UserTaskList, error) {
	if workspace == nil || workspace.ID == "" {
		return nil, fmt.Errorf("a workspace is required to load a user task list")
	}

	client.trace("Loading task list for user %q in workspace %q", u.ID, workspace.ID)

	result := &UserTaskList{}

	// The workspace is applied last: client.get merges options in order, so
	// this ensures the request targets the requested workspace even if the
	// caller supplied one of their own
	queryOptions := append(append([]*Options{}, options...), &Options{Workspace: workspace.ID})
	_, err := client.get(fmt.Sprintf("/users/%s/user_task_list", u.ID), nil, result, queryOptions...)
	return result, err
}

// Tasks returns the compact list of tasks in this user task list
func (u *UserTaskList) Tasks(client *Client, options ...*Options) ([]*Task, *NextPage, error) {
	client.trace("Listing tasks in user task list %q", u.ID)

	var result []*Task

	// Make the request
	nextPage, err := client.get(fmt.Sprintf("/user_task_lists/%s/tasks", u.ID), nil, &result, options...)
	return result, nextPage, err
}
