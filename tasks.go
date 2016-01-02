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
	HasName
	HasNotes

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
	return nil
}

// NewTask represents a request to create a new Task
type NewTask struct {
	TaskBase

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
	TaskBase

	HasID
	HasParent
	HasDates
	HasWorkspace
	HasHearts
	HasFollowers

	expandable

	// Read-only. The time at which this task was completed, or null if the
	// task is incomplete.
	CompletedAt *time.Time `json:"completed_at,omitempty"`

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

// Task retrieves a task record by ID
func (c *Client) Task(id int64) (*Task, error) {
	c.trace("Loading task %d", id)
	result := &Task{}
	result.expanded = true

	err := c.get(fmt.Sprintf("/tasks/%d", id), nil, result)
	return result, err
}

// Expand loads the full details for this Task
func (t *Task) Expand() error {
	t.trace("Loading task details for %q", t.Name)

	if t.expanded {
		return nil
	}

	e, err := t.Client.Task(t.ID)
	if err != nil {
		return err
	}

	*t = *e
	return nil
}

// Tasks returns a list of tasks in this project
func (p *Project) Tasks(opts ...*Options) ([]*Task, error) {
	p.trace("Listing tasks in %q", p.Name)
	var result []*Task

	// Make the request
	err := p.Client.get(fmt.Sprintf("/projects/%d/tasks", p.ID), nil, &result, opts...)
	return result, err
}

// Subtasks returns a list of tasks in this project
func (t *Task) Subtasks(opts ...*Options) ([]*Task, error) {
	t.trace("Listing subtasks for %q", t.Name)

	var result []*Task

	// Make the request
	err := t.Client.get(fmt.Sprintf("/tasks/%d/subtasks", t.ID), nil, &result, opts...)
	return result, err
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

	err := t.Client.post(fmt.Sprintf("/tasks/%d/subtasks", t.ID), task, result)
	return result, err
}

// QueryTasks finds matching tasks visible to the current client account
func (c *Client) QueryTasks(query *TaskQuery) ([]*Task, error) {
	var result []*Task

	err := c.get("/tasks", &result, query)
	return result, err
}
