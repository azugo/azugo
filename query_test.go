package azugo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestQueryValidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		v := ctx.Query.Values("multi")
		assert.ElementsMatch(t, []string{"a", "b", "c"}, v, "Query parameter multi values should match")

		s, err := ctx.Query.String("s")
		assert.NoError(t, err, "Query parameter s should be present")
		assert.Equal(t, "test", s, "Query parameter s should be equal to test")

		i, err := ctx.Query.Int("i")
		require.NoError(t, err, "Query parameter i should be present")
		assert.Equal(t, 1, i, "Query parameter i should be equal to 1")

		l, err := ctx.Query.Int64("l")
		require.NoError(t, err, "Query parameter l should be present")
		assert.Equal(t, int64(500), l, "Query parameter l should be equal to 500")

		b, err := ctx.Query.Bool("b")
		require.NoError(t, err, "Query parameter b should be present")
		assert.True(t, b, "Query parameter b should be true")

		b, err = ctx.Query.Bool("bb")
		require.NoError(t, err, "Query parameter bb should be present")
		assert.False(t, b, "Query parameter bb should be false")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user?multi=a,c&i=1&multi=b&s=test&l=500&b=TRUE&bb=0")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestQueryRequiredError(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		v := ctx.Query.Values("multi")
		assert.Len(t, v, 0, "Query parameter multi should be empty")

		_, err := ctx.Query.String("s")
		assert.ErrorIs(t, err, ErrParamRequired{"s"}, "Query parameter s should result in required error")
		assert.Equal(t, "Key: 's' Error:Field validation for 's' failed on the 'required' tag", err.(SafeError).SafeError())

		_, err = ctx.Query.Int("i")
		assert.ErrorIs(t, err, ErrParamRequired{"i"}, "Query parameter i should result in required error")
		assert.Equal(t, "Key: 'i' Error:Field validation for 'i' failed on the 'required' tag", err.(SafeError).SafeError())

		_, err = ctx.Query.Int64("l")
		assert.ErrorIs(t, err, ErrParamRequired{"l"}, "Query parameter l should result in required error")
		assert.Equal(t, "Key: 'l' Error:Field validation for 'l' failed on the 'required' tag", err.(SafeError).SafeError())

		_, err = ctx.Query.Bool("b")
		assert.ErrorIs(t, err, ErrParamRequired{"b"}, "Query parameter b should result in required error")
		assert.Equal(t, "Key: 'b' Error:Field validation for 'b' failed on the 'required' tag", err.(SafeError).SafeError())

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestQueryInvalidValueError(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		_, err := ctx.Query.Int("i")
		assert.ErrorAs(t, err, &ErrParamInvalid{}, "Query parameter i should result in invalid parameter error")
		assert.Equal(t, "Key: 'i' Error:Field validation for 'i' failed on the 'numeric' tag", err.(SafeError).SafeError())

		_, err = ctx.Query.Int64("l")
		assert.ErrorAs(t, err, &ErrParamInvalid{}, "Query parameter i should result in invalid parameter error")
		assert.Equal(t, "Key: 'l' Error:Field validation for 'l' failed on the 'numeric' tag", err.(SafeError).SafeError())

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user?i=test&l=test")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}
