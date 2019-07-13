// Package asana provides a client for the Asana API
package asana // import "bitbucket.org/mikehouston/asana-go"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strings"

	"github.com/rs/xid"

	"github.com/google/go-querystring/query"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
)

const (
	// BaseURL is the default URL used to access the Asana API
	BaseURL = "https://app.asana.com/api/1.0"
)

type Feature string

func (f Feature) String() string {
	return string(f)
}

const (
	NewTaskSubtypes Feature = "new_task_subtypes"
	NewSections     Feature = "new_sections"
	StringIDs       Feature = "string_ids"
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

type NextPage struct {
	Offset string `json:"offset"`
	Path   string `json:"path"`
	URI    string `json:"uri"`
}

// An API response
type Response struct {
	Data     json.RawMessage `json:"data"`
	NextPage *NextPage       `json:"next_page"`
	Errors   []*Error        `json:"errors"`
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
		return errors.Wrap(err, "Unable to marshal request to query parameters")
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

func (c *Client) get(path string, data, result interface{}, opts ...*Options) (*NextPage, error) {
	requestID := xid.New()

	// Prepare options
	options, err := c.mergeOptions(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "%s unable to merge options", requestID)
	}

	// Encode default options
	if c.Debug {
		log.Printf("%s Default options: %+v", requestID, c.DefaultOptions)
	}
	q, err := query.Values(c.DefaultOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "%s Unable to marshal DefaultOptions to query parameters", requestID)
	}

	// Encode data
	if data != nil {
		if c.Debug {
			log.Printf("%s Data: %+v", requestID, data)
		}

		// Validate
		if validator, ok := data.(Validator); ok {
			if err := validator.Validate(); err != nil {
				return nil, err
			}
		}

		if err := mergeQuery(q, data); err != nil {
			return nil, err
		}
	}

	// Encode query options
	for _, options := range opts {
		if c.Debug {
			log.Printf("%s Options: %+v", requestID, options)
		}
		if err := mergeQuery(q, options); err != nil {
			return nil, err
		}
	}
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}

	// Make request
	if c.Debug {
		log.Printf("%s GET %s", requestID, path)
	}
	request, err := http.NewRequest(http.MethodGet, c.getURL(path), nil)
	if err != nil {
		return nil, errors.Wrapf(err, "%s Request error", requestID)
	}
	c.addHeaders(request, options)
	resp, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, errors.Wrapf(err, "%s GET error", requestID)
	}

	// Parse the result
	resultData, err := c.parseResponse(resp, result, requestID)
	if err != nil {
		return nil, err
	}

	return resultData.NextPage, nil
}

func (c *Client) addHeaders(request *http.Request, options *Options) {
	if options.FastAPI {
		request.Header.Add("Asana-Fast-Api", "true")
	}

	if len(options.Enable) > 0 {
		request.Header.Add("Asana-Enable", joinFeatures(options.Enable))
	}
	if len(options.Disable) > 0 {
		request.Header.Add("Asana-Disable", joinFeatures(options.Disable))
	}

	if c.Debug {
		request.Header.Write(os.Stderr)
	}
}

func joinFeatures(features []Feature) string {
	b := strings.Builder{}
	for _, feature := range features {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(string(feature))
	}
	return b.String()
}

func (c *Client) post(path string, data, result interface{}, opts ...*Options) error {
	return c.do(http.MethodPost, path, data, result, opts...)
}

func (c *Client) put(path string, data, result interface{}, opts ...*Options) error {
	return c.do(http.MethodPut, path, data, result, opts...)
}

func (c *Client) delete(path string, opts ...*Options) error {
	return c.do(http.MethodDelete, path, nil, nil, opts...)
}

func (c *Client) do(method, path string, data, result interface{}, opts ...*Options) error {

	requestID := xid.New()

	// Prepare options
	options, err := c.mergeOptions(opts...)
	if err != nil {
		return errors.Wrapf(err, "%s unable to merge options", requestID)
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
		log.Printf("%s %s %s\n%s", requestID, method, path, body)
	}
	request, err := http.NewRequest(method, c.getURL(path), bytes.NewReader(body))
	if err != nil {
		return errors.Wrap(err, "Request error")
	}

	request.Header.Add("Content-Type", "application/json")
	c.addHeaders(request, options)
	resp, err := c.HTTPClient.Do(request)
	if err != nil {
		return errors.Wrapf(err, "%s error", method)
	}

	_, err = c.parseResponse(resp, result, requestID)
	return err
}

func (c *Client) mergeOptions(opts ...*Options) (*Options, error) {
	var options *Options
	if opts != nil {
		options = opts[0]
	}
	if options == nil {
		options = &Options{}
	}
	err := mergo.Merge(options, c.DefaultOptions)
	return options, err
}

// From mime.multipart package ------
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

// --------

func (c *Client) postMultipart(path string, result interface{}, field string, r io.ReadCloser, filename string, contentType string, opts ...*Options) error {
	// Make request
	requestID := xid.New()
	options, err := c.mergeOptions(opts...)
	if err != nil {
		return errors.Wrapf(err, "%s unable to merge options", requestID)
	}

	if c.Debug {
		log.Printf("%s POST multipart %s\n%s=%s;ContentType=%s", requestID, path, field, filename, contentType)
	}
	defer r.Close()

	// Write header
	buffer := &bytes.Buffer{}
	partWriter := multipart.NewWriter(buffer)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes(field), escapeQuotes(filename)))
	h.Set("Content-Type", contentType)

	_, err = partWriter.CreatePart(h)
	if err != nil {
		return errors.Wrapf(err, "%s create multipart header", requestID)
	}
	headerSize := buffer.Len()

	// Write footer
	if err = partWriter.Close(); err != nil {
		return errors.Wrapf(err, "%s create multipart footer", requestID)
	}

	// Create request
	request, err := http.NewRequest(http.MethodPost, c.getURL(path), io.MultiReader(
		bytes.NewReader(buffer.Bytes()[:headerSize]),
		r,
		bytes.NewReader(buffer.Bytes()[headerSize:])))
	if err != nil {
		return errors.Wrapf(err, "%s Request error", requestID)
	}

	request.Header.Add("Content-Type", partWriter.FormDataContentType())
	c.addHeaders(request, options)
	resp, err := c.HTTPClient.Do(request)
	if err != nil {
		return errors.Wrapf(err, "%s POST error", requestID)
	}

	_, err = c.parseResponse(resp, result, requestID)
	return err
}

func (c *Client) parseResponse(resp *http.Response, result interface{}, requestID xid.ID) (*Response, error) {

	// Get response body
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if c.Debug {
		resp.Header.Write(os.Stderr)
		fmt.Fprintf(os.Stderr, "%s %s\n%s\n", requestID, resp.Status, body)
	}

	// Decode the response
	value := &Response{}
	if err := json.Unmarshal(body, value); err != nil {
		value.Errors = []*Error{{
			StatusCode: resp.StatusCode,
			Type:       "unknown",
			Message:    http.StatusText(resp.StatusCode),
			RequestID:  requestID.String(),
		}}
	}

	// Check for errors
	switch resp.StatusCode {
	case 200: // OK
	case 201: // Object created
	default:
		return nil, value.Error(resp, requestID)
	}

	// Decode the data field
	if value.Data == nil {
		return nil, errors.Errorf("%s Missing data from response", requestID)
	}

	return value, c.parseResponseData(value.Data, result, requestID)
}

func (c *Client) parseResponseData(data []byte, result interface{}, requestID xid.ID) error {
	if err := json.Unmarshal(data, result); err != nil {
		return errors.Wrapf(err, "%s Unable to parse response data", requestID)
	}

	return nil
}
