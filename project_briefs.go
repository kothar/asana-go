package asana

import "fmt"

// ProjectBriefBase contains the modifiable fields for a ProjectBrief
type ProjectBriefBase struct {
	// The title of the project brief.
	Title string `json:"title,omitempty"`

	// HTML formatted text for the project brief.
	HTMLText string `json:"html_text,omitempty"`

	// The plain text of the project brief. When writing to a project brief,
	// you can specify either html_text (preferred) or text, but not both.
	Text string `json:"text,omitempty"`
}

// ProjectBriefRequest represents a request to create or update a project brief
type ProjectBriefRequest struct {
	ProjectBriefBase
}

// Validate rejects ambiguous content rather than choosing one representation.
func (r *ProjectBriefRequest) Validate() error {
	if r == nil {
		return fmt.Errorf("a project brief request is required")
	}
	if r.Text != "" && r.HTMLText != "" {
		return fmt.Errorf("project brief text and html_text are mutually exclusive")
	}
	return nil
}

// ProjectBrief represents the rich-text summary on a project's Overview tab.
type ProjectBrief struct {
	// Read-only. Globally unique ID of the object
	ID string `json:"gid,omitempty"`

	ProjectBriefBase

	// Read-only. A url that points directly to the object within Asana.
	PermalinkURL string `json:"permalink_url,omitempty"`

	// Read-only. The project with which this project brief is associated.
	Project *Project `json:"project,omitempty"`
}

func (b *ProjectBrief) GetID() string {
	return b.ID
}

// Fetch loads the full details for this ProjectBrief
func (b *ProjectBrief) Fetch(client *Client, opts ...*Options) error {
	client.trace("Loading project brief details for %q", b.ID)

	_, err := client.get(fmt.Sprintf("/project_briefs/%s", b.ID), nil, b, opts...)
	return err
}

// CreateProjectBrief creates a new project brief on this project
func (p *Project) CreateProjectBrief(client *Client, request *ProjectBriefRequest, opts ...*Options) (*ProjectBrief, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	client.info("Creating project brief %q on project %q", request.Title, p.ID)

	result := &ProjectBrief{}
	err := client.post(fmt.Sprintf("/projects/%s/project_briefs", p.ID), request, result, opts...)
	return result, err
}

// Update applies new values to a ProjectBrief record
func (b *ProjectBrief) Update(client *Client, request *ProjectBriefRequest, opts ...*Options) error {
	client.trace("Updating project brief %q", b.ID)

	return client.put(fmt.Sprintf("/project_briefs/%s", b.ID), request, b, opts...)
}

// Delete removes this project brief
func (b *ProjectBrief) Delete(client *Client) error {
	client.info("Deleting project brief %q", b.ID)

	return client.delete(fmt.Sprintf("/project_briefs/%s", b.ID))
}
