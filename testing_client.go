package azugo

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/goccy/go-json"
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

// WithHeader adds header to request.
func (c *TestClient) WithHeader(key, value string) TestClientOption {
	return func(tc *TestClient, r *fasthttp.Request) {
		r.Header.Add(key, value)
	}
}

// WithQuery adds query parameters from map to query arguments.
func (c *TestClient) WithQuery(params map[string]interface{}) TestClientOption {
	return func(tc *TestClient, r *fasthttp.Request) {
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
	return c.CallRaw([]byte(method), []byte(fmt.Sprintf("http://test%s", endpoint)), body, options...)
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
func (c *TestClient) PatchJSON(endpoint string, body interface{}, options ...TestClientOption) (*fasthttp.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.Patch(endpoint, b, options...)
}

// Put calls PUT method with given body and options.
func (c *TestClient) Put(endpoint string, body []byte, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodPut, endpoint, body, options...)
}

// PutJSON calls PUT method with given object marshaled as JSON and options.
func (c *TestClient) PutJSON(endpoint string, body interface{}, options ...TestClientOption) (*fasthttp.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.Put(endpoint, b, options...)
}

// Post calls POST method with given body and options.
func (c *TestClient) Post(endpoint string, body []byte, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodPost, endpoint, body, options...)
}

// PostJSON calls POST method with given object marshaled as JSON and options.
func (c *TestClient) PostJSON(endpoint string, body interface{}, options ...TestClientOption) (*fasthttp.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.Post(endpoint, b, options...)
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
