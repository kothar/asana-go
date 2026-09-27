package integration

import (
	"testing"

	"github.com/kothar/asana-go"
)

func TestProjectLifecycle(t *testing.T) {
	f := setup(t)

	p := f.newProject(t, "project")
	if p.ID == "" {
		t.Fatal("expected the new project to have a gid")
	}

	fetched := &asana.Project{ID: p.ID}
	if err := fetched.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
	if fetched.Name != p.Name {
		t.Errorf("expected name %q, got %q", p.Name, fetched.Name)
	}
	if fetched.Workspace == nil || fetched.Workspace.ID != f.workspace.ID {
		t.Errorf("expected the project to be in workspace %s, got %+v", f.workspace.ID, fetched.Workspace)
	}

	const notes = "Updated by the asana-go integration tests"
	if err := p.Update(f.client, &asana.UpdateProjectRequest{
		ProjectBase: asana.ProjectBase{Notes: notes},
	}); err != nil {
		t.Fatal(err)
	}
	if p.Notes != notes {
		t.Errorf("expected Update to refresh notes to %q, got %q", notes, p.Notes)
	}
	if err := fetched.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
	if fetched.Notes != notes {
		t.Errorf("expected fetched notes %q, got %q", notes, fetched.Notes)
	}

	projects, err := f.workspace.AllProjects(f.client)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(projects, p.ID, projectID) {
		t.Errorf("expected AllProjects to include %s", p.ID)
	}

	if f.team != nil {
		projects, err := f.team.AllProjects(f.client)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(projects, p.ID, projectID) {
			t.Errorf("expected the team's projects to include %s", p.ID)
		}
	}

	memberships, _, err := p.Memberships(f.client)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(memberships, f.me.ID, memberID) {
		t.Errorf("expected the project's creator %s to be a member, got %d memberships", f.me.ID, len(memberships))
	}

	if err := p.Delete(f.client); err != nil {
		t.Fatal(err)
	}
	if err := fetched.Fetch(f.client); !asana.IsNotFoundError(err) {
		t.Errorf("expected a not found error fetching a deleted project, got %v", err)
	}
}

func TestSections(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")

	first, err := p.CreateSection(f.client, &asana.SectionBase{Name: "First"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := p.CreateSection(f.client, &asana.SectionBase{Name: "Second"})
	if err != nil {
		t.Fatal(err)
	}

	fetched := &asana.Section{ID: first.ID}
	if err := fetched.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
	if fetched.Name != "First" {
		t.Errorf("expected section name %q, got %q", "First", fetched.Name)
	}
	if fetched.Project == nil || fetched.Project.ID != p.ID {
		t.Errorf("expected the section to be in project %s, got %+v", p.ID, fetched.Project)
	}

	sections, _, err := p.Sections(f.client)
	if err != nil {
		t.Fatal(err)
	}
	if i, j := indexOf(sections, first.ID, sectionID), indexOf(sections, second.ID, sectionID); i < 0 || j < 0 || i > j {
		t.Errorf("expected both sections in creation order, found them at %d and %d", i, j)
	}

	renamed, err := second.Update(f.client, &asana.UpdateSectionRequest{
		SectionBase: asana.SectionBase{Name: "Renamed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "Renamed" {
		t.Errorf("expected the section to be renamed, got %q", renamed.Name)
	}

	if err := p.InsertSection(f.client, &asana.SectionInsertRequest{
		Section:       second.ID,
		BeforeSection: first.ID,
	}); err != nil {
		t.Fatal(err)
	}
	sections, _, err = p.Sections(f.client)
	if err != nil {
		t.Fatal(err)
	}
	if i, j := indexOf(sections, first.ID, sectionID), indexOf(sections, second.ID, sectionID); j < 0 || i < 0 || j > i {
		t.Errorf("expected the second section to move before the first, found them at %d and %d", i, j)
	}

	if err := second.Delete(f.client); err != nil {
		t.Fatal(err)
	}
	sections, _, err = p.Sections(f.client)
	if err != nil {
		t.Fatal(err)
	}
	if contains(sections, second.ID, sectionID) {
		t.Error("expected the deleted section to be gone")
	}
}
