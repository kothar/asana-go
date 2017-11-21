package asana

import (
	"fmt"
	"time"
)

// TaskQuery specifies which tasks to return from QueryTasks
type TaskQuery struct {
	// The assignee to filter tasks on.
	//
	// Note: If you specify assignee, you must also specify the workspace to filter on.
	//
	// May be a string ('me', 'me@example.com') or an integer
	Assignee interface{} `url:"assignee,omitempty"`

	// The workspace or organization to filter tasks on.
	//
	// Note: If you specify workspace, you must also specify the assignee to filter on.
	Workspace uint64 `url:"workspace,omitempty"`

	// Only return tasks that are either incomplete or that have been completed since this time.
	//
	// May be 'now' or a date string
	CompletedSince string `url:"completed_since,omitempty"`

	// Only return tasks that have been modified since the given time.
	//
	// Note: A task is considered “modified” if any of its properties change,
	// or associations between it and other objects are modified (e.g. a task
	// being added to a project). A task is not considered modified just
	// because another object it is associated with (e.g. a subtask) is
	// modified. Actions that count as modifying the task include assigning,
	// renaming, completing, and adding stories.
	//
	// May be 'now' or a date string
	ModifiedSince string `url:"modified_since,omitempty"`
}

// Membership describes projects a task is associated with and the section it
// is in.
type Membership struct {
	Project *Project `json:"project,omitempty"`
	Section *Task    `json:"section,omitempty"`
}

// ExternalData allows a client application to add app-specific metadata to
// Tasks in the API. The custom data includes a string id that can be used to
// retrieve objects and a data blob that can store character strings.
//
// The blob may store unicode-safe serialized data such as JSON or YAML. The
// external id is capped at 1,024 characters, while data blobs are capped at
// 32,768 characters. Each object supporting external data can have one id and
// one data blob stored with it. You can also use either or both of those
// fields.
//
// The external id field is a good choice to create a reference between a
// resource in Asana and another database, such as cross-referencing an Asana
// task with a customer record in a CRM, or a bug in a dedicated bug tracker.
// Since it is just a unicode string, this field can store numeric IDs as well
// as URIs, however, when using URIs extra care must be taken when forming
// queries that the parameter is escaped correctly. By assigning an external
// id you can use the notation external:custom_id to reference your object
// anywhere that you may use the original object id.
//
// Note: that you will need to authenticate with Oauth, as the id and data are
// app-specific, and these fields are not visible in the UI. This also means
// that external data set by one Oauth app will be invisible to all other
// Oauth apps. However, the data is visible to all users of the same app that
// can view the resource to which the data is attached, so this should not be
// used for private user data.
type ExternalData struct {
	ID   string `json:"id,omitempty"`
	Data string `json:"data,omitempty"`
}

// TaskBase contains the modifiable fields for the Task object
type TaskBase struct {
	WithName
	WithNotes

	// User to which this task is assigned, or null if the task is unassigned.
	Assignee *User `json:"assignee,omitempty"`

	// Scheduling status of this task for the user it is assigned to. This
	// field can only be set if the assignee is non-null.
	AssigneeStatus string `json:"assignee_status,omitempty"`

	// True if the task is currently marked complete, false if not.
	Completed bool `json:"completed,omitempty"`

	// Date on which this task is due, or null if the task has no due date.
	// This takes a date with YYYY-MM-DD format and should not be used
	// together with due_at.
	DueOn *Date `json:"due_on,omitempty"`

	// Date and time on which this task is due, or null if the task has no due
	// time. This takes a UTC timestamp and should not be used together with
	// due_on.
	DueAt *time.Time `json:"due_at,omitempty"`

	// Oauth Required. The external field allows you to store app-specific
	// metadata on tasks, including an id that can be used to retrieve tasks
	// and a data blob that can store app-specific character strings. Note
	// that you will need to authenticate with Oauth to access or modify this
	// data. Once an external id is set, you can use the notation
	// external:custom_id to reference your object anywhere in the API where
	// you may use the original object id. See the page on Custom External
	// Data for more details.
	External *ExternalData `json:"external,omitempty"`
}

// Validate checks the task data and fixes any problems
func (t *TaskBase) Validate() error {
	if t.Assignee == nil {
		t.AssigneeStatus = ""
	}

	if t.DueAt != nil {
		t.DueOn = nil
	}
	return nil
}

// NewTask represents a request to create a new Task
type NewTask struct {
	TaskBase

	// TODO CustomFields
	Workspace int64   `json:"workspace,omitempty"`
	Parent    int64   `json:"parent,omitempty"`
	Projects  []int64 `json:"projects,omitempty"`
	Tags      []int64 `json:"tags,omitempty"`
}

// Task is the basic object around which many operations in Asana are
// centered. In the Asana application, multiple tasks populate the middle pane
// according to some view parameters, and the set of selected tasks determines
// the more detailed information presented in the details pane.
//
// A section, at its core, is a task whose name ends with the colon character
// :. Sections are unique in that they will be included in the memberships
// field of task objects returned in the API when the task is within a
// section. As explained below they can also be used to manipulate the
// ordering of a task within a project.
//
// Queries return a compact representation of each object which is typically
// the id and name fields. Interested in a specific set of fields or all of
// the fields? Use field selectors to manipulate what data is included in a
// response.
type Task struct {
	Expandable
	TaskBase

	WithParent
	WithDates
	WithWorkspace
	WithHearts
	WithFollowers

	// Read-only. The time at which this task was completed, or null if the
	// task is incomplete.
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Array of custom fields applied to the task. These custom fields
	// represent the values recorded on this task for a particular custom
	// field. For example, these fields will contain an enum_value property
	// for custom fields of type enum, a string_value property for custom
	// fields of type string, and so on. Please note that the id returned on
	// each custom field value is identical to the id of the custom field,
	// which allows referencing the custom field metadata through the
	// /custom_fields/custom_field-id endpoint.
	CustomFields []*CustomFieldValue `json:"custom_fields,omitempty"`

	// Create-only. Array of projects this task is associated with. At task
	// creation time, this array can be used to add the task to many projects
	// at once. After task creation, these associations can be modified using
	// the addProject and removeProject endpoints.
	Projects []*Project `json:"projects,omitempty"`

	// Create-only. Array of projects this task is associated with and the
	// section it is in. At task creation time, this array can be used to add
	// the task to specific sections. After task creation, these associations
	// can be modified using the addProject and removeProject endpoints. Note
	// that over time, more types of memberships may be added to this
	// property.
	Memberships []*Membership `json:"memberships,omitempty"`

	// Create-only. Array of tags associated with this task. This property may
	// be specified on creation using just an array of tag IDs. In order to
	// change tags on an existing task use addTag and removeTag.
	Tags []*Tag `json:"tags,omitempty"`
}

// Task creates an unexpanded Task object with the given ID
func (c *Client) Task(id int64) *Task {
	result := &Task{}
	result.init(id, c)
	return result
}

// Expand loads the full details for this Task
func (t *Task) Expand() error {
	t.trace("Loading task details for %q", t.Name)

	if t.expanded {
		return nil
	}

	_, err := t.client.get(fmt.Sprintf("/tasks/%d", t.ID), nil, t)
	return err
}

// Tasks returns a list of tasks in this project
func (p *Project) Tasks(opts ...*Options) ([]*Task, *NextPage, error) {
	p.trace("Listing tasks in %q", p.Name)
	var result []*Task

	// Make the request
	nextPage, err := p.client.get(fmt.Sprintf("/projects/%d/tasks", p.ID), nil, &result, opts...)
	return result, nextPage, err
}

// Subtasks returns a list of tasks in this project
func (t *Task) Subtasks(opts ...*Options) ([]*Task, *NextPage, error) {
	t.trace("Listing subtasks for %q", t.Name)

	var result []*Task

	// Make the request
	nextPage, err := t.client.get(fmt.Sprintf("/tasks/%d/subtasks", t.ID), nil, &result, opts...)
	return result, nextPage, err
}

// CreateTask creates a new task in the given project
func (c *Client) CreateTask(task *NewTask) (*Task, error) {
	c.info("Creating task %q", task.Name)

	result := &Task{}
	result.expanded = true

	err := c.post("/tasks", task, result)
	return result, err
}

// CreateSubtask creates a new task as a subtask of this task
func (t *Task) CreateSubtask(task *Task) (*Task, error) {
	t.info("Creating subtask %q", task.Name)

	result := &Task{}
	result.expanded = true

	err := t.client.post(fmt.Sprintf("/tasks/%d/subtasks", t.ID), task, result)
	return result, err
}

// QueryTasks returns the compact task records for some filtered set of tasks.
// Use one or more of the parameters provided to filter the tasks returned.
// You must specify a project or tag if you do not specify assignee and workspace.
func (c *Client) QueryTasks(query *TaskQuery, opts ...*Options) ([]*Task, *NextPage, error) {
	var result []*Task

	nextPage, err := c.get("/tasks", query, &result, opts...)
	return result, nextPage, err
}
