package asana

import (
	"encoding/json"
	"fmt"
	"time"
)

// TaskQuery specifies which tasks to return from QueryTasks
type TaskQuery struct {
	// The assignee to filter tasks on.
	//
	// Note: If you specify assignee, you must also specify the workspace to filter on.
	//
	// May be a GID, 'me' or user email string ('14113', 'me', 'me@example.com')
	Assignee string `url:"assignee,omitempty"`

	// The project to filter tasks on
	Project string `url:"project,omitempty"`

	// The section to filter tasks on.
	//
	// Note: Currently, this is only supported in board views.
	Section string `url:"section,omitempty"`

	// The workspace or organization to filter tasks on.
	//
	// Note: If you specify workspace, you must also specify the assignee to filter on.
	Workspace string `url:"workspace,omitempty"`

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
	Section *Section `json:"section,omitempty"`
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
	ID   string `json:"gid,omitempty"`
	Data string `json:"data,omitempty"`
}

// TaskBase contains the modifiable fields for the Task object
type TaskBase struct {
	// Name of the task. This is generally a short sentence fragment that
	// fits on a line in the UI for maximum readability. However, it can be longer.
	Name string `json:"name,omitempty"`

	// The type of task. Different subtypes of tasks retain many of
	// the same fields and behavior, but may render differently in Asana or
	// represent tasks with different semantic meaning.
	ResourceSubtype string `json:"resource_subtype,omitempty"`

	// More detailed, free-form textual information associated with the
	// task.
	Notes string `json:"notes,omitempty"`

	// The notes of the text with formatting as HTML.
	HTMLNotes string `json:"html_notes,omitempty"`

	// Scheduling status of this task for the user it is assigned to. This
	// field can only be set if the assignee is non-null.
	AssigneeStatus string `json:"assignee_status,omitempty"`

	// True if the task is currently marked complete, false if not.
	Completed *bool `json:"completed,omitempty"`

	// Date on which this task is due, or null if the task has no due date.
	// This takes a date with YYYY-MM-DD format and should not be used
	// together with due_at.
	DueOn *Date `json:"due_on,omitempty"`

	// Date and time on which this task is due, or null if the task has no due
	// time. This takes a UTC timestamp and should not be used together with
	// due_on.
	DueAt *time.Time `json:"due_at,omitempty"`

	// Date on which this task is due, or null if the task has no start date.
	// This field takes a date with YYYY-MM-DD format.
	// Note: due_on or due_at must be present in the request when setting or
	// unsetting the start_on parameter.
	StartOn *Date `json:"start_on,omitempty"`

	// Oauth Required. The external field allows you to store app-specific
	// metadata on tasks, including an id that can be used to retrieve tasks
	// and a data blob that can store app-specific character strings. Note
	// that you will need to authenticate with Oauth to access or modify this
	// data. Once an external id is set, you can use the notation
	// external:custom_id to reference your object anywhere in the API where
	// you may use the original object id. See the page on Custom External
	// Data for more details.
	External *ExternalData `json:"external,omitempty"`

	// Indicates whether a default task is rendered as bolded and underlined
	// when viewed in a list of subtasks or in a user’s My Tasks.
	// Requires that the NewSections deprecation is enabled.
	IsRenderedAsSeparator bool `json:"is_rendered_as_separator,omitempty"`
}

// Validate checks the task data and fixes any problems
func (t *CreateTaskRequest) Validate() error {
	if t.Assignee == "" {
		t.AssigneeStatus = ""
	}

	if t.DueAt != nil {
		t.DueOn = nil
	}
	return nil
}

// CreateTaskRequest represents a request to create a new Task
type CreateTaskRequest struct {
	TaskBase

	Assignee  string   `json:"assignee,omitempty"`  // User to which this task is assigned, or null if the task is unassigned.
	Followers []string `json:"followers,omitempty"` // Array of users following this task.

	Workspace    string                 `json:"workspace,omitempty"`
	Parent       string                 `json:"parent,omitempty"`
	Projects     []string               `json:"projects,omitempty"`
	Memberships  []*CreateMembership    `json:"memberships,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

type CreateMembership struct {
	Project string `json:"project"`
	Section string `json:"section"`
}

type UpdateTaskRequest struct {
	TaskBase

	Assignee  string   `json:"assignee,omitempty"`  // User to which this task is assigned, or null if the task is unassigned.
	Followers []string `json:"followers,omitempty"` // Array of users following this task.
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
	client *Client

	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	TaskBase

	// Read-only. The task this object is attached to.
	Parent *Task `json:"parent,omitempty"`

	// Read-only. The time at which this object was created.
	CreatedAt *time.Time `json:"created_at,omitempty"`

	// Read-only. The time at which this object was last modified.
	//
	// Note: This does not currently reflect any changes in associations such
	// as tasks or comments that may have been added or removed from the
	// object.
	ModifiedAt *time.Time `json:"modified_at,omitempty"`

	// Create-only. The workspace or organization this object is associated
	// with. Once created, objects cannot be moved to a different workspace.
	// This attribute can only be specified at creation time.
	Workspace *Workspace `json:"workspace,omitempty"`

	// True if the task is liked by the authorized user, false if not.
	Liked bool `json:"liked,omitempty"`

	// Read-only. Array of users who have liked this task.
	Likes []*User `json:"likes,omitempty"`

	// Read-only. The number of users who have liked this task.
	NumLikes int32 `json:"num_likes,omitempty"`

	// Read-only. Opt In. The number of subtasks on this task.
	NumSubtasks int32 `json:"num_subtasks,omitempty"`

	// Read-only. Array of users following this task. Followers are a
	// subset of members who receive all notifications for a project, the
	// default notification setting when adding members to a project in-
	// product.
	Followers []*User `json:"followers,omitempty"`

	// User to which this task is assigned, or null if the task is unassigned.
	Assignee *User `json:"assignee,omitempty"`

	// Scheduling status of this task for the user it is assigned to. This
	// field can only be set if the assignee is non-null.
	AssigneeStatus string `json:"assignee_status,omitempty"`

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

	// Read-only. Array of resources referencing tasks that this task depends on.
	// The objects contain only the ID of the dependency.
	Dependencies []*Task `json:"dependencies,omitempty"`

	// Read-only. Array of resources referencing tasks that depend on this task.
	// The objects contain only the ID of the dependent.
	Dependents []*Task `json:"dependents,omitempty"`
}

// Fetch loads the full details for this Task
func (t *Task) Fetch(client *Client) error {
	client.trace("Loading task details for %q", t.Name)

	_, err := client.get(fmt.Sprintf("/tasks/%s", t.ID), nil, t)
	return err
}

// Update applies new values to a Task record
func (t *Task) Update(update *UpdateTaskRequest) error {
	t.client.trace("Updating task %q", t.Name)

	err := t.client.put(fmt.Sprintf("/tasks/%s", t.ID), update, t)
	return err
}

func (t *Task) Delete(client *Client) error {
	client.info("Deleting task %q", t.Name)

	return client.delete(fmt.Sprintf("/tasks/%s", t.ID))
}

// AddProjectRequest defines the location a task should be added to a project
type AddProjectRequest struct {
	Project      string // Required: The project to add the task to.
	InsertAfter  string // A task in the project to insert the task after, or "-" to insert at the beginning of the list.
	InsertBefore string // A task in the project to insert the task before, or "-" to insert at the end of the list.
	Section      string // A section in the project to insert the task into. The task will be inserted at the bottom of the section.
}

// AddProject adds this task to an existing project at the provided location
func (t *Task) AddProject(request *AddProjectRequest) error {
	t.client.trace("Adding task %q to project %q", t.ID, request.Project)

	// Custom encoding of Insert fields needed
	m := map[string]interface{}{
		"project": request.Project,
	}

	if request.InsertAfter == "-" {
		m["insert_after"] = nil
	} else if request.InsertAfter != "" {
		m["insert_after"] = request.InsertAfter
	}

	if request.InsertBefore == "-" {
		m["insert_before"] = nil
	} else if request.InsertBefore != "" {
		m["insert_before"] = request.InsertBefore
	}

	if request.Section != "" {
		m["section"] = request.Section
	}

	err := t.client.post(fmt.Sprintf("/tasks/%s/addProject", t.ID), m, &json.RawMessage{})
	return err
}

func (t *Task) RemoveProject(projectID string) error {
	t.client.trace("Removing task %q from project %q", t.ID, projectID)

	// Custom encoding of Insert fields needed
	m := map[string]interface{}{
		"project": projectID,
	}

	err := t.client.post(fmt.Sprintf("/tasks/%s/removeProject", t.ID), m, &json.RawMessage{})
	return err
}

// SetParentRequest changes the parent of a task. Each task may only be a subtask of a single parent, or no parent task at all.
// When using insert_before and insert_after, at most one of those two options can be specified, and they must already be subtasks of the parent.
type SetParentRequest struct {
	Parent       string // Required: The new parent of the task, or null for no parent.
	InsertAfter  string // A subtask of the parent to insert the task after, or "-" to insert at the beginning of the list.
	InsertBefore string // A subtask of the parent to insert the task before, or "-" to insert at the end of the list.
}

// SetParent changes the parent of a task
func (t *Task) SetParent(request *SetParentRequest) error {
	t.client.trace("Setting the parent of task %q to %q", t.ID, request.Parent)

	// Custom encoding of Insert fields needed
	m := map[string]interface{}{
		"parent": request.Parent,
	}

	if request.InsertAfter == "-" {
		m["insert_after"] = nil
	} else if request.InsertBefore == "-" {
		m["insert_before"] = nil
	} else if request.InsertAfter != "" {
		m["insert_after"] = request.InsertAfter
	} else if request.InsertBefore != "" {
		m["insert_before"] = request.InsertBefore
	}

	err := t.client.post(fmt.Sprintf("/tasks/%s/setParent", t.ID), m, &json.RawMessage{})
	return err
}

// AddDependenciesRequest
type AddDependenciesRequest struct {
	// Required: An array of task IDs that this task should depend on.
	Dependencies []string `json:"dependencies"`
}

// AddDependencies marks a set of tasks as dependencies of this task, if they
// are not already dependencies. A task can have at most 15 dependencies.
func (t *Task) AddDependencies(request *AddDependenciesRequest) error {
	t.client.trace("Adding dependencies to task %q", t.ID)

	err := t.client.post(fmt.Sprintf("/tasks/%s/addDependencies", t.ID), request, &json.RawMessage{})
	return err
}

// AddDependentsRequest
type AddDependentsRequest struct {
	// Required: An array of task IDs that this task should depend on.
	Dependents []string `json:"dependents"`
}

// AddDependents marks a set of tasks as dependents of this task, if they
// are not already dependents. A task can have at most 30 dependents.
func (t *Task) AddDependents(request *AddDependentsRequest) error {
	t.client.trace("Adding dependents to task %q", t.ID)

	err := t.client.post(fmt.Sprintf("/tasks/%s/addDependents", t.ID), request, &json.RawMessage{})
	return err
}

// Tasks returns a list of tasks in this project
func (p *Project) Tasks(opts ...*Options) ([]*Task, *NextPage, error) {
	p.client.trace("Listing tasks in %q", p.Name)
	var result []*Task

	// Make the request
	nextPage, err := p.client.get(fmt.Sprintf("/projects/%s/tasks", p.ID), nil, &result, opts...)
	for _, r := range result {
		r.client = p.client
	}
	return result, nextPage, err
}

// Tasks returns a list of tasks in this section. Board view only.
func (s *Section) Tasks(opts ...*Options) ([]*Task, *NextPage, error) {
	s.client.trace("Listing tasks in %q", s.Name)
	var result []*Task

	// Make the request
	nextPage, err := s.client.get(fmt.Sprintf("/sections/%s/tasks", s.ID), nil, &result, opts...)
	for _, r := range result {
		r.client = s.client
	}
	return result, nextPage, err
}

// Subtasks returns a list of tasks in this project
func (t *Task) Subtasks(opts ...*Options) ([]*Task, *NextPage, error) {
	t.client.trace("Listing subtasks for %q", t.Name)

	var result []*Task

	// Make the request
	nextPage, err := t.client.get(fmt.Sprintf("/tasks/%s/subtasks", t.ID), nil, &result, opts...)
	for _, r := range result {
		r.client = t.client
	}
	return result, nextPage, err
}

// CreateTask creates a new task in the given project
func (c *Client) CreateTask(task *CreateTaskRequest) (*Task, error) {
	c.info("Creating task %q", task.Name)

	result := &Task{}

	err := c.post("/tasks", task, result)
	result.client = c
	return result, err
}

// CreateSubtask creates a new task as a subtask of this task
func (t *Task) CreateSubtask(task *Task) (*Task, error) {
	t.client.info("Creating subtask %q", task.Name)

	result := &Task{}

	err := t.client.post(fmt.Sprintf("/tasks/%s/subtasks", t.ID), task, result)
	result.client = t.client
	return result, err
}

// QueryTasks returns the compact task records for some filtered set of tasks.
// Use one or more of the parameters provided to filter the tasks returned.
// You must specify a project or tag if you do not specify assignee and workspace.
func (c *Client) QueryTasks(query *TaskQuery, opts ...*Options) ([]*Task, *NextPage, error) {
	var result []*Task

	nextPage, err := c.get("/tasks", query, &result, opts...)
	for _, r := range result {
		r.client = c
	}
	return result, nextPage, err
}
