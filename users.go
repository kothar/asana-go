package asana

import "fmt"

// User represents an account in Asana that can be given access to various
// workspaces, projects, and tasks.
//
// Like other objects in the system, users are referred to by numerical IDs.
// However, the special string identifier me can be used anywhere a user ID is
// accepted, to refer to the current authenticated user.
type User struct {
	Expandable
	WithName

	// Read-only. The user’s email address.
	Email string `json:"email,omitempty"`

	// Read-only. A map of the user’s profile photo in various sizes, or null
	// if no photo is set. Sizes provided are 21, 27, 36, 60, and 128. Images
	// are in PNG format.
	Photo map[string]string `json:"photo,omitempty"`

	// Read-only. Workspaces and organizations this user may access.
	//
	// Note: The API will only return workspaces and organizations that also
	// contain the authenticated user.
	Workspaces []*Workspace `json:"workspaces,omitempty"`
}

// CurrentUser gets the currently authorized user
func (c *Client) CurrentUser() (*User, error) {

	result := &User{}

	_, err := c.get("/users/me", nil, result)

	return result, err
}

// User retrieves a user record by ID
func (c *Client) User(id int64) *User {
	result := &User{}
	result.init(id, c)
	return result
}

// Expand loads the full details for this User
func (u *User) Expand() error {
	u.trace("Loading details for user %q", u.ID)

	if u.expanded {
		return nil
	}

	_, err := u.client.get(fmt.Sprintf("/users/%d", u.ID), nil, u)
	return err
}
