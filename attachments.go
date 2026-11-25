package asana

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

// Attachment represents any file attached to a task in Asana,
// whether it’s an uploaded file or one associated via a third-party service
// such as Dropbox or Google Drive.
type Attachment struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	// Read-only. The base type of this resource
	ResourceType string `json:"resource_type,omitempty"`

	// Read-only. The name of the object.
	Name string `json:"name,omitempty"`

	// Read-only. The service hosting the attachment. Valid values are
	// asana, dropbox, gdrive, onedrive, box, vimeo, and external.
	ResourceSubtype string `json:"resource_subtype,omitempty"`

	// Read-only. Whether the attachment is connected to the app making the request for the
	// purposes of showing an app components widget.
	// Only present when the resource_subtype is external or gdrive.
	ConnectedToApp *bool `json:"connected_to_app,omitempty"`

	// Read-only. The time at which this object was created.
	CreatedAt *time.Time `json:"created_at,omitempty"`

	// Read-only. The URL containing the content of the attachment.
	//
	// Note: May be null if the attachment is hosted by box. If present, this
	// URL may only be valid for 1 hour from the time of retrieval. You should
	// avoid persisting this URL somewhere and just refresh it on demand to
	// ensure you do not keep stale URLs.
	DownloadURL string `json:"download_url,omitempty"`

	// Read-only. The service hosting the attachment. Valid values are
	// asana, dropbox, gdrive, box, and vimeo.
	Host string `json:"host,omitempty"`

	// Read-only. The task this object is attached to.
	//
	// Note: attachments can also belong to a project or project brief. Asana
	// documents this field as a compact task, so for those parents only the
	// gid and name are decoded here.
	Parent *Task `json:"parent,omitempty"`

	// Undocumented. A permanent asana.com link which should be a permalink
	PermanentURL string `json:"permanent_url,omitempty"`

	// Read-only. The size of the attachment in bytes. Only present when the resource_subtype is asana.
	Size *int `json:"size,omitempty"`

	// Read-only. The URL where the attachment can be viewed, which may be
	// friendlier to users in a browser than just directing them to a raw
	// file.
	ViewURL string `json:"view_url,omitempty"`
}

func (a *Attachment) GetID() string {
	return a.ID
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
	err := client.postMultipart(fmt.Sprintf("/tasks/%s/attachments", t.ID), result, nil, "file", request.Reader, request.FileName, request.ContentType)
	if err != nil {
		return nil, fmt.Errorf("upload attachment: %w", err)
	}
	return result, nil
}

type ExternalAttachmentRequest struct {
	ConnectToApp    *bool  `json:"connect_to_app,omitempty"`
	Name            string `json:"name"`
	URL             string `json:"url"`
	ResourceSubtype string `json:"resource_subtype"`
}

func (t *Task) CreateExternalAttachment(client *Client, request *ExternalAttachmentRequest) (*Attachment, error) {
	client.trace("Creating external attachment for %q", t.Name)
	request.ResourceSubtype = "external"

	result := &Attachment{}
	err := client.post(fmt.Sprintf("/tasks/%s/attachments", t.ID), request, result)
	if err != nil {
		return nil, fmt.Errorf("create external attachment: %w", err)
	}
	return result, nil
}

// attachmentsRequestParams selects the object to list attachments from
type attachmentsRequestParams struct {
	// Globally unique identifier for a project, project brief or task
	Parent string `url:"parent"`
}

// parentAttachments lists the attachments on a project, project brief or
// task using GET /attachments
func (c *Client) parentAttachments(parent string, opts ...*Options) ([]*Attachment, *NextPage, error) {
	var result []*Attachment

	query := &attachmentsRequestParams{Parent: parent}
	nextPage, err := c.get("/attachments", query, &result, opts...)
	return result, nextPage, err
}

// createParentAttachment uploads a file attachment to a project, project
// brief or task using POST /attachments
func (c *Client) createParentAttachment(parent string, request *NewAttachment) (*Attachment, error) {
	fields := []formField{{name: "parent", value: parent}}

	result := &Attachment{}
	err := c.postMultipart("/attachments", result, fields, "file", request.Reader, request.FileName, request.ContentType)
	if err != nil {
		return nil, fmt.Errorf("upload attachment: %w", err)
	}
	return result, nil
}

// createParentExternalAttachment attaches an external URL to a project,
// project brief or task using POST /attachments. The endpoint only accepts
// multipart/form-data, so the request is sent as form fields rather than JSON.
func (c *Client) createParentExternalAttachment(parent string, request *ExternalAttachmentRequest) (*Attachment, error) {
	fields := []formField{
		{name: "parent", value: parent},
		{name: "resource_subtype", value: "external"},
		{name: "name", value: request.Name},
		{name: "url", value: request.URL},
	}
	if request.ConnectToApp != nil {
		fields = append(fields, formField{name: "connect_to_app", value: strconv.FormatBool(*request.ConnectToApp)})
	}

	result := &Attachment{}
	err := c.postMultipart("/attachments", result, fields, "", nil, "", "")
	if err != nil {
		return nil, fmt.Errorf("create external attachment: %w", err)
	}
	return result, nil
}

// Attachments lists the files in the "Key resources" section of this
// project's Overview
func (p *Project) Attachments(client *Client, opts ...*Options) ([]*Attachment, *NextPage, error) {
	client.trace("Listing attachments for project %q", p.Name)

	return client.parentAttachments(p.ID, opts...)
}

// CreateAttachment uploads a file to the "Key resources" section of this
// project's Overview
func (p *Project) CreateAttachment(client *Client, request *NewAttachment) (*Attachment, error) {
	client.trace("Uploading attachment for project %q", p.Name)

	return client.createParentAttachment(p.ID, request)
}

// CreateExternalAttachment attaches an external URL to the "Key resources"
// section of this project's Overview. ConnectToApp only applies to task
// attachments.
func (p *Project) CreateExternalAttachment(client *Client, request *ExternalAttachmentRequest) (*Attachment, error) {
	client.trace("Creating external attachment for project %q", p.Name)

	return client.createParentExternalAttachment(p.ID, request)
}

// Attachments lists the files inline in this project brief
func (b *ProjectBrief) Attachments(client *Client, opts ...*Options) ([]*Attachment, *NextPage, error) {
	client.trace("Listing attachments for project brief %q", b.ID)

	return client.parentAttachments(b.ID, opts...)
}

// CreateAttachment uploads a file to this project brief
func (b *ProjectBrief) CreateAttachment(client *Client, request *NewAttachment) (*Attachment, error) {
	client.trace("Uploading attachment for project brief %q", b.ID)

	return client.createParentAttachment(b.ID, request)
}

// CreateExternalAttachment attaches an external URL to this project brief.
// ConnectToApp only applies to task attachments.
func (b *ProjectBrief) CreateExternalAttachment(client *Client, request *ExternalAttachmentRequest) (*Attachment, error) {
	client.trace("Creating external attachment for project brief %q", b.ID)

	return client.createParentExternalAttachment(b.ID, request)
}
