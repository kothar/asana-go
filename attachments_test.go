package asana

import (
	"encoding/json"
	"io"
	"maps"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/h2non/gock"
)

// multipartBody is the decoded content of a multipart/form-data request
type multipartBody struct {
	fields   map[string]string
	files    map[string]string
	fileName string
	fileType string
}

// captureMultipart records the parts of a multipart/form-data request body.
func captureMultipart(sent *multipartBody) gock.MatchFunc {
	return func(req *http.Request, _ *gock.Request) (bool, error) {
		mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			return false, err
		}

		*sent = multipartBody{fields: map[string]string{}, files: map[string]string{}}
		reader := multipart.NewReader(req.Body, params["boundary"])
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				return true, nil
			}
			if err != nil {
				return false, err
			}
			content, err := io.ReadAll(part)
			if err != nil {
				return false, err
			}
			if part.FileName() != "" {
				sent.files[part.FormName()] = string(content)
				sent.fileName = part.FileName()
				sent.fileType = part.Header.Get("Content-Type")
			} else {
				sent.fields[part.FormName()] = string(content)
			}
		}
	}
}

func TestParentAttachments_List(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		list   func(*Client) ([]*Attachment, *NextPage, error)
	}{
		{
			name:   "project",
			parent: "10",
			list: func(c *Client) ([]*Attachment, *NextPage, error) {
				return (&Project{ID: "10"}).Attachments(c)
			},
		},
		{
			name:   "project brief",
			parent: "20",
			list: func(c *Client) ([]*Attachment, *NextPage, error) {
				return (&ProjectBrief{ID: "20"}).Attachments(c)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer gock.Off()

			gock.New("https://app.asana.com").
				Get("/api/1.0/attachments").
				MatchParam("parent", "^"+tc.parent+"$").
				Reply(200).
				JSON(o{"data": []o{{"gid": "30", "resource_type": "attachment", "name": "spec.pdf"}}})

			attachments, _, err := tc.list(NewClient(http.DefaultClient))
			if err != nil {
				t.Fatal(err)
			}
			if !gock.IsDone() {
				t.Fatal("Expected GET /attachments?parent=" + tc.parent)
			}
			if len(attachments) != 1 || attachments[0].ID != "30" || attachments[0].Name != "spec.pdf" {
				t.Errorf("Unexpected attachments: %+v", attachments)
			}
		})
	}
}

func TestParentAttachments_Upload(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		create func(*Client, *NewAttachment) (*Attachment, error)
	}{
		{
			name:   "project",
			parent: "10",
			create: func(c *Client, r *NewAttachment) (*Attachment, error) {
				return (&Project{ID: "10"}).CreateAttachment(c, r)
			},
		},
		{
			name:   "project brief",
			parent: "20",
			create: func(c *Client, r *NewAttachment) (*Attachment, error) {
				return (&ProjectBrief{ID: "20"}).CreateAttachment(c, r)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer gock.Off()

			var sent multipartBody
			gock.New("https://app.asana.com").
				Post("/api/1.0/attachments").
				AddMatcher(captureMultipart(&sent)).
				Reply(200).
				JSON(o{"data": o{"gid": "30", "name": "notes.txt", "resource_subtype": "asana"}})

			attachment, err := tc.create(NewClient(http.DefaultClient), &NewAttachment{
				Reader:      io.NopCloser(strings.NewReader("hello")),
				FileName:    "notes.txt",
				ContentType: "text/plain",
			})
			if err != nil {
				t.Fatal(err)
			}

			if want := map[string]string{"parent": tc.parent}; !maps.Equal(sent.fields, want) {
				t.Errorf("form fields = %v, want %v", sent.fields, want)
			}
			if want := map[string]string{"file": "hello"}; !maps.Equal(sent.files, want) {
				t.Errorf("file parts = %v, want %v", sent.files, want)
			}
			if sent.fileName != "notes.txt" || sent.fileType != "text/plain" {
				t.Errorf("file name/type = %q/%q, want notes.txt/text/plain", sent.fileName, sent.fileType)
			}
			if attachment.ID != "30" {
				t.Errorf("Unexpected attachment: %+v", attachment)
			}
		})
	}
}

func TestParentAttachments_External(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		create func(*Client, *ExternalAttachmentRequest) (*Attachment, error)
	}{
		{
			name:   "project",
			parent: "10",
			create: func(c *Client, r *ExternalAttachmentRequest) (*Attachment, error) {
				return (&Project{ID: "10"}).CreateExternalAttachment(c, r)
			},
		},
		{
			name:   "project brief",
			parent: "20",
			create: func(c *Client, r *ExternalAttachmentRequest) (*Attachment, error) {
				return (&ProjectBrief{ID: "20"}).CreateExternalAttachment(c, r)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer gock.Off()

			var sent multipartBody
			gock.New("https://app.asana.com").
				Post("/api/1.0/attachments").
				AddMatcher(captureMultipart(&sent)).
				Reply(200).
				JSON(o{"data": o{"gid": "31", "name": "Design doc", "resource_subtype": "external"}})

			// ConnectToApp only applies to task attachments, so it is not sent
			connect := true
			attachment, err := tc.create(NewClient(http.DefaultClient), &ExternalAttachmentRequest{
				Name:         "Design doc",
				URL:          "https://example.com/doc",
				ConnectToApp: &connect,
			})
			if err != nil {
				t.Fatal(err)
			}

			want := map[string]string{
				"parent":           tc.parent,
				"resource_subtype": "external",
				"name":             "Design doc",
				"url":              "https://example.com/doc",
			}
			if !maps.Equal(sent.fields, want) {
				t.Errorf("form fields = %v, want %v", sent.fields, want)
			}
			if len(sent.files) != 0 {
				t.Errorf("Expected no file part, got %v", sent.files)
			}
			if attachment.ID != "31" || attachment.ResourceSubtype != "external" {
				t.Errorf("Unexpected attachment: %+v", attachment)
			}
		})
	}
}

func TestParentAttachments_UploadWithoutFile(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Post("/api/1.0/attachments").
		Reply(200).
		JSON(o{"data": o{"gid": "30"}})

	client := NewClient(http.DefaultClient)
	for _, request := range []*NewAttachment{nil, {FileName: "notes.txt"}} {
		if _, err := (&Project{ID: "10"}).CreateAttachment(client, request); err == nil {
			t.Errorf("Expected an error for %+v", request)
		}
	}
	if !gock.IsPending() {
		t.Error("Expected no request without a file")
	}
}

// TestTask_CreateAttachmentUnchanged guards the existing task upload, which
// shares postMultipart with the parent-based endpoints.
func TestTask_CreateAttachmentUnchanged(t *testing.T) {
	defer gock.Off()

	var sent multipartBody
	gock.New("https://app.asana.com").
		Post("/api/1.0/tasks/40/attachments").
		AddMatcher(captureMultipart(&sent)).
		Reply(200).
		JSON(o{"data": o{"gid": "32", "name": "notes.txt"}})

	_, err := (&Task{ID: "40"}).CreateAttachment(NewClient(http.DefaultClient), &NewAttachment{
		Reader:      io.NopCloser(strings.NewReader("hello")),
		FileName:    "notes.txt",
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sent.fields) != 0 {
		t.Errorf("Expected no form fields, got %v", sent.fields)
	}
	if want := map[string]string{"file": "hello"}; !maps.Equal(sent.files, want) {
		t.Errorf("file parts = %v, want %v", sent.files, want)
	}
}

func TestAttachment_ProjectParent(t *testing.T) {
	attachment := &Attachment{}
	if err := json.Unmarshal([]byte(`{
		"gid": "30",
		"parent": {"gid": "10", "resource_type": "project", "name": "Roadmap"}
	}`), attachment); err != nil {
		t.Fatal(err)
	}
	if attachment.Parent == nil || attachment.Parent.ID != "10" || attachment.Parent.Name != "Roadmap" {
		t.Errorf("Parent = %+v, want gid 10 Roadmap", attachment.Parent)
	}
}
