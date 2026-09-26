package asana

import "fmt"

// WorkspaceMembership describes a user's membership of a workspace or
// organization, including whether they are a guest
type WorkspaceMembership struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	// Read-only. The base type of this resource
	ResourceType string `json:"resource_type,omitempty"`

	// Read-only. The user this membership belongs to
	User *User `json:"user,omitempty"`

	// Read-only. The workspace this membership grants access to
	Workspace *Workspace `json:"workspace,omitempty"`

	// Read-only. The user's My Tasks list in this workspace
	UserTaskList *UserTaskList `json:"user_task_list,omitempty"`

	// Read-only. Whether the user is currently an active member of the
	// workspace
	IsActive *bool `json:"is_active,omitempty"`

	// Read-only. Whether the user is an admin of the workspace
	IsAdmin *bool `json:"is_admin,omitempty"`

	// Read-only. Whether the user is a guest of the workspace. Guests can
	// only see content that has been shared with them, and cannot use
	// premium features such as custom fields.
	IsGuest *bool `json:"is_guest,omitempty"`

	// Read-only. Whether the user is a limited access (view only) member
	IsViewOnly *bool `json:"is_view_only,omitempty"`
}

type workspaceMembershipQuery struct {
	User string `url:"user,omitempty"`
}

// Fetch loads the full details for this WorkspaceMembership
func (m *WorkspaceMembership) Fetch(client *Client, options ...*Options) error {
	client.trace("Loading details for workspace membership %q", m.ID)

	_, err := client.get(fmt.Sprintf("/workspace_memberships/%s", m.ID), nil, m, options...)
	return err
}

// WorkspaceMemberships returns the compact workspace membership records for
// this user. The special user ID "me" refers to the authorized user.
func (u *User) WorkspaceMemberships(client *Client, options ...*Options) ([]*WorkspaceMembership, *NextPage, error) {
	client.trace("Listing workspace memberships for user %q", u.ID)
	var result []*WorkspaceMembership

	nextPage, err := client.get(fmt.Sprintf("/users/%s/workspace_memberships", u.ID), nil, &result, options...)
	return result, nextPage, err
}

// WorkspaceMemberships returns the compact workspace membership records for
// this workspace
func (w *Workspace) WorkspaceMemberships(client *Client, options ...*Options) ([]*WorkspaceMembership, *NextPage, error) {
	client.trace("Listing workspace memberships in workspace %q", w.ID)
	var result []*WorkspaceMembership

	nextPage, err := client.get(fmt.Sprintf("/workspaces/%s/workspace_memberships", w.ID), nil, &result, options...)
	return result, nextPage, err
}

// MembershipFor returns the full workspace membership record for the given
// user in this workspace, or nil if the user is not a member. The special
// user ID "me" refers to the authorized user.
func (w *Workspace) MembershipFor(client *Client, userID string, options ...*Options) (*WorkspaceMembership, error) {
	client.trace("Loading workspace membership for user %q in workspace %q", userID, w.ID)
	var result []*WorkspaceMembership

	query := &workspaceMembershipQuery{User: userID}
	allOptions := append([]*Options{Fields(WorkspaceMembership{})}, options...)
	if _, err := client.get(fmt.Sprintf("/workspaces/%s/workspace_memberships", w.ID), query, &result, allOptions...); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result[0], nil
}
