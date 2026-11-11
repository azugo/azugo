package azugo

import (
	"bytes"
	"io"
	"mime/multipart"
	"strings"
	"testing"

	"azugo.io/core/http"
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
		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().Post("/user", expect)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK), qt.Commentf("wrong status code"))
}

func TestBodyStream(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	expect := make([]byte, fasthttp.DefaultMaxRequestBodySize+1024)
	var received []byte

	a.Post("/user", func(ctx *Context) {
		var buf bytes.Buffer
		_, err := ctx.Body.WriteTo(&buf)
		if err != nil {
			ctx.Error(err)
			return
		}

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
		formErr = ctx.Form.Err()
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

func postChunked(t *testing.T, a *TestApp, endpoint string, body []byte, options ...TestClientOption) *fasthttp.Response {
	t.Helper()

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("http://test" + endpoint)
	req.SetBodyStream(bytes.NewReader(body), -1)

	c := a.TestClient()
	c.applyOptions(req, options)

	resp := fasthttp.AcquireResponse()
	qt.Assert(t, qt.IsNil(c.Client().Do(req, resp)))

	return resp
}

func TestBodyJSONChunkedTooLarge(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/user", func(ctx *Context) {
		ctx.SetMaxBodySize(16)

		var u testBodyUser
		if err := ctx.Body.JSON(&u); err != nil {
			ctx.Error(err)
			return
		}

		ctx.StatusCode(http.StatusOK)
	})

	resp := postChunked(t, a, "/user", []byte(`{"name":"`+strings.Repeat("a", 1024)+`"}`))
	defer fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusRequestEntityTooLarge))
}

func TestBodyReadAfterTooLarge(t *testing.T) {
	tests := []struct {
		name  string
		first func(ctx *Context)
	}{
		{
			name:  "Bytes",
			first: func(ctx *Context) { _ = ctx.Body.Bytes() },
		},
		{
			name: "Reader",
			first: func(ctx *Context) {
				if r, err := ctx.Body.Reader(); err == nil {
					_, _ = io.ReadAll(r)
				}
			},
		},
		{
			name: "Bytes then Request.Body",
			first: func(ctx *Context) {
				_ = ctx.Body.Bytes()
				_ = ctx.Request().Body()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewTestApp()
			a.Start(t)
			defer a.Stop()

			var (
				body []byte
				err  error
			)

			a.Post("/user", func(ctx *Context) {
				ctx.SetMaxBodySize(16)

				tt.first(ctx)
				body, err = ctx.Body.ReadAll()

				ctx.StatusCode(http.StatusOK)
			})

			resp := postChunked(t, a, "/user", bytes.Repeat([]byte("a"), 24))
			defer fasthttp.ReleaseResponse(resp)

			qt.Check(t, qt.ErrorIs(err, RequestEntityTooLargeError{}))
			qt.Check(t, qt.IsNil(body))
		})
	}
}

func TestBodyStreamedReadIsCachedWithoutWarning(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	var cached string

	a.Post("/user", func(ctx *Context) {
		var u testBodyUser
		if err := ctx.Body.JSON(&u); err != nil {
			ctx.Error(err)
			return
		}

		cached = string(ctx.Request().Body())

		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().Post("/user", []byte(`{"name":"test"}`))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))
	qt.Check(t, qt.Equals(cached, `{"name":"test"}`))
	qt.Check(t, qt.Equals(a.logs.FilterMessageSnippet("Request.Body()").Len(), 0))
}

func TestBodyReadersShareLimit(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	var (
		read int64
		err  error
	)

	a.Post("/user", func(ctx *Context) {
		ctx.SetMaxBodySize(16)

		for range 2 {
			var r io.Reader
			if r, err = ctx.Body.Reader(); err != nil {
				break
			}

			var n int64

			n, err = io.Copy(io.Discard, io.LimitReader(r, 16))
			read += n

			if err != nil {
				break
			}
		}

		ctx.StatusCode(http.StatusOK)
	})

	resp := postChunked(t, a, "/user", bytes.Repeat([]byte("a"), 32))
	defer fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.ErrorIs(err, RequestEntityTooLargeError{}))
	qt.Check(t, qt.Equals(read, int64(16)))
}

func TestFormAfterBodyTooLarge(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	var (
		role string
		err  error
	)

	a.Post("/form", func(ctx *Context) {
		ctx.SetMaxBodySize(16)

		var u testBodyUser

		_ = ctx.Body.JSON(&u)

		role, err = ctx.Form.String("role")

		ctx.StatusCode(http.StatusOK)
	})

	body := []byte("v=" + strings.Repeat("a", 15) + "&role=admin")

	resp := postChunked(t, a, "/form", body, a.TestClient().WithHeader("Content-Type", "application/x-www-form-urlencoded"))
	defer fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.ErrorIs(err, RequestEntityTooLargeError{}))
	qt.Check(t, qt.Equals(role, ""))
}

func TestBodyWriteToAfterTooLarge(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	var (
		buf bytes.Buffer
		err error
	)

	a.Post("/user", func(ctx *Context) {
		ctx.SetMaxBodySize(16)

		_ = ctx.Body.Bytes()
		_, err = ctx.Body.WriteTo(&buf)

		ctx.StatusCode(http.StatusOK)
	})

	resp := postChunked(t, a, "/user", bytes.Repeat([]byte("a"), 24))
	defer fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.ErrorIs(err, RequestEntityTooLargeError{}))
	qt.Check(t, qt.Equals(buf.Len(), 0))
}
