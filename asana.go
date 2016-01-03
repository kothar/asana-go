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
	Verbose        []bool
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
	Help    string `json:"help"`
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

func mergeQuery(q url.Values, request interface{}) error {
	queryParams, err := query.Values(request)
	if err != nil {
		return fmt.Errorf("Unable to marshal request to query parameters: %s", err)
	}

	// Merge with defaults
	for key, values := range queryParams {
		q.Del(key)
		for _, value := range values {
			q.Add(key, value)
		}
	}

	return nil
}

func (c *Client) get(path string, data, result interface{}, opts ...*Options) error {

	// Encode default options
	if c.Debug {
		log.Printf("Default options: %+v", c.DefaultOptions)
	}
	q, err := query.Values(c.DefaultOptions)
	if err != nil {
		return fmt.Errorf("Unable to marshal DefaultOptions to query parameters: %s", err)
	}

	// Encode data
	if data != nil {
		if c.Debug {
			log.Printf("Data: %+v", data)
		}

		// Validate
		if validator, ok := data.(Validator); ok {
			if err := validator.Validate(); err != nil {
				return err
			}
		}

		if err := mergeQuery(q, data); err != nil {
			return err
		}
	}

	// Encode query options
	for _, options := range opts {
		if c.Debug {
			log.Printf("Options: %+v", options)
		}
		if err := mergeQuery(q, options); err != nil {
			return err
		}
	}
	path = path + "?" + q.Encode()

	// Make request
	if c.Debug {
		log.Printf("GET %s", path)
	}
	resp, err := c.HTTPClient.Get(c.getURL(path))
	if err != nil {
		return fmt.Errorf("GET error: %s", err)
	}

	return c.parseResponse(resp, result)
}

func (c *Client) post(path string, data, result interface{}, opts ...*Options) error {

	// Prepare options
	var options *Options
	if opts != nil {
		options = opts[0]
	}
	if options == nil {
		options = &Options{}
	}
	if err := mergo.Merge(options, c.DefaultOptions); err != nil {
		return fmt.Errorf("unable to merge options: %s", err)
	}

	// Validate data
	if validator, ok := data.(Validator); ok {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	// Build request
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
		body, _ := json.MarshalIndent(req, "", "  ")
		log.Printf("POST %s\n%s", path, body)
	}
	resp, err := c.HTTPClient.Post(c.getURL(path), "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("POST error: %s", err)
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
		log.Printf("%s\n%s", resp.Status, body)
	}

	// Decode the response
	value := &response{}
	if err := json.Unmarshal(body, value); err != nil {
		return err
	}

	// Check for errors
	switch resp.StatusCode {
	case 200: // OK
	case 201: // Object created
		// TODO cache the response based on the Location header
	case 401:
		if value.Errors != nil {
			return fmt.Errorf("Authorization error: %s", value.Errors[0])
		}

		return fmt.Errorf("Authorization error: Status %d", resp.StatusCode)
	default:
		if value.Errors != nil {
			return fmt.Errorf("API error: %s", value.Errors[0])
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
	c.inject(result)

	return nil

}
