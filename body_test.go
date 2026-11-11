package azugo

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"testing"

	"azugo.io/core/http"
	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

type failingBodyReader struct {
	err error
}

func (r failingBodyReader) Read(p []byte) (int, error) {
	copy(p, "x")

	return 1, r.err
}

type failingBodyWriter struct {
	err error
}

func (w failingBodyWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestStreamedBodyReadAndWriteErrors(t *testing.T) {
	sourceErr := io.ErrUnexpectedEOF
	for _, read := range []struct {
		name string
		fn   func(*BodyCtx) error
	}{
		{"reader", func(body *BodyCtx) error {
			r, err := body.Reader()
			if err != nil {
				return err
			}
			_, err = io.ReadAll(r)
			return err
		}},
		{"write-to", func(body *BodyCtx) error {
			_, err := body.WriteTo(io.Discard)
			return err
		}},
		{"read-all", func(body *BodyCtx) error {
			_, err := body.ReadAll()
			return err
		}},
	} {
		t.Run(read.name, func(t *testing.T) {
			var raw fasthttp.RequestCtx
			raw.Request.SetBodyStream(failingBodyReader{sourceErr}, -1)
			defer raw.Request.Reset()
			ctx := &Context{context: &raw, bodyLimit: 10}
			ctx.Body.ctx = ctx

			err := read.fn(&ctx.Body)
			var badRequest BadRequestError
			qt.Check(t, qt.ErrorAs(err, &badRequest))
			qt.Check(t, qt.ErrorIs(err, sourceErr))
		})
	}

	var raw fasthttp.RequestCtx
	raw.Request.SetBodyStream(strings.NewReader("body"), -1)
	defer raw.Request.Reset()
	ctx := &Context{context: &raw, bodyLimit: 10}
	ctx.Body.ctx = ctx
	destinationErr := errors.New("destination failed")
	_, err := ctx.Body.WriteTo(failingBodyWriter{destinationErr})
	qt.Check(t, qt.Equals(err, destinationErr))
}

func TestStreamedBodyReadersShareLimit(t *testing.T) {
	var raw fasthttp.RequestCtx
	raw.Request.SetBodyStream(strings.NewReader("abcdefgh"), -1)
	defer raw.Request.Reset()

	ctx := &Context{context: &raw, bodyLimit: 4}
	ctx.Body.ctx = ctx

	first, err := ctx.Body.Reader()
	qt.Assert(t, qt.IsNil(err))
	buf := make([]byte, 3)
	_, err = io.ReadFull(first, buf)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(buf), "abc"))

	second, err := ctx.Body.Reader()
	qt.Assert(t, qt.IsNil(err))
	remaining, err := io.ReadAll(second)
	qt.Check(t, qt.Equals(string(remaining), "d"))
	qt.Check(t, qt.IsTrue(errors.Is(err, RequestEntityTooLargeError{})))

	ctx.SetMaxBodySize(100)
	_, err = ctx.Body.Reader()
	qt.Check(t, qt.IsTrue(errors.Is(err, RequestEntityTooLargeError{})))
	_, err = ctx.Body.ReadAll()
	qt.Check(t, qt.IsTrue(errors.Is(err, RequestEntityTooLargeError{})))
	_, err = ctx.Body.WriteTo(io.Discard)
	qt.Check(t, qt.IsTrue(errors.Is(err, RequestEntityTooLargeError{})))

	// Bypassing the reader must not clear the error for later framework reads.
	_ = raw.Request.Body()
	_, err = ctx.Body.ReadAll()
	qt.Check(t, qt.IsTrue(errors.Is(err, RequestEntityTooLargeError{})))
}

func TestStreamedBodyLimitChange(t *testing.T) {
	var raw fasthttp.RequestCtx
	raw.Request.SetBodyStream(strings.NewReader("abcdefgh"), -1)
	defer raw.Request.Reset()

	ctx := &Context{context: &raw, bodyLimit: 4}
	ctx.Body.ctx = ctx

	r, err := ctx.Body.Reader()
	qt.Assert(t, qt.IsNil(err))
	_, err = io.ReadFull(r, make([]byte, 3))
	qt.Assert(t, qt.IsNil(err))

	ctx.SetMaxBodySize(8)
	r, err = ctx.Body.Reader()
	qt.Assert(t, qt.IsNil(err))
	body, err := io.ReadAll(r)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(body), "defgh"))
}

func TestStreamedBodyLoweredLimit(t *testing.T) {
	var raw fasthttp.RequestCtx
	raw.Request.SetBodyStream(strings.NewReader("abcdefgh"), -1)
	defer raw.Request.Reset()

	ctx := &Context{context: &raw, bodyLimit: 8}
	ctx.Body.ctx = ctx

	r, err := ctx.Body.Reader()
	qt.Assert(t, qt.IsNil(err))
	_, err = io.ReadFull(r, make([]byte, 3))
	qt.Assert(t, qt.IsNil(err))

	ctx.SetMaxBodySize(2)
	r, err = ctx.Body.Reader()
	qt.Assert(t, qt.IsNil(err))
	_, err = r.Read(make([]byte, 1))
	qt.Check(t, qt.IsTrue(errors.Is(err, RequestEntityTooLargeError{})))

	ctx.SetMaxBodySize(8)
	_, err = ctx.Body.Reader()
	qt.Check(t, qt.IsTrue(errors.Is(err, RequestEntityTooLargeError{})))
}

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
		body, err := ctx.Body.Bytes()
		qt.Check(t, qt.IsNil(err))
		qt.Check(t, qt.ContentEquals(body, expect), qt.Commentf("Body should be equal to test"))
		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().Post("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK), qt.Commentf("wrong status code"))
}

func TestBodyBytesReturnsTooLarge(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()
	a.Post("/user", func(ctx *Context) {
		ctx.SetMaxBodySize(4)
		_, err := ctx.Body.Bytes()
		if err != nil {
			ctx.Error(err)
			return
		}
		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().Post("/user", []byte("hello"))
	qt.Assert(t, qt.IsNil(err))
	defer fasthttp.ReleaseResponse(resp)
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusRequestEntityTooLarge))
}

func TestBodyStream(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	expect := make([]byte, fasthttp.DefaultMaxRequestBodySize+1024)
	var received []byte

	a.Post("/user", func(ctx *Context) {
		ctx.SetMaxBodySize(len(expect))
		var buf bytes.Buffer
		n, err := ctx.Body.WriteTo(&buf)
		if err != nil {
			ctx.Error(err)
			return
		}

		qt.Check(t, qt.Equals(n, int64(len(expect))))
		received = buf.Bytes()

		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().Post("/user", expect)
	qt.Assert(t, qt.IsNotNil(resp))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK), qt.Commentf("wrong status code"))
	if !bytes.Equal(received, expect) {
		t.Fatal("Received request body should be equal to sent body")
	}
}

func TestBodyWriteToLimit(t *testing.T) {
	for _, tt := range []struct {
		name       string
		streamSize int
		limit      int
		want       string
		tooLarge   bool
	}{
		{"buffered", 0, 4, "", true},
		{"known-length", 8, 4, "", true},
		{"chunked", -1, 4, "abcd", true},
		{"within-limit", -1, 8, "abcdefgh", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var raw fasthttp.RequestCtx
			if tt.name == "buffered" {
				raw.Request.SetBodyString("abcdefgh")
			} else {
				raw.Request.SetBodyStream(strings.NewReader("abcdefgh"), tt.streamSize)
			}
			defer raw.Request.Reset()

			ctx := &Context{context: &raw, bodyLimit: tt.limit}
			ctx.Body.ctx = ctx
			var dst bytes.Buffer
			n, err := ctx.Body.WriteTo(&dst)
			qt.Check(t, qt.Equals(dst.String(), tt.want))
			qt.Check(t, qt.Equals(n, int64(len(tt.want))))
			qt.Check(t, qt.Equals(errors.Is(err, RequestEntityTooLargeError{}), tt.tooLarge))
			if !tt.tooLarge {
				qt.Check(t, qt.IsNil(err))
			}
		})
	}
}

func TestBodyWriteToAfterPartialRead(t *testing.T) {
	var raw fasthttp.RequestCtx
	raw.Request.SetBodyStream(strings.NewReader("abcdefgh"), -1)
	defer raw.Request.Reset()

	ctx := &Context{context: &raw, bodyLimit: 4}
	ctx.Body.ctx = ctx
	r, err := ctx.Body.Reader()
	qt.Assert(t, qt.IsNil(err))
	_, err = io.ReadFull(r, make([]byte, 3))
	qt.Assert(t, qt.IsNil(err))

	var dst bytes.Buffer
	n, err := ctx.Body.WriteTo(&dst)
	qt.Check(t, qt.Equals(n, int64(1)))
	qt.Check(t, qt.Equals(dst.String(), "d"))
	qt.Check(t, qt.ErrorIs(err, RequestEntityTooLargeError{}))
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

		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().PostJSON("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Assert(t, qt.Equals(resp.StatusCode(), http.StatusOK))
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

		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().PutJSON("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Assert(t, qt.Equals(resp.StatusCode(), http.StatusOK))
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

		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().PatchJSON("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Assert(t, qt.Equals(resp.StatusCode(), http.StatusOK))
	qt.Assert(t, qt.DeepEquals(expect, user))
}

func TestBodyJSONValidationError(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	expect := testBodyUser{Name: "test1234567890"}

	a.Patch("/user", func(ctx *Context) {
		ctx.ContentType(http.ContentTypeJSON)

		var user testBodyUser
		if err := ctx.Body.JSON(&user); err != nil {
			ctx.Error(err)

			return
		}

		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().PatchJSON("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Assert(t, qt.Equals(resp.StatusCode(), http.StatusUnprocessableEntity))
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

		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().Put("/user", []byte("<user><name>test</name></user>"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Assert(t, qt.Equals(resp.StatusCode(), http.StatusOK))
	qt.Assert(t, qt.DeepEquals(expect, user))
}

func TestBodyJSONTooLarge(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/user", func(ctx *Context) {
		var u testBodyUser
		if err := ctx.Body.JSON(&u); err != nil {
			ctx.Error(err)
			return
		}

		ctx.StatusCode(http.StatusOK)
	})

	big := bytes.Repeat([]byte("a"), fasthttp.DefaultMaxRequestBodySize+1024)

	resp, err := a.TestClient().Post("/user", big)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusRequestEntityTooLarge))
}

func TestFormLazyAndTooLarge(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	var formErr error

	a.Post("/ignored", func(ctx *Context) {
		ctx.StatusCode(http.StatusOK)
	})
	a.Post("/form", func(ctx *Context) {
		_, formErr = ctx.Form.Values("v")
		ctx.StatusCode(http.StatusOK)
	})

	params := map[string]any{"v": strings.Repeat("a", fasthttp.DefaultMaxRequestBodySize+1024)}

	resp, err := a.TestClient().PostForm("/ignored", params)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))

	resp2, err := a.TestClient().PostForm("/form", params)
	defer fasthttp.ReleaseResponse(resp2)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp2.StatusCode(), http.StatusOK))
	qt.Check(t, qt.ErrorIs(formErr, RequestEntityTooLargeError{}))
}

func TestBodyMultipartStreamLargeUpload(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	size := 3 * fasthttp.DefaultMaxRequestBodySize
	var got int64

	a.Post("/upload", func(ctx *Context) {
		ctx.SetMaxBodySize(size * 2)

		mr, err := ctx.Body.MultipartReader()
		if err != nil {
			ctx.Error(err)
			return
		}

		for {
			p, err := mr.NextPart()
			if err != nil {
				break
			}

			n, err := io.Copy(io.Discard, p)
			if err != nil {
				ctx.Error(err)
				return
			}

			got += n
		}

		ctx.StatusCode(http.StatusOK)
	})
	a.Post("/limited", func(ctx *Context) {
		mr, err := ctx.Body.MultipartReader()
		if err != nil {
			ctx.Error(err)
			return
		}

		for {
			p, err := mr.NextPart()
			if err != nil {
				ctx.Error(err)
				return
			}

			if _, err = io.Copy(io.Discard, p); err != nil {
				ctx.Error(err)
				return
			}
		}
	})

	form := &multipart.Form{Value: map[string][]string{"f": {strings.Repeat("a", size)}}}

	resp, err := a.TestClient().PostMultiPartForm("/upload", form)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))
	qt.Check(t, qt.Equals(got, int64(size)))

	resp2, err := a.TestClient().PostMultiPartForm("/limited", form)
	defer fasthttp.ReleaseResponse(resp2)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp2.StatusCode(), http.StatusRequestEntityTooLarge))
}

func TestFormGetterReturnsTooLarge(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/form", func(ctx *Context) {
		if _, err := ctx.Form.String("v"); err != nil {
			ctx.Error(err)
			return
		}

		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().PostForm("/form", map[string]any{"v": strings.Repeat("a", fasthttp.DefaultMaxRequestBodySize+1024)})
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusRequestEntityTooLarge))
}

func TestBodyBufferedRespectsLowerLimit(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/user", func(ctx *Context) {
		ctx.SetMaxBodySize(3)

		if _, err := ctx.Body.ReadAll(); err != nil {
			ctx.Error(err)
			return
		}

		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().Post("/user", []byte("hello world"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusRequestEntityTooLarge))
}
