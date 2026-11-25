package asana

import (
	"fmt"
	"time"
)

// StatusType represents the state reported by a StatusUpdate
type StatusType string

const (
	// Project and portfolio status types
	StatusTypeOnTrack  StatusType = "on_track"
	StatusTypeAtRisk   StatusType = "at_risk"
	StatusTypeOffTrack StatusType = "off_track"
	StatusTypeOnHold   StatusType = "on_hold"
	StatusTypeComplete StatusType = "complete"

	// Goal status types
	StatusTypeAchieved StatusType = "achieved"
	StatusTypePartial  StatusType = "partial"
	StatusTypeMissed   StatusType = "missed"
	StatusTypeDropped  StatusType = "dropped"
)

// StatusUpdateBase contains the content fields for a StatusUpdate
type StatusUpdateBase struct {
	// The title of the status update.
	Title string `json:"title,omitempty"`

	// The text content of the status update.
	Text string `json:"text,omitempty"`

	// Opt In. The text content of the status update with formatting as HTML.
	HTMLText string `json:"html_text,omitempty"`

	// The type associated with the status update.
	StatusType StatusType `json:"status_type,omitempty"`
}

// CreateStatusUpdateRequest represents a request to create a new StatusUpdate
type CreateStatusUpdateRequest struct {
	StatusUpdateBase

	// Required: The gid of the parent (project, portfolio, or goal) to post
	// this status update to.
	Parent string `json:"parent"`
}

// Validate checks that required fields are populated on CreateStatusUpdateRequest
func (r *CreateStatusUpdateRequest) Validate() error {
	if r.Parent == "" {
		return fmt.Errorf("a parent ID is required to create a status update")
	}
	return nil
}

// StatusUpdateQuery specifies which status updates to return from StatusUpdates
type StatusUpdateQuery struct {
	// Required: Globally unique identifier for the parent project, portfolio, or goal.
	Parent string `url:"parent"`

	// Optional: Only return status updates created since the given time (ISO 8601).
	CreatedSince string `url:"created_since,omitempty"`
}

// Validate checks that required fields are populated on StatusUpdateQuery
func (q *StatusUpdateQuery) Validate() error {
	if q.Parent == "" {
		return fmt.Errorf("a parent ID is required to list status updates")
	}
	return nil
}

// StatusUpdate represents a point-in-time status update on a project, portfolio, or goal.
type StatusUpdate struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	// Read-only. The subtype of this status update (project_status_update,
	// portfolio_status_update, or goal_status_update).
	ResourceSubtype string `json:"resource_subtype,omitempty"`

	StatusUpdateBase

	// Read-only. The user who authored this status update.
	Author *User `json:"author,omitempty"`

	// Read-only. The time at which this status update was created.
	CreatedAt *time.Time `json:"created_at,omitempty"`

	// Read-only. The user who created this status update.
	CreatedBy *User `json:"created_by,omitempty"`

	// Read-only. The time at which this status update was last modified.
	ModifiedAt *time.Time `json:"modified_at,omitempty"`

	// True if the status update is liked by the authorized user.
	Liked bool `json:"liked,omitempty"`

	// Read-only. The number of users who have liked this status update.
	NumLikes int32 `json:"num_likes,omitempty"`

	// Read-only. The parent resource (project, portfolio, or goal) this status
	// update belongs to.
	Parent *CompactResource `json:"parent,omitempty"`
}

func (s *StatusUpdate) GetID() string {
	return s.ID
}

// Fetch loads the full details for this StatusUpdate
func (s *StatusUpdate) Fetch(client *Client, opts ...*Options) error {
	client.trace("Loading status update details for %q", s.ID)

	_, err := client.get(fmt.Sprintf("/status_updates/%s", s.ID), nil, s, opts...)
	return err
}

// Delete removes this status update
func (s *StatusUpdate) Delete(client *Client) error {
	client.info("Deleting status update %q", s.ID)

	return client.delete(fmt.Sprintf("/status_updates/%s", s.ID))
}

// StatusUpdates lists status updates matching the given query (requiring Parent)
func (c *Client) StatusUpdates(query *StatusUpdateQuery, opts ...*Options) ([]*StatusUpdate, *NextPage, error) {
	if query == nil {
		return nil, nil, fmt.Errorf("a status update query with a parent ID is required")
	}

	c.trace("Listing status updates for parent %q", query.Parent)
	var result []*StatusUpdate

	nextPage, err := c.get("/status_updates", query, &result, opts...)
	return result, nextPage, err
}

// CreateStatusUpdate creates a new status update on the specified parent object
func (c *Client) CreateStatusUpdate(request *CreateStatusUpdateRequest, opts ...*Options) (*StatusUpdate, error) {
	if request == nil {
		return nil, fmt.Errorf("a status update request is required")
	}
	c.info("Creating status update %q on parent %q", request.Title, request.Parent)

	result := &StatusUpdate{}
	err := c.post("/status_updates", request, result, opts...)
	return result, err
}

// StatusUpdates returns a page of status updates for this project
func (p *Project) StatusUpdates(client *Client, opts ...*Options) ([]*StatusUpdate, *NextPage, error) {
	return client.StatusUpdates(&StatusUpdateQuery{Parent: p.ID}, opts...)
}
