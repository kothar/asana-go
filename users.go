package asana

// User represents an account in Asana that can be given access to various
// workspaces, projects, and tasks.
//
// Like other objects in the system, users are referred to by numerical IDs.
// However, the special string identifier me can be used anywhere a user ID is
// accepted, to refer to the current authenticated user.
type User struct {
	expandable
	WithName

	// Read-only. The user’s email address.
	Email string `json:"email,omitempty" dynamo:"email"`

	// Read-only. A map of the user’s profile photo in various sizes, or null
	// if no photo is set. Sizes provided are 21, 27, 36, 60, and 128. Images
	// are in PNG format.
	Photo map[string]string `json:"photo,omitempty" dynamo:"photo"`

	// Read-only. Workspaces and organizations this user may access.
	//
	// Note: The API will only return workspaces and organizations that also
	// contain the authenticated user.
	Workspaces []*Workspace `json:"workspaces,omitempty" dynamo:"workspaces"`
}

// CurrentUser gets the currently authorized user
func (c *Client) CurrentUser() (*User, error) {

	result := &User{}

	err := c.get("/users/me", nil, result)

	return result, err
}
