package azugo

import (
	"mime/multipart"
	"testing"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestFormValidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/user", func(ctx *Context) {
		v := ctx.Form.Values("multi")
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

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().PostForm("/user", map[string]any{
		"multi": "a,c,b",
		"i":     1,
		"s":     "test",
		"l":     500,
		"b":     true,
		"bb":    0,
	})
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK), qt.Commentf("wrong response status code"))
}

func TestMultiPartFormValidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/user", func(ctx *Context) {
		v := ctx.Form.Values("multi")
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

		ctx.StatusCode(fasthttp.StatusOK)
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

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK), qt.Commentf("wrong response status code"))
}
