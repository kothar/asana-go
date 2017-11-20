package asana

import (
	"net/http"
)

// NewClientWithAccessToken creates a new instance of the Asana client which uses a
// Personal Access Token for authentication
func NewClientWithAccessToken(accessToken string) *Client {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			req.Header.Add("Authorization", "Bearer "+accessToken)
			return nil
		},
	}
	return NewClient(client)
}
