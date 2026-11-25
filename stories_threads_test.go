package asana

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/h2non/gock"
)

func TestStory_StickerAndThreadFields(t *testing.T) {
	fields := Fields(Story{}).Fields
	for _, name := range []string{"sticker_name", "comment_gid", "anchor_gid"} {
		if !slices.Contains(fields, name) {
			t.Errorf("Expected Fields(Story{}) to include %q, got %v", name, fields)
		}
	}

	story := &Story{}
	if err := json.Unmarshal([]byte(`{
		"gid": "50",
		"resource_subtype": "comment_added",
		"text": "Nice work",
		"is_pinned": true,
		"sticker_name": "dancing_unicorn",
		"comment_gid": "60",
		"anchor_gid": "70"
	}`), story); err != nil {
		t.Fatal(err)
	}

	if story.StickerName != "dancing_unicorn" {
		t.Errorf("StickerName = %q, want %q", story.StickerName, "dancing_unicorn")
	}
	if story.CommentGID != "60" || story.AnchorGID != "70" {
		t.Errorf("CommentGID/AnchorGID = %q/%q, want 60/70", story.CommentGID, story.AnchorGID)
	}
	if !story.IsPinned {
		t.Error("IsPinned = false, want true")
	}
}

func TestTask_CreateStory(t *testing.T) {
	tests := []struct {
		name    string
		request *CreateStoryRequest
		want    map[string]any
	}{
		{
			name: "threaded reply",
			request: &CreateStoryRequest{
				StoryBase:     StoryBase{HTMLText: "<body>Agreed</body>"},
				ParentComment: "60",
			},
			want: map[string]any{"html_text": "<body>Agreed</body>", "parent_comment": "60"},
		},
		{
			name: "pinned comment with sticker",
			request: &CreateStoryRequest{
				StoryBase: StoryBase{Text: "Shipped", IsPinned: true, StickerName: "trophy"},
			},
			want: map[string]any{"text": "Shipped", "is_pinned": true, "sticker_name": "trophy"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer gock.Off()

			var sent map[string]any
			gock.New("https://app.asana.com").
				Post("/api/1.0/tasks/40/stories").
				AddMatcher(func(req *http.Request, _ *gock.Request) (bool, error) {
					var body struct {
						Data map[string]any `json:"data"`
					}
					err := json.NewDecoder(req.Body).Decode(&body)
					sent = body.Data
					return err == nil, err
				}).
				Reply(201).
				JSON(o{"data": o{"gid": "51", "resource_subtype": "comment_added", "anchor_gid": "60"}})

			story, err := (&Task{ID: "40"}).CreateStory(NewClient(http.DefaultClient), tc.request)
			if err != nil {
				t.Fatal(err)
			}
			if !maps.Equal(sent, tc.want) {
				t.Errorf("POST /tasks/40/stories data = %v, want %v", sent, tc.want)
			}
			if story.ID != "51" || story.AnchorGID != "60" {
				t.Errorf("Unexpected story: %+v", story)
			}
		})
	}
}
