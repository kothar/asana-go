package integration

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kothar/asana-go"
)

func TestComments(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")
	task := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})

	comment := mustReturn(task.CreateComment(f.client, &asana.StoryBase{Text: "First comment"}))(t)
	if comment.ResourceSubtype != "comment_added" || comment.Text != "First comment" {
		t.Errorf("expected a comment_added story with the text, got %q %q", comment.ResourceSubtype, comment.Text)
	}
	if comment.CreatedBy == nil || comment.CreatedBy.ID != f.me.ID {
		t.Errorf("expected the comment to be created by %s, got %+v", f.me.ID, comment.CreatedBy)
	}

	stories := mustPage(task.Stories(f.client))(t)
	if !contains(stories, comment.ID) {
		t.Error("expected the task's stories to include the comment")
	}

	updated := mustReturn(comment.UpdateStory(f.client, &asana.StoryBase{Text: "Edited comment"}))(t)
	if updated.Text != "Edited comment" {
		t.Errorf("expected the edited text, got %q", updated.Text)
	}

	must(t, comment.Delete(f.client))
	stories = mustPage(task.Stories(f.client))(t)
	if contains(stories, comment.ID) {
		t.Error("expected the deleted comment to be gone")
	}
}

func TestTags(t *testing.T) {
	f := setup(t)

	tag := mustReturn(f.workspace.CreateTag(f.client, &asana.TagBase{
		Name:  f.name(t, "tag"),
		Notes: "Created by the asana-go integration tests",
	}))(t)
	cleanup(t, "tag "+tag.ID, func() error { return tag.Delete(f.client) })

	fetched := &asana.Tag{ID: tag.ID}
	must(t, fetched.Fetch(f.client))
	if fetched.Name != tag.Name {
		t.Errorf("expected tag name %q, got %q", tag.Name, fetched.Name)
	}
	if fetched.Workspace == nil || fetched.Workspace.ID != f.workspace.ID {
		t.Errorf("expected the tag to be in workspace %s, got %+v", f.workspace.ID, fetched.Workspace)
	}

	tags := mustReturn(f.workspace.AllTags(f.client))(t)
	if !contains(tags, tag.ID) {
		t.Error("expected AllTags to include the tag")
	}

	p := f.newProject(t, "project")
	task := f.newTask(t, &asana.CreateTaskRequest{
		Projects: []string{p.ID},
		Tags:     []string{tag.ID},
	})
	fetchedTask := &asana.Task{ID: task.ID}
	must(t, fetchedTask.Fetch(f.client))
	if !contains(fetchedTask.Tags, tag.ID) {
		t.Error("expected the task to carry the tag")
	}

	must(t, tag.Delete(f.client))
	if err := fetched.Fetch(f.client); !asana.IsNotFoundError(err) {
		t.Errorf("expected a not found error fetching a deleted tag, got %v", err)
	}
}

func TestAttachments(t *testing.T) {
	f := setup(t)
	p := f.newProject(t, "project")
	task := f.newTask(t, &asana.CreateTaskRequest{Projects: []string{p.ID}})

	const content = "Uploaded by the asana-go integration tests\n"
	uploaded := mustReturn(task.CreateAttachment(f.client, &asana.NewAttachment{
		Reader:      io.NopCloser(strings.NewReader(content)),
		FileName:    "upload.txt",
		ContentType: "text/plain",
	}))(t)
	if uploaded.ID == "" || uploaded.Name != "upload.txt" {
		t.Errorf("expected an attachment named upload.txt, got %+v", uploaded)
	}

	external := mustReturn(task.CreateExternalAttachment(f.client, &asana.ExternalAttachmentRequest{
		Name: "Example link",
		URL:  "https://example.com/",
	}))(t)
	if external.ResourceSubtype != "external" {
		t.Errorf("expected an external attachment, got %q", external.ResourceSubtype)
	}

	attachments := mustPage(task.Attachments(f.client, asana.Fields(asana.Attachment{})))(t)
	if !contains(attachments, external.ID) {
		t.Error("expected the task's attachments to include the external link")
	}
	i := indexOf(attachments, uploaded.ID)
	if i < 0 {
		t.Fatal("expected the task's attachments to include the upload")
	}

	// The download URL is pre-signed, so it is fetched without the token
	url := attachments[i].DownloadURL
	if url == "" {
		t.Fatal("expected the upload to have a download URL")
	}
	req := mustReturn(http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil))(t)
	resp := mustReturn((&http.Client{Timeout: 30 * time.Second}).Do(req))(t)
	defer resp.Body.Close()
	body := mustReturn(io.ReadAll(resp.Body))(t)
	if string(body) != content {
		t.Errorf("expected the downloaded attachment to match the upload, got %q", body)
	}
}
