package azugo

import (
	"bytes"
	"compress/gzip"
	"mime/multipart"
	"strings"
	"testing"

	"azugo.io/core/http"
	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestGzipMultipartFormBodyLimit(t *testing.T) {
	const limit = 4096

	makeBody := func(size int) []byte {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		qt.Assert(t, qt.IsNil(writer.SetBoundary("boundary")))
		qt.Assert(t, qt.IsNil(writer.WriteField("name", "")))
		qt.Assert(t, qt.IsNil(writer.Close()))
		empty := body.String()
		result := []byte(strings.Replace(empty, "\r\n--boundary--", strings.Repeat("x", size-len(empty))+"\r\n--boundary--", 1))
		qt.Assert(t, qt.HasLen(result, size))

		return result
	}

	compress := func(body []byte) []byte {
		t.Helper()
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		_, err := writer.Write(body)
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.IsNil(writer.Close()))
		return compressed.Bytes()
	}

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()
	a.Post("/form", func(ctx *Context) {
		ctx.SetMaxBodySize(limit)
		value, err := ctx.Form.String("name")
		if err != nil {
			ctx.Error(err)
			return
		}
		ctx.Text(value)
	})

	bomb := compress(makeBody(limit * 100))
	qt.Assert(t, qt.IsTrue(len(bomb) < limit))

	for _, tt := range []struct {
		name   string
		body   []byte
		status int
	}{
		{"at-limit", compress(makeBody(limit)), http.StatusOK},
		{"over-limit", compress(makeBody(limit + 1)), http.StatusRequestEntityTooLarge},
		{"compressed-bomb", bomb, http.StatusRequestEntityTooLarge},
		{"trailing-decoded-data", compress(append(makeBody(limit), 'x')), http.StatusRequestEntityTooLarge},
		{"wire-over-limit", bytes.Repeat([]byte("x"), limit+1), http.StatusRequestEntityTooLarge},
		{"invalid-gzip", []byte("not gzip"), http.StatusBadRequest},
		{"invalid-multipart", compress([]byte("not multipart")), http.StatusBadRequest},
	} {
		for _, mode := range []string{"buffered", "streamed", "known-length-stream"} {
			t.Run(tt.name+"/"+mode, func(t *testing.T) {
				var raw fasthttp.RequestCtx
				raw.Request.Header.SetMethod("POST")
				raw.Request.SetRequestURI("/form")
				raw.Request.Header.SetContentType("multipart/form-data; boundary=boundary")
				raw.Request.Header.Set(http.HeaderContentEncoding, "gzip")
				if mode == "streamed" {
					raw.Request.SetBodyStream(bytes.NewReader(tt.body), -1)
				} else if mode == "known-length-stream" {
					raw.Request.SetBodyStream(bytes.NewReader(tt.body), len(tt.body))
				} else {
					raw.Request.SetBody(tt.body)
				}
				defer raw.Request.Reset()

				a.Handler(&raw)
				qt.Check(t, qt.Equals(raw.Response.StatusCode(), tt.status), qt.Commentf("%s", raw.Response.Body()))
			})
		}
	}
}

func TestFormMediaTypes(t *testing.T) {
	var multipartBody bytes.Buffer
	writer := multipart.NewWriter(&multipartBody)
	qt.Assert(t, qt.IsNil(writer.WriteField("name", "test")))
	qt.Assert(t, qt.IsNil(writer.Close()))

	tests := []struct {
		name        string
		contentType string
		body        []byte
		wantValue   bool
	}{
		{"urlencoded", "Application/X-Www-Form-Urlencoded; charset=utf-8", []byte("name=test"), true},
		{"multipart", "Multipart/Form-Data; BOUNDARY=" + writer.Boundary(), multipartBody.Bytes(), true},
		{"invalid-urlencoded-prefix", "application/x-www-form-urlencoded-extra", []byte("name=test"), false},
		{"invalid-multipart-prefix", "multipart/form-data-extra; boundary=" + writer.Boundary(), multipartBody.Bytes(), false},
	}

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()
	a.Post("/form", func(ctx *Context) {
		value, err := ctx.Form.String("name")
		if err != nil {
			ctx.Error(err)
			return
		}
		ctx.Text(value)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := a.TestClient().Post("/form", tt.body,
				a.TestClient().WithHeader(http.HeaderContentType, tt.contentType))
			qt.Assert(t, qt.IsNil(err))
			defer fasthttp.ReleaseResponse(resp)

			if tt.wantValue {
				qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))
				qt.Check(t, qt.Equals(string(resp.Body()), "test"))
			} else {
				qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusBadRequest))
			}
		})
	}
}

func TestFormValidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/user", func(ctx *Context) {
		v, err := ctx.Form.Values("multi")
		qt.Check(t, qt.IsNil(err))
		qt.Check(t, qt.ContentEquals(v, []string{"a", "b", "c"}), qt.Commentf("Form parameter multi values should match"))

		s, err := ctx.Form.String("s")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Form parameter s should be present"))
		qt.Check(t, qt.Equals(s, "test"), qt.Commentf("Form parameter s should be equal to test"))

		i, err := ctx.Form.Int("i")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Form parameter i should be present"))
		qt.Check(t, qt.Equals(i, 1), qt.Commentf("Form parameter i should be equal to 1"))

		l, err := ctx.Form.Int64("l")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Form parameter l should be present"))
		qt.Check(t, qt.Equals(l, int64(500)), qt.Commentf("Form parameter l should be equal to 500"))

		b, err := ctx.Form.Bool("b")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Form parameter b should be present"))
		qt.Check(t, qt.IsTrue(b), qt.Commentf("Form parameter b should be true"))

		b, err = ctx.Form.Bool("bb")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Form parameter bb should be present"))
		qt.Check(t, qt.IsFalse(b), qt.Commentf("Form parameter bb should be false"))

		cb, err := ctx.Form.BoolOptional("cb")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Form parameter cb should be present"))
		qt.Check(t, qt.DeepEquals(cb, new(true)), qt.Commentf("Checkbox value on should be true"))

		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().PostForm("/user", map[string]any{
		"multi": "a,c,b",
		"i":     1,
		"s":     "test",
		"l":     500,
		"b":     true,
		"bb":    0,
		"cb":    "on",
	})
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK), qt.Commentf("wrong response status code"))
}

func TestFormOptionalGetters(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/user", func(ctx *Context) {
		value, err := ctx.Form.StringOptional("name")
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.IsNotNil(value))
		qt.Check(t, qt.Equals(*value, "test"))

		missing, err := ctx.Form.StringOptional("missing")
		qt.Check(t, qt.IsNil(err))
		qt.Check(t, qt.IsNil(missing))

		file, err := ctx.Form.FileOptional("upload")
		qt.Assert(t, qt.IsNil(err))
		qt.Assert(t, qt.IsNotNil(file))
		qt.Check(t, qt.Equals(file.Filename, "test.txt"))
		files, err := ctx.Form.Files("upload")
		qt.Check(t, qt.IsNil(err))
		qt.Check(t, qt.Equals(len(files), 1))
		missingFile, err := ctx.Form.FileOptional("missing")
		qt.Check(t, qt.IsNil(err))
		qt.Check(t, qt.IsNil(missingFile))
		ctx.StatusCode(http.StatusOK)
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	qt.Assert(t, qt.IsNil(writer.WriteField("name", "test")))
	part, err := writer.CreateFormFile("upload", "test.txt")
	qt.Assert(t, qt.IsNil(err))
	_, err = part.Write([]byte("content"))
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.IsNil(writer.Close()))

	resp, err := a.TestClient().Post("/user", body.Bytes(), a.TestClient().WithHeader(http.HeaderContentType, writer.FormDataContentType()))
	qt.Assert(t, qt.IsNil(err))
	defer fasthttp.ReleaseResponse(resp)
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))
}

func TestFormOptionalGettersReturnParseError(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/user", func(ctx *Context) {
		ctx.SetMaxBodySize(4)

		values, err := ctx.Form.Values("name")
		qt.Check(t, qt.IsNil(values))
		qt.Check(t, qt.ErrorIs(err, RequestEntityTooLargeError{}))
		value, err := ctx.Form.StringOptional("name")
		qt.Check(t, qt.IsNil(value))
		qt.Check(t, qt.ErrorIs(err, RequestEntityTooLargeError{}))
		file, err := ctx.Form.FileOptional("upload")
		qt.Check(t, qt.IsNil(file))
		qt.Check(t, qt.ErrorIs(err, RequestEntityTooLargeError{}))
		files, err := ctx.Form.Files("upload")
		qt.Check(t, qt.IsNil(files))
		qt.Check(t, qt.ErrorIs(err, RequestEntityTooLargeError{}))
		_, err = ctx.Form.Int64Optional("number")
		qt.Check(t, qt.ErrorIs(err, RequestEntityTooLargeError{}))
		_, err = ctx.Form.BoolOptional("flag")
		qt.Check(t, qt.ErrorIs(err, RequestEntityTooLargeError{}))
		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().PostForm("/user", map[string]any{"name": "test"})
	qt.Assert(t, qt.IsNil(err))
	defer fasthttp.ReleaseResponse(resp)
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))
}

func TestFormOptionalGettersReturnMalformedMultipartError(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/user", func(ctx *Context) {
		_, err := ctx.Form.StringOptional("name")
		var badRequest BadRequestError
		qt.Check(t, qt.ErrorAs(err, &badRequest))
		_, err = ctx.Form.Files("upload")
		qt.Check(t, qt.ErrorAs(err, &badRequest))
		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().Post("/user", []byte("not multipart"),
		a.TestClient().WithHeader(http.HeaderContentType, "multipart/form-data; boundary=test"))
	qt.Assert(t, qt.IsNil(err))
	defer fasthttp.ReleaseResponse(resp)
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))
}

func TestQueryMethodFormParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Query("/user", func(ctx *Context) {
		s, err := ctx.Form.String("s")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Form parameter s should be present"))
		qt.Check(t, qt.Equals(s, "test"), qt.Commentf("Form parameter s should be equal to test"))

		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().QueryForm("/user", map[string]any{
		"s": "test",
	})
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK), qt.Commentf("wrong response status code"))
}

func TestMultiPartFormValidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/user", func(ctx *Context) {
		v, err := ctx.Form.Values("multi")
		qt.Check(t, qt.IsNil(err))
		qt.Check(t, qt.ContentEquals(v, []string{"a", "b", "c"}), qt.Commentf("Form parameter multi values should match"))

		s, err := ctx.Form.String("s")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Form parameter s should be present"))
		qt.Check(t, qt.Equals(s, "test"), qt.Commentf("Form parameter s should be equal to test"))

		i, err := ctx.Form.Int("i")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Form parameter i should be present"))
		qt.Check(t, qt.Equals(i, 1), qt.Commentf("Form parameter i should be equal to 1"))

		l, err := ctx.Form.Int64("l")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Form parameter l should be present"))
		qt.Check(t, qt.Equals(l, int64(500)), qt.Commentf("Form parameter l should be equal to 500"))

		b, err := ctx.Form.Bool("b")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Form parameter b should be present"))
		qt.Check(t, qt.IsTrue(b), qt.Commentf("Form parameter b should be true"))

		b, err = ctx.Form.Bool("bb")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Form parameter bb should be present"))
		qt.Check(t, qt.IsFalse(b), qt.Commentf("Form parameter bb should be false"))

		ctx.StatusCode(http.StatusOK)
	})

	form := &multipart.Form{
		Value: make(map[string][]string),
		File:  make(map[string][]*multipart.FileHeader),
	}

	form.Value["multi"] = []string{"a", "c", "b"}
	form.Value["i"] = []string{"1"}
	form.Value["s"] = []string{"test"}
	form.Value["l"] = []string{"500"}
	form.Value["b"] = []string{"TRUE"}
	form.Value["bb"] = []string{"0"}

	resp, err := a.TestClient().PostMultiPartForm("/user", form)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK), qt.Commentf("wrong response status code"))
}
