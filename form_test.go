package azugo

import (
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestFormValidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/user", func(ctx *Context) {
		v := ctx.Form.Values("multi")
		assert.ElementsMatch(t, []string{"a", "b", "c"}, v, "Form parameter multi values should match")

		s, err := ctx.Form.String("s")
		assert.NoError(t, err, "Form parameter s should be present")
		assert.Equal(t, "test", s, "Form parameter s should be equal to test")

		i, err := ctx.Form.Int("i")
		assert.NoError(t, err, "Form parameter i should be present")
		assert.Equal(t, 1, i, "Form parameter i should be equal to 1")

		l, err := ctx.Form.Int64("l")
		assert.NoError(t, err, "Form parameter l should be present")
		assert.Equal(t, int64(500), l, "Form parameter l should be equal to 500")

		b, err := ctx.Form.Bool("b")
		assert.NoError(t, err, "Form parameter b should be present")
		assert.True(t, b, "Form parameter b should be true")

		b, err = ctx.Form.Bool("bb")
		assert.NoError(t, err, "Form parameter bb should be present")
		assert.False(t, b, "Form parameter bb should be false")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().PostForm("/user", map[string]interface{}{
		"multi": "a,c,b",
		"i":     1,
		"s":     "test",
		"l":     500,
		"b":     true,
		"bb":    0,
	})
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestMultiPartFormValidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/user", func(ctx *Context) {
		v := ctx.Form.Values("multi")
		assert.ElementsMatch(t, []string{"a", "b", "c"}, v, "Form parameter multi values should match")

		s, err := ctx.Form.String("s")
		assert.NoError(t, err, "Form parameter s should be present")
		assert.Equal(t, "test", s, "Form parameter s should be equal to test")

		i, err := ctx.Form.Int("i")
		assert.NoError(t, err, "Form parameter i should be present")
		assert.Equal(t, 1, i, "Form parameter i should be equal to 1")

		l, err := ctx.Form.Int64("l")
		assert.NoError(t, err, "Form parameter l should be present")
		assert.Equal(t, int64(500), l, "Form parameter l should be equal to 500")

		b, err := ctx.Form.Bool("b")
		assert.NoError(t, err, "Form parameter b should be present")
		assert.True(t, b, "Form parameter b should be true")

		b, err = ctx.Form.Bool("bb")
		assert.NoError(t, err, "Form parameter bb should be present")
		assert.False(t, b, "Form parameter bb should be false")

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
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}
