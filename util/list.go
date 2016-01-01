package util

import (
	"fmt"

	"bitbucket.org/mikehouston/asana-go"
)

func ListWorkspaces(c *asana.Client) error {
	// List workspaces
	workspaces, err := c.Workspaces()
	if err != nil {
		return err
	}

	for _, workspace := range workspaces {
		if workspace.IsOrganization {
			fmt.Printf("Organization %d: %s\n", workspace.ID, workspace.Name)
		} else {
			fmt.Printf("Workspace %d: %s\n", workspace.ID, workspace.Name)
		}
	}
	return nil
}

func ListProjects(w *asana.Workspace) error {
	// List projects
	projects, err := w.Projects()
	if err != nil {
		return err
	}

	for _, project := range projects {
		fmt.Printf("Project %d: %s\n", project.ID, project.Name)
	}
	return nil
}

func ListTasks(p *asana.Project) error {
	// List projects
	tasks, err := p.Tasks()
	if err != nil {
		return err
	}

	for _, task := range tasks {
		fmt.Printf("Task %d: %s\n", task.ID, task.Name)
	}
	return nil
}
