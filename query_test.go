package azugo

import (
	"testing"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestQueryValidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		v := ctx.Query.Values("multi")
		qt.Check(t, qt.ContentEquals(v, []string{"a", "b", "c"}), qt.Commentf("Query parameter multi values should match"))

		s, err := ctx.Query.String("s")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Query parameter s should not be nil"))
		qt.Check(t, qt.Equals(s, "test"), qt.Commentf("Query parameter s should be equal to test"))

		i, err := ctx.Query.Int("i")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Query parameter i should not be nil"))
		qt.Check(t, qt.Equals(i, 1), qt.Commentf("Query parameter i should be equal to 1"))

		l, err := ctx.Query.Int64("l")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Query parameter l should not be nil"))
		qt.Check(t, qt.Equals(l, int64(500)), qt.Commentf("Query parameter l should be equal to 500"))

		b, err := ctx.Query.Bool("b")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Query parameter b should not be nil"))
		qt.Check(t, qt.IsTrue(b), qt.Commentf("Query parameter b should be true"))

		b, err = ctx.Query.Bool("bb")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Query parameter bb should not be nil"))
		qt.Check(t, qt.IsFalse(b), qt.Commentf("Query parameter bb should be false"))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user?multi=a,c&i=1&multi=b&s=test&l=500&b=TRUE&bb=0")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
}

func TestQueryRequiredError(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		v := ctx.Query.Values("multi")
		qt.Check(t, qt.HasLen(v, 0), qt.Commentf("Query parameter multi should be empty"))

		_, err := ctx.Query.String("s")
		qt.Check(t, qt.ErrorIs(err, ParamRequiredError{"s"}), qt.Commentf("Query parameter s should result in required error"))
		qt.Check(t, qt.Equals(err.(SafeError).SafeError(), "Key: 's' Error:Field validation for 's' failed on the 'required' tag"))

		_, err = ctx.Query.Int("i")
		qt.Check(t, qt.ErrorIs(err, ParamRequiredError{"i"}), qt.Commentf("Query parameter i should result in required error"))
		qt.Check(t, qt.Equals(err.(SafeError).SafeError(), "Key: 'i' Error:Field validation for 'i' failed on the 'required' tag"))

		_, err = ctx.Query.Int64("l")
		qt.Check(t, qt.ErrorIs(err, ParamRequiredError{"l"}), qt.Commentf("Query parameter l should result in required error"))
		qt.Check(t, qt.Equals(err.(SafeError).SafeError(), "Key: 'l' Error:Field validation for 'l' failed on the 'required' tag"))

		_, err = ctx.Query.Bool("b")
		qt.Check(t, qt.ErrorIs(err, ParamRequiredError{"b"}), qt.Commentf("Query parameter b should result in required error"))
		qt.Check(t, qt.Equals(err.(SafeError).SafeError(), "Key: 'b' Error:Field validation for 'b' failed on the 'required' tag"))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
}

func TestQueryInvalidValueError(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		_, err := ctx.Query.Int("i")
		qt.Check(t, qt.ErrorAs(err, &ParamInvalidError{}), qt.Commentf("Query parameter i should result in invalid parameter error"))
		qt.Check(t, qt.Equals(err.(SafeError).SafeError(), "Key: 'i' Error:Field validation for 'i' failed on the 'numeric' tag"))

		_, err = ctx.Query.Int64("l")
		qt.Check(t, qt.ErrorAs(err, &ParamInvalidError{}), qt.Commentf("Query parameter i should result in invalid parameter error"))
		qt.Check(t, qt.Equals(err.(SafeError).SafeError(), "Key: 'l' Error:Field validation for 'l' failed on the 'numeric' tag"))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithQuery(map[string]any{
		"i": "test",
		"l": "test",
	}))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
}

func TestQueryOptionalValidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		s := ctx.Query.StringOptional("s")
		if qt.Check(t, qt.IsNotNil(s), qt.Commentf("Query parameter s should not be nil")) {
			qt.Check(t, qt.Equals(*s, "test"), qt.Commentf("Query parameter s should be equal to test"))
		}

		i, err := ctx.Query.IntOptional("i")
		if qt.Check(t, qt.IsNil(err), qt.Commentf("Query parameter i should not be nil")) {
			qt.Check(t, qt.Equals(*i, 1), qt.Commentf("Query parameter i should be equal to 1"))
		}

		l, err := ctx.Query.Int64Optional("l")
		if qt.Check(t, qt.IsNil(err), qt.Commentf("Query parameter l should not be nil")) {
			qt.Check(t, qt.Equals(*l, int64(500)), qt.Commentf("Query parameter l should be equal to 500"))
		}

		b, err := ctx.Query.BoolOptional("b")
		if qt.Check(t, qt.IsNil(err), qt.Commentf("Query parameter b should not be nil")) {
			qt.Check(t, qt.IsTrue(*b), qt.Commentf("Query parameter b should be true"))
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
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
}

func TestQueryOptionalNoValues(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		v := ctx.Query.Values("multi")
		qt.Check(t, qt.HasLen(v, 0), qt.Commentf("Query parameter multi should be empty"))

		s := ctx.Query.StringOptional("s")
		qt.Check(t, qt.IsNil(s), qt.Commentf("Query parameter s should be nil"))

		i, err := ctx.Query.IntOptional("i")
		if qt.Check(t, qt.IsNil(err)) {
			qt.Check(t, qt.IsNil(i), qt.Commentf("Query parameter i should be nil"))
		}

		l, err := ctx.Query.Int64Optional("l")
		if qt.Check(t, qt.IsNil(err)) {
			qt.Check(t, qt.IsNil(l), qt.Commentf("Query parameter l should be nil"))
		}

		b, err := ctx.Query.BoolOptional("b")
		if qt.Check(t, qt.IsNil(err)) {
			qt.Check(t, qt.IsNil(b), qt.Commentf("Query parameter b should be nil"))
		}

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
}

func TestQueryOptionalInvalidValueError(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		_, err := ctx.Query.IntOptional("i")
		if qt.Check(t, qt.ErrorAs(err, &ParamInvalidError{}), qt.Commentf("Query parameter i should result in invalid parameter error")) {
			qt.Check(t, qt.Equals(err.(SafeError).SafeError(), "Key: 'i' Error:Field validation for 'i' failed on the 'numeric' tag"))
		}

		_, err = ctx.Query.Int64Optional("l")
		if qt.Check(t, qt.ErrorAs(err, &ParamInvalidError{}), qt.Commentf("Query parameter i should result in invalid parameter error")) {
			qt.Check(t, qt.Equals(err.(SafeError).SafeError(), "Key: 'l' Error:Field validation for 'l' failed on the 'numeric' tag"))
		}

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithQuery(map[string]any{
		"i": "test",
		"l": "test",
	}))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
}
