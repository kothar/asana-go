package integration

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kothar/asana-go"
)

func TestComments(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")
	task := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})

	comment, err := task.CreateComment(f.client, &asana.StoryBase{Text: "First comment"})
	if err != nil {
		t.Fatal(err)
	}
	if comment.ResourceSubtype != "comment_added" || comment.Text != "First comment" {
		t.Errorf("expected a comment_added story with the text, got %q %q", comment.ResourceSubtype, comment.Text)
	}
	if comment.CreatedBy == nil || comment.CreatedBy.ID != f.me.ID {
		t.Errorf("expected the comment to be created by %s, got %+v", f.me.ID, comment.CreatedBy)
	}

	stories, _, err := task.Stories(f.client)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(stories, comment.ID, storyID) {
		t.Error("expected the task's stories to include the comment")
	}

	updated, err := comment.UpdateStory(f.client, &asana.StoryBase{Text: "Edited comment"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Text != "Edited comment" {
		t.Errorf("expected the edited text, got %q", updated.Text)
	}

	if err := comment.Delete(f.client); err != nil {
		t.Fatal(err)
	}
	stories, _, err = task.Stories(f.client)
	if err != nil {
		t.Fatal(err)
	}
	if contains(stories, comment.ID, storyID) {
		t.Error("expected the deleted comment to be gone")
	}
}

func TestTags(t *testing.T) {
	f := setup(t)

	tag, err := f.workspace.CreateTag(f.client, &asana.TagBase{
		Name:  f.name(t, "tag"),
		Notes: "Created by the asana-go integration tests",
	})
	if err != nil {
		t.Fatal(err)
	}
	cleanup(t, "tag "+tag.ID, func() error { return tag.Delete(f.client) })

	fetched := &asana.Tag{ID: tag.ID}
	if err := fetched.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
	if fetched.Name != tag.Name {
		t.Errorf("expected tag name %q, got %q", tag.Name, fetched.Name)
	}
	if fetched.Workspace == nil || fetched.Workspace.ID != f.workspace.ID {
		t.Errorf("expected the tag to be in workspace %s, got %+v", f.workspace.ID, fetched.Workspace)
	}

	tags, err := f.workspace.AllTags(f.client)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(tags, tag.ID, tagID) {
		t.Error("expected AllTags to include the tag")
	}

	p := f.newProject(t, "project")
	task := f.newTask(t, &asana.CreateTaskRequest{
		Projects: []string{p.ID},
		Tags:     []string{tag.ID},
	})
	fetchedTask := &asana.Task{ID: task.ID}
	if err := fetchedTask.Fetch(f.client); err != nil {
		t.Fatal(err)
	}
	if !contains(fetchedTask.Tags, tag.ID, tagID) {
		t.Error("expected the task to carry the tag")
	}

	if err := tag.Delete(f.client); err != nil {
		t.Fatal(err)
	}
	if err := fetched.Fetch(f.client); !asana.IsNotFoundError(err) {
		t.Errorf("expected a not found error fetching a deleted tag, got %v", err)
	}
}

func TestAttachments(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")
	task := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})

	const content = "Uploaded by the asana-go integration tests\n"
	uploaded, err := task.CreateAttachment(f.client, &asana.NewAttachment{
		Reader:      io.NopCloser(strings.NewReader(content)),
		FileName:    "upload.txt",
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatal(err)
	}
	if uploaded.ID == "" || uploaded.Name != "upload.txt" {
		t.Errorf("expected an attachment named upload.txt, got %+v", uploaded)
	}

	external, err := task.CreateExternalAttachment(f.client, &asana.ExternalAttachmentRequest{
		Name: "Example link",
		URL:  "https://example.com/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if external.ResourceSubtype != "external" {
		t.Errorf("expected an external attachment, got %q", external.ResourceSubtype)
	}

	attachments, _, err := task.Attachments(f.client, asana.Fields(asana.Attachment{}))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(attachments, external.ID, attachmentID) {
		t.Error("expected the task's attachments to include the external link")
	}
	i := indexOf(attachments, uploaded.ID, attachmentID)
	if i < 0 {
		t.Fatal("expected the task's attachments to include the upload")
	}

	// The download URL is pre-signed, so it is fetched without the token
	url := attachments[i].DownloadURL
	if url == "" {
		t.Fatal("expected the upload to have a download URL")
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != content {
		t.Errorf("expected the downloaded attachment to match the upload, got %q", body)
	}
}
