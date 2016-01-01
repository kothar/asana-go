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
	return []byte(s), nil
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

// HasID is a mixin for objects with an ID
type HasID struct {
	// Read-only. Globally unique ID of the object
	ID int64 `json:"id,omitempty"`
}

// HasName is a mixin for objects with a human-readable name
type HasName struct {
	// Read-only. The name of the object.
	Name string `json:"name,omitempty"`
}

// HasParent is a mixin for objects which are children of a Task object
type HasParent struct {
	// Read-only. The task this object is attached to.
	Parent *Task `json:"parent,omitempty"`
}

// HasCreated is a mixin for objects with a creation date
type HasCreated struct {
	// Read-only. The time at which this object was created.
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// HasDates is a mixin for objects with creation and modification dates
type HasDates struct {
	HasCreated

	// Read-only. The time at which this object was last modified.
	//
	// Note: This does not currently reflect any changes in associations such
	// as tasks or comments that may have been added or removed from the
	// object.
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
}

// HasNotes is a mixin for objects with notes attached
type HasNotes struct {
	// More detailed, free-form textual information associated with the
	// object.
	Notes string `json:"notes,omitempty"`
}

// HasWorkspace is a mixin for objects which define the workspace they belong to
type HasWorkspace struct {
	// Create-only. The workspace or organization this object is associated
	// with. Once created, objects cannot be moved to a different workspace.
	// This attribute can only be specified at creation time.
	Workspace *Workspace `json:"workspace,omitempty"`
}

// HasHearts is a mixin for objects which may be 'hearted'
type HasHearts struct {
	// True if the object is hearted by the authorized user, false if not.
	Hearted bool `json:"hearted,omitempty"`

	// Read-only. Array of users who have hearted this object.
	Hearts []*User `json:"hearts,omitempty"`

	// Read-only. The number of users who have hearted this object.
	NumHearts int32 `json:"num_hearts,omitempty"`
}

// HasFollowers is a mixin for objects which may have followers
type HasFollowers struct {
	// Read-only. Array of users following this project. Followers are a
	// subset of members who receive all notifcations for a project, the
	// default notification setting when adding members to a project in-
	// product.
	Followers []*User `json:"followers,omitempty"`
}

// HasColor is a mixin for objects with a color field
type HasColor struct {
	// Color of the object. Must be either null or one of: dark-pink, dark-
	// green, dark-blue, dark-red, dark-teal, dark-brown, dark-orange, dark-
	// purple, dark-warm-gray, light-pink, light-green, light-blue, light-red,
	// light-teal, light-yellow, light-orange, light-purple, light-warm-gray.
	Color string `json:"color,omitempty"`
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
	Pretty bool `json:"pretty,omitempty" url:"opt_pretty,omitempty"`

	// In environments that do not support the full range of HTTP verbs, this
	// can be helpful to override the request method sent by the browser,
	// allowing access to the full range of actions in the API. This should
	// only be used when absolutely necessary, as circumventing the browser’s
	// normal interpretation of the HTTP verbs can cause issues. For security
	// reasons, you cannot use this parameter from a GET request, you can only
	// use it to transform a POST request into something else (like a PUT or
	// DELETE).
	Method string `json:"method,omitempty" url:"-"`

	// Some requests return compact representations of objects, to conserve
	// resources and complete the request more efficiently. Other times
	// requests return more information than you may need. This option allows
	// you to list the exact set of fields that the API should be sure to
	// return for the objects. The field names should be provided as paths,
	// described below.   The id of included objects will always be returned,
	// regardless of the field options.
	Fields []string `json:"fields,omitempty" url:"opt_fields,omitempty"`

	// Query results and sub-objects are returned in compact form by default.
	// This option can be used to expand query results or sub-objects to
	// return more detailed information. Be sure you really need the
	// information in the expanded form, as executing a query with many
	// results in expanded form can be costly and return you a lot of data to
	// consume. If the fields option is also used, it will take precedence
	// over the expand option and prevent expansion.
	Expand []string `json:"expand,omitempty" url:"opt_expand,omitempty"`

	// Returns the output in JSON-P format instead of plain JSON, to allow
	// requests to come from within browsers and work around the “same origin
	// policy.” The function named as the value of the opt_jsonp parameter
	// will be called with a single argument, a JavaScript object representing
	// the response.
	JSONP string `json:"jsonp,omitempty" url:"opt_jsonp,omitempty"`
}

// Attachment represents any file attached to a task in Asana,
// whether it’s an uploaded file or one associated via a third-party service
// such as Dropbox or Google Drive.
type Attachment struct {
	HasID
	HasName
	HasParent
	HasCreated

	expandable

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

// Team is used to group related projects and people together within an
// organization. Each project in an organization is associated with a team.
type Team struct {
	HasID
	HasName

	expandable
}

// User represents an account in Asana that can be given access to various
// workspaces, projects, and tasks.
//
// Like other objects in the system, users are referred to by numerical IDs.
// However, the special string identifier me can be used anywhere a user ID is
// accepted, to refer to the current authenticated user.
type User struct {
	HasID
	HasName

	expandable

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

// Story represents an activity associated with an object in the Asana
// system. Stories are generated by the system whenever users take actions
// such as creating or assigning tasks, or moving tasks between projects.
// Comments are also a form of user-generated story.
//
// Stories are a form of history in the system, and as such they are read-
// only. Once generated, it is not possible to modify a story.
type Story struct {
	HasID
	HasCreated
	HasHearts

	expandable

	// The user who created the story.
	CreatedBy *User `json:"created_by,omitempty"`

	// Create-only. Human-readable text for the story or comment. This will
	// not include the name of the creator.
	//
	// Note: This is not guaranteed to be stable for a given type of story.
	// For example, text for a reassignment may not always say “assigned to
	// …”. The API currently does not provide a structured way of inspecting
	// the meaning of a story.
	Text string `json:"text,omitempty"`

	// Read-only. HTML formatted text for a comment. This will not include the
	// name of the creator.
	//
	// Note: This field is only returned if explicitly requested using the
	// opt_fields query parameter.
	HTMLText string `json:"html_text,omitempty"`

	// Read-only. The object this story is associated with. Currently may only
	// be a task.
	Target *Task `json:"target,omitempty"`

	// Read-only. The component of the Asana product the user used to trigger
	// the story.
	Source string `json:"source,omitempty"`

	// Read-only. The type of story this is.
	Type string `json:"type,omitempty"`
}
