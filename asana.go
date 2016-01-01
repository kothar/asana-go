// Package asana provides a client for the Asana API
package asana //import "bitbucket.org/mikehouston/asana-go"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/google/go-querystring/query"
	"github.com/imdario/mergo"
)

const (
	// BaseURL is the default URL used to access the Asana API
	BaseURL = "https://app.asana.com/api/1.0"
)

// Client is the root client for the Asana API. The nested HTTPClient should provide
// Authorization header injection.
type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client

	Debug          bool
	DefaultOptions Options
}

// NewClient instantiates a new Asana client with the given HTTP client and
// the default base URL
func NewClient(httpClient *http.Client) *Client {
	u, _ := url.Parse(BaseURL)
	return &Client{
		BaseURL:    u,
		HTTPClient: httpClient,
	}
}

// A POST API request
type request struct {
	Data    interface{} `json:"data"`
	Options *Options    `json:"options,omitempty"`
}

// An API response
type response struct {
	Data   json.RawMessage `json:"data"`
	Errors []*Error        `json:"errors"`
}

// Error is an error message returned by the API
type Error struct {
	Message string `json:"message"`
	Phrase  string `json:"phrase"`
}

func (err Error) Error() string {
	return err.Message
}

func (c *Client) getURL(path string) string {
	if path[0] != '/' {
		panic("Invalid API path")
	}
	return c.BaseURL.String() + path
}

func (c *Client) get(path string, request, result interface{}) error {

	// Encode default options
	q, err := query.Values(c.DefaultOptions)
	if err != nil {
		return err
	}

	// Encode query params
	if request != nil {
		queryParams, err := query.Values(request)
		if err != nil {
			return err
		}

		// Merge with defaults
		for key, values := range queryParams {
			q.Del(key)
			for _, value := range values {
				q.Add(key, value)
			}
		}
	}
	path = path + "?" + q.Encode()

	// Make request
	if c.Debug {
		log.Printf("GET %s", path)
	}
	resp, err := c.HTTPClient.Get(c.getURL(path))
	if err != nil {
		return err
	}

	return c.parseResponse(resp, result)
}

func (c *Client) post(path string, data, result interface{}, options *Options) error {

	// Build request
	if err := mergo.Merge(options, c.DefaultOptions); err != nil {
		return err
	}
	req := &request{
		Data:    data,
		Options: options,
	}

	// Encode request body
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// Make request
	if c.Debug {
		log.Printf("POST %s", path)
		log.Printf("Request: %s", body)
	}
	resp, err := c.HTTPClient.Post(c.getURL(path), "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}

	return c.parseResponse(resp, result)
}

func (c *Client) parseResponse(resp *http.Response, result interface{}) error {

	// Get response body
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if c.Debug {
		log.Printf("Response: %s", body)
	}

	// Decode the response
	value := &response{}
	if err := json.Unmarshal(body, value); err != nil {
		return err
	}

	// Check for errors
	if resp.StatusCode != 200 {
		if value.Errors != nil {
			return value.Errors[0]
		}

		return fmt.Errorf("Unexpected error: Status %d", resp.StatusCode)
	}

	// Decode the data field
	if value.Data == nil {
		return fmt.Errorf("Missing data from response")
	}
	if err := json.Unmarshal(value.Data, result); err != nil {
		return err
	}

	// Inject the client into expandable types
	injectClient(c, result)

	return nil

}
