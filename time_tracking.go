package asana

import (
	"fmt"
	"time"
)

// BillableStatus represents whether a TimeTrackingEntry is billable.
// Setting this field requires the Timesheets and Budgets add-on.
type BillableStatus string

const (
	BillableStatusBillable      BillableStatus = "billable"
	BillableStatusNonBillable   BillableStatus = "nonBillable"
	BillableStatusNotApplicable BillableStatus = "notApplicable"
)

// Approval statuses reported on TimeTrackingEntry.ApprovalStatus
const (
	TimeTrackingApprovalDraft     = "DRAFT"
	TimeTrackingApprovalSubmitted = "SUBMITTED"
	TimeTrackingApprovalApproved  = "APPROVED"
	TimeTrackingApprovalRejected  = "REJECTED"
)

// TimeTrackingCategory represents a workspace time tracking category.
type TimeTrackingCategory struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	// The name of the time tracking category.
	Name string `json:"name,omitempty"`

	// The color associated with this category for display purposes.
	Color string `json:"color,omitempty"`
}

func (c *TimeTrackingCategory) GetID() string {
	return c.ID
}

// TimeTrackingEntry represents time logged against a task in Asana.
type TimeTrackingEntry struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	// Time in minutes tracked by the entry.
	DurationMinutes int `json:"duration_minutes,omitempty"`

	// The day that this entry is logged on (YYYY-MM-DD).
	EnteredOn *Date `json:"entered_on,omitempty"`

	// The project which the time is attributable to, or null if none.
	AttributableTo *Project `json:"attributable_to,omitempty"`

	// The user this entry's logged time is attributed to.
	Assignee *User `json:"assignee,omitempty"`

	// Read-only. The user who created the entry.
	CreatedBy *User `json:"created_by,omitempty"`

	// Read-only. The time at which this entry was created.
	CreatedAt *time.Time `json:"created_at,omitempty"`

	// Read-only. The task this entry is logged on.
	Task *Task `json:"task,omitempty"`

	// Read-only. The current approval status of the entry (DRAFT, SUBMITTED,
	// APPROVED, or REJECTED).
	ApprovalStatus string `json:"approval_status,omitempty"`

	// The current billable status of the entry; writes require Timesheets and Budgets.
	BillableStatus BillableStatus `json:"billable_status,omitempty"`

	// The description of the entry; writes require Timesheets and Budgets.
	Description string `json:"description,omitempty"`

	// Read-only. Categories assigned to this entry. Requires the
	// time_tracking_categories:read scope, so it is excluded from
	// Fields(TimeTrackingEntry{}) by default; request "categories" explicitly
	// in Options.Fields when that scope is available.
	Categories []*TimeTrackingCategory `json:"categories,omitempty" asana:"-"`
}

func (e *TimeTrackingEntry) GetID() string {
	return e.ID
}

// CreateTimeTrackingEntryRequest represents a request to create a time tracking
// entry on a task.
type CreateTimeTrackingEntryRequest struct {
	// Required: Time in minutes tracked by the entry. Must be greater than 0.
	DurationMinutes int `json:"duration_minutes"`

	// Optional: The day that this entry is logged on. Defaults to today if omitted.
	EnteredOn *Date `json:"entered_on,omitempty"`

	// Optional: The gid of the project which the time is attributable to.
	AttributableTo string `json:"attributable_to,omitempty"`

	// Optional: The gid of the user this entry's logged time is attributed to.
	// Defaults to the current user; can only be set to another user if the
	// caller is their time reviewer or an admin (requires Timesheets and Budgets).
	Assignee string `json:"assignee,omitempty"`

	// Optional: The billable status of the entry. Requires Timesheets and Budgets.
	BillableStatus BillableStatus `json:"billable_status,omitempty"`

	// Optional: The description of the entry. Requires Timesheets and Budgets.
	Description string `json:"description,omitempty"`

	// Optional: Time tracking category gids to assign to the entry.
	// Requires Timesheets and Budgets.
	Categories []string `json:"categories,omitempty"`
}

// Validate checks that required fields are valid on CreateTimeTrackingEntryRequest
func (r *CreateTimeTrackingEntryRequest) Validate() error {
	if r.DurationMinutes <= 0 {
		return fmt.Errorf("duration_minutes must be greater than 0")
	}
	return nil
}

// UpdateTimeTrackingEntryRequest represents a request to update a time tracking entry.
type UpdateTimeTrackingEntryRequest struct {
	// Optional: Time in minutes tracked by the entry.
	DurationMinutes int `json:"duration_minutes,omitempty"`

	// Optional: The day that this entry is logged on.
	EnteredOn *Date `json:"entered_on,omitempty"`

	// Optional: The gid of the project which the time is attributable to.
	AttributableTo string `json:"attributable_to,omitempty"`

	// Optional: The gid of the user which the time is attributed to.
	Assignee string `json:"assignee,omitempty"`

	// Optional: The billable status of the entry. Requires Timesheets and Budgets.
	BillableStatus BillableStatus `json:"billable_status,omitempty"`

	// Optional: The description of the entry. Requires Timesheets and Budgets.
	Description string `json:"description,omitempty"`

	// Optional: Time tracking category gids to assign to the entry.
	// Requires Timesheets and Budgets.
	Categories []string `json:"categories,omitempty"`
}

// Validate checks that updated fields are valid on UpdateTimeTrackingEntryRequest
func (r *UpdateTimeTrackingEntryRequest) Validate() error {
	if r.DurationMinutes < 0 {
		return fmt.Errorf("duration_minutes must not be negative")
	}
	return nil
}

// Fetch loads the full details for this TimeTrackingEntry
func (e *TimeTrackingEntry) Fetch(client *Client, opts ...*Options) error {
	client.trace("Loading time tracking entry details for %q", e.ID)

	_, err := client.get(fmt.Sprintf("/time_tracking_entries/%s", e.ID), nil, e, opts...)
	return err
}

// Update applies new values to this TimeTrackingEntry
func (e *TimeTrackingEntry) Update(client *Client, request *UpdateTimeTrackingEntryRequest, opts ...*Options) error {
	client.trace("Updating time tracking entry %q", e.ID)

	return client.put(fmt.Sprintf("/time_tracking_entries/%s", e.ID), request, e, opts...)
}

// Delete removes this TimeTrackingEntry
func (e *TimeTrackingEntry) Delete(client *Client) error {
	client.info("Deleting time tracking entry %q", e.ID)

	return client.delete(fmt.Sprintf("/time_tracking_entries/%s", e.ID))
}

// TimeTrackingEntries returns a page of time tracking entries for this task
func (t *Task) TimeTrackingEntries(client *Client, opts ...*Options) ([]*TimeTrackingEntry, *NextPage, error) {
	client.trace("Listing time tracking entries for task %q", t.ID)
	var result []*TimeTrackingEntry

	nextPage, err := client.get(fmt.Sprintf("/tasks/%s/time_tracking_entries", t.ID), nil, &result, opts...)
	return result, nextPage, err
}

// CreateTimeTrackingEntry logs a new time tracking entry on this task
func (t *Task) CreateTimeTrackingEntry(client *Client, request *CreateTimeTrackingEntryRequest, opts ...*Options) (*TimeTrackingEntry, error) {
	client.info("Creating time tracking entry on task %q", t.ID)

	result := &TimeTrackingEntry{}
	err := client.post(fmt.Sprintf("/tasks/%s/time_tracking_entries", t.ID), request, result, opts...)
	return result, err
}
