package asana

import "fmt"

type Portfolio struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`
}

// Projects returns a list of projects in this workspace
func (w *Workspace) Portfolios(client *Client, options ...*Options) ([]*Portfolio, *NextPage, error) {
	client.trace("Listing portfolios in %q", w.Name)

	var result []*Portfolio

	o := &Options{
		Workspace: w.ID,
		Owner:     "me",
	}

	// Make the request
	nextPage, err := client.get(fmt.Sprintf("/portfolios"), nil, &result, append(options, o)...)
	return result, nextPage, err
}
