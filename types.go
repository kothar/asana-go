package asana

import (
	"encoding/json"
	"time"
)

// Date wraps the default time.Time type with appropriate JSON formatting for
// the Asana API when a date is required: '2012-03-26'
type Date time.Time

// Jan 2 15:04:05 2006 MST
const dateLayout = "2006-01-02"

// MarshalJSON implements the json.Marshaller interface
func (d *Date) MarshalJSON() ([]byte, error) {
	s := time.Time(*d).Format(dateLayout)
	return json.Marshal(s)
}

// UnmarshalJSON implements the json.Unmarshaller interface
func (d *Date) UnmarshalJSON(value []byte) error {

	var dateString string
	if err := json.Unmarshal(value, &dateString); err != nil {
		return err
	}

	t, err := time.Parse(dateLayout, dateString)
	if err != nil {
		return err
	}

	*d = Date(t)
	return nil
}

// WithID is a mixin for objects with an ID
type WithID struct {
	// Read-only. Globally unique ID of the object
	ID int64 `json:"id,omitempty"`
}

// HasID objects have int64 IDs
type HasID interface {
	GetID() int64
}

// GetID returns the ID of an object
func (a *WithID) GetID() int64 {
	return a.ID
}

// Marks whether an object is partially loaded
type Expandable struct {
	WithID

	client   *Client
	expanded bool
}

func (e *Expandable) init(id int64, c *Client) {
	e.ID = id
	e.client = c
}

func (e *Expandable) Client() *Client {
	return e.client
}

func (e *Expandable) SetClient(c *Client) {
	e.client = c
}

// Validator types have a Validate method which is called before posting the
// data to the API
type Validator interface {
	Validate() error
}

// WithName is a mixin for objects with a human-readable name
type WithName struct {
	// Read-only. The name of the object.
	Name string `json:"name,omitempty" dynamo:"name"`
}

// WithParent is a mixin for objects which are children of a Task object
type WithParent struct {
	// Read-only. The task this object is attached to.
	Parent *Task `json:"parent,omitempty" dynamo:"parent"`
}

// WithCreated is a mixin for objects with a creation date
type WithCreated struct {
	// Read-only. The time at which this object was created.
	CreatedAt *time.Time `json:"created_at,omitempty" dynamo:"created_at"`
}

// WithDates is a mixin for objects with creation and modification dates
type WithDates struct {
	WithCreated

	// Read-only. The time at which this object was last modified.
	//
	// Note: This does not currently reflect any changes in associations such
	// as tasks or comments that may have been added or removed from the
	// object.
	ModifiedAt *time.Time `json:"modified_at,omitempty" dynamo:"modified_at"`
}

// WithNotes is a mixin for objects with notes attached
type WithNotes struct {
	// More detailed, free-form textual information associated with the
	// object.
	Notes string `json:"notes,omitempty" dynamo:"notes"`
}

// WithWorkspace is a mixin for objects which define the workspace they belong to
type WithWorkspace struct {
	// Create-only. The workspace or organization this object is associated
	// with. Once created, objects cannot be moved to a different workspace.
	// This attribute can only be specified at creation time.
	Workspace *Workspace `json:"workspace,omitempty" dynamo:"workspace"`
}

// WithHearts is a mixin for objects which may be 'hearted'
type WithHearts struct {
	// True if the object is hearted by the authorized user, false if not.
	Hearted bool `json:"hearted,omitempty" dynamo:"hearted"`

	// Read-only. Array of users who have hearted this object.
	Hearts []*User `json:"hearts,omitempty" dynamo:"hearts"`

	// Read-only. The number of users who have hearted this object.
	NumHearts int32 `json:"num_hearts,omitempty" dynamo:"num_hearts"`
}

// WithFollowers is a mixin for objects which may have followers
type WithFollowers struct {
	// Read-only. Array of users following this project. Followers are a
	// subset of members who receive all notifcations for a project, the
	// default notification setting when adding members to a project in-
	// product.
	Followers []*User `json:"followers,omitempty" dynamo:"followers"`
}

// WithColor is a mixin for objects with a color field
type WithColor struct {
	// Color of the object. Must be either null or one of: dark-pink, dark-
	// green, dark-blue, dark-red, dark-teal, dark-brown, dark-orange, dark-
	// purple, dark-warm-gray, light-pink, light-green, light-blue, light-red,
	// light-teal, light-yellow, light-orange, light-purple, light-warm-gray.
	Color string `json:"color,omitempty" dynamo:"color"`
}

// Options - In addition to providing fields and their values in a request,
// you may also specify options to control how your request is interpreted and
// how the response is generated. For GET requests, options are specified as
// URL parameters prefixed with opt_. For POST or PUT requests, options are
// specified in the body, inside the top-level options object (a sibling of
// the data object).
//
// These options can be used in combination in a single request, though some
// of them may conflict in their impact on the response.
type Options struct {

	// Provides the response in “pretty” output. In the case of JSON this
	// means doing proper line breaking and indentation to make it readable.
	// This will take extra time and increase the response size so it is
	// advisable only to use this during debugging.
	Pretty bool `json:"pretty,omitempty" url:"opt_pretty,omitempty" dynamo:"pretty"`

	// In environments that do not support the full range of HTTP verbs, this
	// can be helpful to override the request method sent by the browser,
	// allowing access to the full range of actions in the API. This should
	// only be used when absolutely necessary, as circumventing the browser’s
	// normal interpretation of the HTTP verbs can cause issues. For security
	// reasons, you cannot use this parameter from a GET request, you can only
	// use it to transform a POST request into something else (like a PUT or
	// DELETE).
	Method string `json:"method,omitempty" url:"-" dynamo:"method"`

	// Some requests return compact representations of objects, to conserve
	// resources and complete the request more efficiently. Other times
	// requests return more information than you may need. This option allows
	// you to list the exact set of fields that the API should be sure to
	// return for the objects. The field names should be provided as paths,
	// described below.   The id of included objects will always be returned,
	// regardless of the field options.
	Fields []string `json:"fields,omitempty" url:"opt_fields,omitempty" dynamo:"fields"`

	// Query results and sub-objects are returned in compact form by default.
	// This option can be used to expand query results or sub-objects to
	// return more detailed information. Be sure you really need the
	// information in the expanded form, as executing a query with many
	// results in expanded form can be costly and return you a lot of data to
	// consume. If the fields option is also used, it will take precedence
	// over the expand option and prevent expansion.
	Expand []string `json:"expand,omitempty" url:"opt_expand,omitempty" dynamo:"expand"`

	// Returns the output in JSON-P format instead of plain JSON, to allow
	// requests to come from within browsers and work around the “same origin
	// policy.” The function named as the value of the opt_jsonp parameter
	// will be called with a single argument, a JavaScript object representing
	// the response.
	JSONP string `json:"jsonp,omitempty" url:"opt_jsonp,omitempty" dynamo:"jsonp"`
}

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
	DownloadURL string `json:"download_url,omitempty" dynamo:"download_url"`

	// Read-only. The service hosting the attachment. Valid values are asana,
	// dropbox, gdrive and box.
	Host string `json:"host,omitempty" dynamo:"host"`

	// Read-only. The URL where the attachment can be viewed, which may be
	// friendlier to users in a browser than just directing them to a raw
	// file.
	ViewURL string `json:"view_url,omitempty" dynamo:"view_url"`
}

// Team is used to group related projects and people together within an
// organization. Each project in an organization is associated with a team.
type Team struct {
	Expandable
	WithName
}
