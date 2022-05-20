package azugo

import (
	"fmt"

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

func (c *TestClient) WithHeader(key, value string) TestClientOption {
	return func(tc *TestClient, r *fasthttp.Request) {
		r.Header.Add(key, value)
	}
}

func (c *TestClient) CallRaw(method, endpoint, body []byte, options ...TestClientOption) (*fasthttp.Response, error) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethodBytes(method)
	req.SetRequestURIBytes(endpoint)

	c.applyOptions(req, options)

	if len(body) > 0 {
		req.SetBodyRaw(body)
	}
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()

	if err := c.client.Do(req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *TestClient) Call(method, endpoint string, body []byte, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.CallRaw([]byte(method), []byte(fmt.Sprintf("http://test%s", endpoint)), body, options...)
}

func (c *TestClient) Get(endpoint string, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodGet, endpoint, nil, options...)
}

func (c *TestClient) Head(endpoint string, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodHead, endpoint, nil, options...)
}

func (c *TestClient) Delete(endpoint string, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodDelete, endpoint, nil, options...)
}

func (c *TestClient) Patch(endpoint string, body []byte, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodPatch, endpoint, body, options...)
}

func (c *TestClient) PatchJSON(endpoint string, body interface{}, options ...TestClientOption) (*fasthttp.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.Patch(endpoint, b, options...)
}

func (c *TestClient) Put(endpoint string, body []byte, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodPut, endpoint, body, options...)
}

func (c *TestClient) PutJSON(endpoint string, body interface{}, options ...TestClientOption) (*fasthttp.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.Put(endpoint, b, options...)
}

func (c *TestClient) Post(endpoint string, body []byte, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodPost, endpoint, body, options...)
}

func (c *TestClient) PostJSON(endpoint string, body interface{}, options ...TestClientOption) (*fasthttp.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.Post(endpoint, b, options...)
}

func (c *TestClient) Connect(endpoint string, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodConnect, endpoint, nil, options...)
}

func (c *TestClient) Options(endpoint string, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodOptions, endpoint, nil, options...)
}

func (c *TestClient) Trace(endpoint string, options ...TestClientOption) (*fasthttp.Response, error) {
	return c.Call(fasthttp.MethodTrace, endpoint, nil, options...)
}
