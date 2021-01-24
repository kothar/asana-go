package asana

type Portfolio struct {
	client *Client
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`
}

// Projects returns a list of projects in this workspace
func (w *Workspace) Portfolios(options ...*Options) ([]*Portfolio, *NextPage, error) {
	w.client.trace("Listing portfolios in %q", w.Name)

	var result []*Portfolio

	o := &Options{
		Workspace: w.ID,
		Owner:     "me",
	}

	// Make the request
	nextPage, err := w.client.get("/portfolios", nil, &result, append(options, o)...)
	for _, r := range result {
		r.client = w.client
	}
	return result, nextPage, err
}
