package asana

import (
	"encoding/json"
	"maps"
	"net/http"
	"testing"

	"github.com/h2non/gock"
)

func TestProject_AddAndRemoveFollowers(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		call          func(*Project, *Client, []string) error
		followers     []string
		wantFollowers string
		replyUsers    []o
		wantUserIDs   []string
	}{
		{
			name:          "add",
			path:          "/api/1.0/projects/10/addFollowers",
			call:          func(p *Project, c *Client, f []string) error { return p.AddFollowers(c, f) },
			followers:     []string{"me", "user@example.com", "123"},
			wantFollowers: "me,user@example.com,123",
			replyUsers:    []o{{"gid": "1"}, {"gid": "2"}, {"gid": "123"}},
			wantUserIDs:   []string{"1", "2", "123"},
		},
		{
			name:          "remove",
			path:          "/api/1.0/projects/10/removeFollowers",
			call:          func(p *Project, c *Client, f []string) error { return p.RemoveFollowers(c, f) },
			followers:     []string{"123"},
			wantFollowers: "123",
			replyUsers:    []o{{"gid": "1"}},
			wantUserIDs:   []string{"1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer gock.Off()

			var sent map[string]any
			gock.New("https://app.asana.com").
				Post(tc.path).
				AddMatcher(func(req *http.Request, _ *gock.Request) (bool, error) {
					var body struct {
						Data map[string]any `json:"data"`
					}
					err := json.NewDecoder(req.Body).Decode(&body)
					sent = body.Data
					return err == nil, err
				}).
				Reply(200).
				JSON(o{"data": o{"gid": "10", "name": "Roadmap", "followers": tc.replyUsers}})

			client := NewClient(http.DefaultClient)
			project := &Project{ID: "10"}
			if err := tc.call(project, client, tc.followers); err != nil {
				t.Fatal(err)
			}
			if !gock.IsDone() {
				t.Fatalf("Expected POST %s to be made", tc.path)
			}

			want := map[string]any{"followers": tc.wantFollowers}
			if !maps.Equal(sent, want) {
				t.Errorf("POST data = %v, want %v", sent, want)
			}

			if project.Name != "Roadmap" || len(project.Followers) != len(tc.wantUserIDs) {
				t.Fatalf("Project not updated from response: %+v", project)
			}
			for i, id := range tc.wantUserIDs {
				if project.Followers[i].ID != id {
					t.Errorf("Followers[%d] = %q, want %q", i, project.Followers[i].ID, id)
				}
			}
		})
	}
}

func TestProject_FollowersEmptyListMakesNoRequest(t *testing.T) {
	defer gock.Off()

	gock.New("https://app.asana.com").
		Post("/api/1.0/projects/10/(add|remove)Followers").
		Times(2).
		Reply(200).
		JSON(o{"data": o{"gid": "10"}})

	client := NewClient(http.DefaultClient)
	project := &Project{ID: "10"}
	if err := project.AddFollowers(client, nil); err != nil {
		t.Fatal(err)
	}
	if err := project.RemoveFollowers(client, []string{}); err != nil {
		t.Fatal(err)
	}
	if !gock.IsPending() {
		t.Error("Expected no request for an empty follower list")
	}
}
