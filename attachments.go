package asana

import "fmt"

// Attachment represents any file attached to a task in Asana,
// whether it’s an uploaded file or one associated via a third-party service
// such as Dropbox or Google Drive.
type Attachment struct {
	Expandable
	WithName
	WithParent
	WithCreated

	// Read-only. The URL containing the content of the attachment.
	//
	// Note: May be null if the attachment is hosted by box. If present, this
	// URL may only be valid for 1 hour from the time of retrieval. You should
	// avoid persisting this URL somewhere and just refresh it on demand to
	// ensure you do not keep stale URLs.
	DownloadURL string `json:"download_url,omitempty"`

	// Read-only. The service hosting the attachment. Valid values are asana,
	// dropbox, gdrive and box.
	Host string `json:"host,omitempty"`

	// Read-only. The URL where the attachment can be viewed, which may be
	// friendlier to users in a browser than just directing them to a raw
	// file.
	ViewURL string `json:"view_url,omitempty"`
}

// Attachments lists all stories attached to a task
func (t *Task) Attachments(opts ...*Options) ([]*Attachment, *NextPage, error) {
	t.trace("Listing attachments for %q", t.Name)

	var result []*Attachment

	// Make the request
	nextPage, err := t.client.get(fmt.Sprintf("/tasks/%d/attachments", t.ID), nil, &result, opts...)
	return result, nextPage, err
}
