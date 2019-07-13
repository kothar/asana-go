package asana

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"time"
)

// Attachment represents any file attached to a task in Asana,
// whether it’s an uploaded file or one associated via a third-party service
// such as Dropbox or Google Drive.
type Attachment struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	// Read-only. The name of the object.
	Name string `json:"name,omitempty"`

	// Read-only. The task this object is attached to.
	Parent *Task `json:"parent,omitempty"`

	// Read-only. The time at which this object was created.
	CreatedAt *time.Time `json:"created_at,omitempty"`

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

	// Undocumented. A permanent asana.com link which should be a permalink
	PermanentURL string `json:"permanent_url,omitempty"`
}

// Attachments lists all attachments attached to a task
func (t *Task) Attachments(client *Client, opts ...*Options) ([]*Attachment, *NextPage, error) {
	client.trace("Listing attachments for %q", t.Name)

	var result []*Attachment

	// Make the request
	nextPage, err := client.get(fmt.Sprintf("/tasks/%s/attachments", t.ID), nil, &result, opts...)
	return result, nextPage, err
}

type NewAttachment struct {
	Reader      io.ReadCloser
	FileName    string
	ContentType string
}

func (t *Task) CreateAttachment(client *Client, request *NewAttachment) (*Attachment, error) {
	client.trace("Uploading attachment for %q", t.Name)

	result := &Attachment{}
	err := client.postMultipart(fmt.Sprintf("/tasks/%s/attachments", t.ID), result, "file", request.Reader, request.FileName, request.ContentType)
	if err != nil {
		return nil, errors.Wrap(err, "Upload attachment")
	}
	return result, nil
}
