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
		assert.NoError(t, err, "Query parameter i should be present")
		assert.Equal(t, 1, i, "Query parameter i should be equal to 1")

		l, err := ctx.Query.Int64("l")
		assert.NoError(t, err, "Query parameter l should be present")
		assert.Equal(t, int64(500), l, "Query parameter l should be equal to 500")

		b, err := ctx.Query.Bool("b")
		assert.NoError(t, err, "Query parameter b should be present")
		assert.True(t, b, "Query parameter b should be true")

		b, err = ctx.Query.Bool("bb")
		assert.NoError(t, err, "Query parameter bb should be present")
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
		assert.ErrorIs(t, err, ParamRequiredError{"s"}, "Query parameter s should result in required error")
		assert.Equal(t, "Key: 's' Error:Field validation for 's' failed on the 'required' tag", err.(SafeError).SafeError())

		_, err = ctx.Query.Int("i")
		assert.ErrorIs(t, err, ParamRequiredError{"i"}, "Query parameter i should result in required error")
		assert.Equal(t, "Key: 'i' Error:Field validation for 'i' failed on the 'required' tag", err.(SafeError).SafeError())

		_, err = ctx.Query.Int64("l")
		assert.ErrorIs(t, err, ParamRequiredError{"l"}, "Query parameter l should result in required error")
		assert.Equal(t, "Key: 'l' Error:Field validation for 'l' failed on the 'required' tag", err.(SafeError).SafeError())

		_, err = ctx.Query.Bool("b")
		assert.ErrorIs(t, err, ParamRequiredError{"b"}, "Query parameter b should result in required error")
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
		assert.ErrorAs(t, err, &ParamInvalidError{}, "Query parameter i should result in invalid parameter error")
		assert.Equal(t, "Key: 'i' Error:Field validation for 'i' failed on the 'numeric' tag", err.(SafeError).SafeError())

		_, err = ctx.Query.Int64("l")
		assert.ErrorAs(t, err, &ParamInvalidError{}, "Query parameter i should result in invalid parameter error")
		assert.Equal(t, "Key: 'l' Error:Field validation for 'l' failed on the 'numeric' tag", err.(SafeError).SafeError())

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithQuery(map[string]any{
		"i": "test",
		"l": "test",
	}))
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestQueryOptionalValidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		s := ctx.Query.StringOptional("s")
		assert.NotNil(t, s, "Query parameter s should not be nil")
		if s != nil {
			assert.Equal(t, "test", *s, "Query parameter s should be equal to test")
		}

		i, err := ctx.Query.IntOptional("i")
		assert.NoError(t, err, "Query parameter i should not be nil")
		if i != nil {
			assert.Equal(t, 1, *i, "Query parameter i should be equal to 1")
		}

		l, err := ctx.Query.Int64Optional("l")
		assert.NoError(t, err, "Query parameter l should not be nil")
		if l != nil {
			assert.Equal(t, int64(500), *l, "Query parameter l should be equal to 500")
		}

		b, err := ctx.Query.BoolOptional("b")
		assert.NoError(t, err, "Query parameter b should not be nil")
		if b != nil {
			assert.True(t, *b, "Query parameter b should be true")
		}

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithQuery(map[string]any{
		"s": "test",
		"i": 1,
		"l": 500,
		"b": true,
	}))
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestQueryOptionalNoValues(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		v := ctx.Query.Values("multi")
		assert.Len(t, v, 0, "Query parameter multi should be empty")

		s := ctx.Query.StringOptional("s")
		assert.Nil(t, s, "Query parameter s should be nil")

		i, err := ctx.Query.IntOptional("i")
		assert.NoError(t, err)
		assert.Nil(t, i, "Query parameter i should be nil")

		l, err := ctx.Query.Int64Optional("l")
		assert.NoError(t, err)
		assert.Nil(t, l, "Query parameter l should be nil")

		b, err := ctx.Query.BoolOptional("b")
		assert.NoError(t, err)
		assert.Nil(t, b, "Query parameter b should be nil")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestQueryOptionalInvalidValueError(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		_, err := ctx.Query.IntOptional("i")
		assert.ErrorAs(t, err, &ParamInvalidError{}, "Query parameter i should result in invalid parameter error")
		assert.Equal(t, "Key: 'i' Error:Field validation for 'i' failed on the 'numeric' tag", err.(SafeError).SafeError())

		_, err = ctx.Query.Int64Optional("l")
		assert.ErrorAs(t, err, &ParamInvalidError{}, "Query parameter i should result in invalid parameter error")
		assert.Equal(t, "Key: 'l' Error:Field validation for 'l' failed on the 'numeric' tag", err.(SafeError).SafeError())

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithQuery(map[string]any{
		"i": "test",
		"l": "test",
	}))
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}
