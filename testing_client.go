package azugo

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"azugo.io/azugo/internal/utils"

	"github.com/goccy/go-json"
	"github.com/oklog/ulid/v2"
	"github.com/valyala/fasthttp"
)

// TestClient is a test client for testing purposes.
type TestClient struct {
	app    *TestApp
	client *fasthttp.Client
}

// TestClientOption is a test client option.
type TestClientOption func(*TestClient, *fasthttp.Request)

func (c *TestClient) applyOptions(request *fasthttp.Request, options []TestClientOption) {
	for _, option := range options {
		option(c, request)
	}
}

// WithHost sets host for the request.
func (c *TestClient) WithHost(host string) TestClientOption {
	return func(_ *TestClient, r *fasthttp.Request) {
		r.SetHost(host)
	}
}

// WithHeader adds header to request.
func (c *TestClient) WithHeader(key, value string) TestClientOption {
	return func(_ *TestClient, r *fasthttp.Request) {
		r.Header.Add(key, value)
	}
}

// WithMultiPartFormBoundary sets multipart form data boundary.
func (c *TestClient) WithMultiPartFormBoundary(boundary string) TestClientOption {
	return func(_ *TestClient, r *fasthttp.Request) {
		r.Header.SetMultipartFormBoundary(boundary)
	}
}

// WithQuery adds query parameters from map to query arguments.
func (c *TestClient) WithQuery(params map[string]any) TestClientOption {
	return func(_ *TestClient, r *fasthttp.Request) {
		for key, value := range params {
			var val string
			switch v := value.(type) {
			case []string:
				val = strings.Join(v, ",")
			default:
				val = fmt.Sprintf("%v", v)
			}

			r.URI().QueryArgs().Add(key, val)
		}
	}
}

// Client returns the underlying fasthttp client.
func (c *TestClient) Client() *fasthttp.Client {
	return c.client
}

// CallRaw calls the given method and endpoint with the given body and options.
func (c *TestClient) CallRaw(method, endpoint, body []byte, options ...TestClientOption) (*fasthttp.Response, error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethodBytes(method)
	req.SetRequestURIBytes(endpoint)

	c.applyOptions(req, options)

	if len(body) > fasthttp.DefaultMaxRequestBodySize {
		req.SetBodyStream(bytes.NewReader(body), len(body))
	} else if len(body) > 0 {
		req.SetBodyRaw(body)
	}
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()

	if err := c.client.Do(req, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// Call calls the given method and endpoint with the given body and options.
func (c *TestClient) Call(method, endpoint string, body []byte, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.CallRaw([]byte(method), []byte("http://test"+endpoint), body, options...)
}

// Get calls GET method with given options.
func (c *TestClient) Get(endpoint string, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodGet, endpoint, nil, options...)
}

// Head calls HEAD method with given options.
func (c *TestClient) Head(endpoint string, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodHead, endpoint, nil, options...)
}

// Delete calls DELETE method with given options.
func (c *TestClient) Delete(endpoint string, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodDelete, endpoint, nil, options...)
}

// Patch calls PATCH method with given body and options.
func (c *TestClient) Patch(endpoint string, body []byte, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodPatch, endpoint, body, options...)
}

// PatchJSON calls PATCH method with given object marshaled as JSON and options.
func (c *TestClient) PatchJSON(endpoint string, body any, options ...TestClientOption) (*fasthttp.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	options = append(options, c.WithHeader("Content-Type", "application/json"))

	return c.Patch(endpoint, b, options...)
}

// PatchForm calls PATCH method with given map marshaled as URL encoded form and options.
func (c *TestClient) PatchForm(endpoint string, params map[string]any, options ...TestClientOption) (*fasthttp.Response, error) {
	options = append(options, c.WithHeader("Content-Type", "application/x-www-form-urlencoded"))

	return c.Patch(endpoint, []byte(utils.MapToURLValues(params)), options...)
}

// PatchMultiPartForm calls PATCH method with given multipart form and options.
func (c *TestClient) PatchMultiPartForm(endpoint string, form *multipart.Form, options ...TestClientOption) (*fasthttp.Response, error) {
	boundary, err := ulid.New(ulid.Timestamp(time.Now().UTC()), ulid.Monotonic(rand.Reader, 1))
	if err != nil {
		return nil, err
	}

	options = append(options, c.WithMultiPartFormBoundary(boundary.String()))

	var body bytes.Buffer
	if err = fasthttp.WriteMultipartForm(&body, form, boundary.String()); err != nil {
		return nil, err
	}

	return c.Put(endpoint, body.Bytes(), options...)
}

// Put calls PUT method with given body and options.
func (c *TestClient) Put(endpoint string, body []byte, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodPut, endpoint, body, options...)
}

// PutJSON calls PUT method with given object marshaled as JSON and options.
func (c *TestClient) PutJSON(endpoint string, body any, options ...TestClientOption) (*fasthttp.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	options = append(options, c.WithHeader("Content-Type", "application/json"))

	return c.Put(endpoint, b, options...)
}

// PutForm calls PUT method with given map marshaled as URL encoded form and options.
func (c *TestClient) PutForm(endpoint string, params map[string]any, options ...TestClientOption) (*fasthttp.Response, error) {
	options = append(options, c.WithHeader("Content-Type", "application/x-www-form-urlencoded"))

	return c.Put(endpoint, []byte(utils.MapToURLValues(params)), options...)
}

// PutMultiPartForm calls PUT method with given multipart form and options.
func (c *TestClient) PutMultiPartForm(endpoint string, form *multipart.Form, options ...TestClientOption) (*fasthttp.Response, error) {
	boundary, err := ulid.New(ulid.Timestamp(time.Now().UTC()), ulid.Monotonic(rand.Reader, 1))
	if err != nil {
		return nil, err
	}

	options = append(options, c.WithMultiPartFormBoundary(boundary.String()))

	var body bytes.Buffer
	if err = fasthttp.WriteMultipartForm(&body, form, boundary.String()); err != nil {
		return nil, err
	}

	return c.Put(endpoint, body.Bytes(), options...)
}

// Post calls POST method with given body and options.
func (c *TestClient) Post(endpoint string, body []byte, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodPost, endpoint, body, options...)
}

// PostJSON calls POST method with given object marshaled as JSON and options.
func (c *TestClient) PostJSON(endpoint string, body any, options ...TestClientOption) (*fasthttp.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	options = append(options, c.WithHeader("Content-Type", "application/json"))

	return c.Post(endpoint, b, options...)
}

// PostForm calls POST method with given map marshaled as URL encoded form and options.
func (c *TestClient) PostForm(endpoint string, params map[string]any, options ...TestClientOption) (*fasthttp.Response, error) {
	options = append(options, c.WithHeader("Content-Type", "application/x-www-form-urlencoded"))

	return c.Post(endpoint, []byte(utils.MapToURLValues(params)), options...)
}

// PostMultiPartForm calls POST method with given multipart form and options.
func (c *TestClient) PostMultiPartForm(endpoint string, form *multipart.Form, options ...TestClientOption) (*fasthttp.Response, error) {
	boundary, err := ulid.New(ulid.Timestamp(time.Now().UTC()), ulid.Monotonic(rand.Reader, 1))
	if err != nil {
		return nil, err
	}

	options = append(options, c.WithMultiPartFormBoundary(boundary.String()))

	var body bytes.Buffer
	if err = fasthttp.WriteMultipartForm(&body, form, boundary.String()); err != nil {
		return nil, err
	}

	return c.Post(endpoint, body.Bytes(), options...)
}

// Connect calls CONNECT method with given options.
func (c *TestClient) Connect(endpoint string, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodConnect, endpoint, nil, options...)
}

// Options calls OPTIONS method with given options.
func (c *TestClient) Options(endpoint string, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodOptions, endpoint, nil, options...)
}

// Trace calls TRACE method with given options.
func (c *TestClient) Trace(endpoint string, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodTrace, endpoint, nil, options...)
}
