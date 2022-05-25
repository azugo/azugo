package azugo

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

type testBodyUser struct {
	Name string `json:"name" validate:"required,max=10"`
}

func (t *testBodyUser) Validate(ctx *Context) error {
	return ctx.Validate().StructCtx(ctx.Context(), t)
}

func TestBodyBytes(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	expect := []byte("test")

	a.Post("/user", func(ctx *Context) {
		assert.Equal(t, expect, ctx.Body.Bytes(), "Body should be equal to test")
		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Post("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestBodyStream(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	expect := make([]byte, fasthttp.DefaultMaxRequestBodySize+1024)

	a.Post("/user", func(ctx *Context) {
		var buf bytes.Buffer
		err := ctx.Body.WriteTo(&buf)
		assert.NoError(t, err, "Body should be copied")
		assert.Equal(t, expect, buf.Bytes(), "Body should be equal to test")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Post("/user", expect)
	require.NotNil(t, resp)
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestBodyPostJSON(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	expect := testBodyUser{Name: "test"}

	a.Post("/user", func(ctx *Context) {
		var user testBodyUser
		err := ctx.Body.JSON(&user)
		assert.NoError(t, err, "JSON should unmarshal without error")
		assert.Equal(t, expect, user, "Body should be equal to test")
		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().PostJSON("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestBodyPutJSON(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	expect := testBodyUser{Name: "test"}

	a.Put("/user", func(ctx *Context) {
		var user testBodyUser
		err := ctx.Body.JSON(&user)
		assert.NoError(t, err, "JSON should unmarshal without error")
		assert.Equal(t, expect, user, "Body should be equal to test")
		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().PutJSON("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestBodyPatchJSON(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	expect := testBodyUser{Name: "test"}

	a.Patch("/user", func(ctx *Context) {
		var user testBodyUser
		err := ctx.Body.JSON(&user)
		assert.NoError(t, err, "JSON should unmarshal without error")
		assert.Equal(t, expect, user, "Body should be equal to test")
		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().PatchJSON("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestBodyJSONValidationError(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	expect := testBodyUser{Name: "test1234567890"}

	a.Patch("/user", func(ctx *Context) {
		ctx.Header.SetContentType("application/json")

		var user testBodyUser
		if err := ctx.Body.JSON(&user); err != nil {
			ctx.Error(err)
			return
		}
		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().PatchJSON("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusUnprocessableEntity, resp.StatusCode(), "wrong status code")
	assert.Equal(t, `{"errors":[{"type":"FieldError","message":"Key: 'testBodyUser.Name' Error:Field validation for 'Name' failed on the 'max' tag"}]}`, string(resp.Body()))
}

func TestBodyXML(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	type User struct {
		Name string `xml:"name"`
	}

	expect := User{Name: "test"}

	a.Put("/user", func(ctx *Context) {
		var user User
		err := ctx.Body.XML(&user)
		assert.NoError(t, err, "XML should unmarshal without error")
		assert.Equal(t, expect, user, "Body should be equal to test")
		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Put("/user", []byte("<user><name>test</name></user>"))
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}
