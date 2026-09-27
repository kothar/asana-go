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
	must(t, fetched.Fetch(f.client))
	if fetched.Name != p.Name {
		t.Errorf("expected name %q, got %q", p.Name, fetched.Name)
	}
	if fetched.Workspace == nil || fetched.Workspace.ID != f.workspace.ID {
		t.Errorf("expected the project to be in workspace %s, got %+v", f.workspace.ID, fetched.Workspace)
	}

	const notes = "Updated by the asana-go integration tests"
	must(t, p.Update(f.client, &asana.UpdateProjectRequest{
		ProjectBase: asana.ProjectBase{Notes: notes},
	}))
	if p.Notes != notes {
		t.Errorf("expected Update to refresh notes to %q, got %q", notes, p.Notes)
	}
	must(t, fetched.Fetch(f.client))
	if fetched.Notes != notes {
		t.Errorf("expected fetched notes %q, got %q", notes, fetched.Notes)
	}

	projects := mustReturn(f.workspace.AllProjects(f.client))(t)
	if !contains(projects, p.ID) {
		t.Errorf("expected AllProjects to include %s", p.ID)
	}

	if f.team != nil {
		projects := mustReturn(f.team.AllProjects(f.client))(t)
		if !contains(projects, p.ID) {
			t.Errorf("expected the team's projects to include %s", p.ID)
		}
	}

	memberships := mustPage(p.Memberships(f.client))(t)
	if !contains(members(memberships), f.me.ID) {
		t.Errorf("expected the project's creator %s to be a member, got %d memberships", f.me.ID, len(memberships))
	}

	must(t, p.Delete(f.client))
	if err := fetched.Fetch(f.client); !asana.IsNotFoundError(err) {
		t.Errorf("expected a not found error fetching a deleted project, got %v", err)
	}
}

func TestSections(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")

	first := mustReturn(p.CreateSection(f.client, &asana.SectionBase{Name: "First"}))(t)
	second := mustReturn(p.CreateSection(f.client, &asana.SectionBase{Name: "Second"}))(t)

	fetched := &asana.Section{ID: first.ID}
	must(t, fetched.Fetch(f.client))
	if fetched.Name != "First" {
		t.Errorf("expected section name %q, got %q", "First", fetched.Name)
	}
	if fetched.Project == nil || fetched.Project.ID != p.ID {
		t.Errorf("expected the section to be in project %s, got %+v", p.ID, fetched.Project)
	}

	sections := mustPage(p.Sections(f.client))(t)
	if i, j := indexOf(sections, first.ID), indexOf(sections, second.ID); i < 0 || j < 0 || i > j {
		t.Errorf("expected both sections in creation order, found them at %d and %d", i, j)
	}

	renamed := mustReturn(second.Update(f.client, &asana.UpdateSectionRequest{
		SectionBase: asana.SectionBase{Name: "Renamed"},
	}))(t)
	if renamed.Name != "Renamed" {
		t.Errorf("expected the section to be renamed, got %q", renamed.Name)
	}

	must(t, p.InsertSection(f.client, &asana.SectionInsertRequest{
		Section:       second.ID,
		BeforeSection: first.ID,
	}))
	sections = mustPage(p.Sections(f.client))(t)
	if i, j := indexOf(sections, first.ID), indexOf(sections, second.ID); j < 0 || i < 0 || j > i {
		t.Errorf("expected the second section to move before the first, found them at %d and %d", i, j)
	}

	must(t, second.Delete(f.client))
	sections = mustPage(p.Sections(f.client))(t)
	if contains(sections, second.ID) {
		t.Error("expected the deleted section to be gone")
	}
}
