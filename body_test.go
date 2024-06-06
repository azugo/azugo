package azugo

import (
	"bytes"
	"testing"

	"github.com/go-quicktest/qt"
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
		qt.Check(t, qt.ContentEquals(ctx.Body.Bytes(), expect), qt.Commentf("Body should be equal to test"))
		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Post("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK), qt.Commentf("wrong status code"))
}

func TestBodyStream(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	expect := make([]byte, fasthttp.DefaultMaxRequestBodySize+1024)
	var received []byte

	a.Post("/user", func(ctx *Context) {
		var buf bytes.Buffer
		err := ctx.Body.WriteTo(&buf)
		if err != nil {
			ctx.Error(err)
			return
		}

		received = buf.Bytes()

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Post("/user", expect)
	qt.Assert(t, qt.IsNotNil(resp))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK), qt.Commentf("wrong status code"))
	if !bytes.Equal(received, expect) {
		t.Fatal("Received request body should be equal to sent body")
	}
}

func TestBodyPostJSON(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	var user testBodyUser

	expect := testBodyUser{Name: "test"}

	a.Post("/user", func(ctx *Context) {
		if err := ctx.Body.JSON(&user); err != nil {
			ctx.Error(err)
			return
		}

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().PostJSON("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Assert(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	qt.Assert(t, qt.DeepEquals(expect, user))
}

func TestBodyPutJSON(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	var user testBodyUser

	expect := testBodyUser{Name: "test"}

	a.Put("/user", func(ctx *Context) {
		if err := ctx.Body.JSON(&user); err != nil {
			ctx.Error(err)
			return
		}

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().PutJSON("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Assert(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	qt.Assert(t, qt.DeepEquals(expect, user))
}

func TestBodyPatchJSON(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	var user testBodyUser

	expect := testBodyUser{Name: "test"}

	a.Patch("/user", func(ctx *Context) {
		if err := ctx.Body.JSON(&user); err != nil {
			ctx.Error(err)
			return
		}

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().PatchJSON("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Assert(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	qt.Assert(t, qt.DeepEquals(expect, user))
}

func TestBodyJSONValidationError(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	expect := testBodyUser{Name: "test1234567890"}

	a.Patch("/user", func(ctx *Context) {
		ctx.ContentType(ContentTypeJSON)

		var user testBodyUser
		if err := ctx.Body.JSON(&user); err != nil {
			ctx.Error(err)
			return
		}

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().PatchJSON("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Assert(t, qt.Equals(resp.StatusCode(), fasthttp.StatusUnprocessableEntity))
	qt.Assert(t, qt.Equals(string(resp.Body()), `{"errors":[{"type":"FieldError","message":"Key: 'testBodyUser.Name' Error:Field validation for 'Name' failed on the 'max' tag"}]}`))
}

func TestBodyXML(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	type User struct {
		Name string `xml:"name"`
	}

	var user User

	expect := User{Name: "test"}

	a.Put("/user", func(ctx *Context) {
		if err := ctx.Body.XML(&user); err != nil {
			ctx.Error(err)
			return
		}

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Put("/user", []byte("<user><name>test</name></user>"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Assert(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	qt.Assert(t, qt.DeepEquals(expect, user))
}
