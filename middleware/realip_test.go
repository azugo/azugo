package middleware

import (
	"fmt"
	"testing"

	"azugo.io/azugo"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestRealIPMiddleware(t *testing.T) {
	for _, test := range []struct {
		name        string
		limit       int
		trusted     []string
		headerName  string
		headerValue string
		expectedIP  string
	}{
		{
			limit:       1,
			trusted:     []string{"X-Forwarded-For"},
			headerName:  "X-Forwarded-For",
			headerValue: "1.1.1.1",
			expectedIP:  "1.1.1.1",
		},
		{
			limit:       2,
			trusted:     []string{"X-Forwarded-For"},
			headerName:  "X-Forwarded-For",
			headerValue: "1.1.1.1",
			expectedIP:  "1.1.1.1",
		},
		{
			limit:       1,
			trusted:     []string{"X-Forwarded-For"},
			headerName:  "X-Forwarded-For",
			headerValue: "1.0.0.1, 1.1.1.1",
			expectedIP:  "1.1.1.1",
		},
		{
			limit:       1,
			trusted:     []string{"X-Forwarded-For"},
			headerName:  "X-Forwarded-For",
			headerValue: "1.0.0.1,1.1.1.1",
			expectedIP:  "1.1.1.1",
		},
		{
			limit:       2,
			trusted:     []string{"X-Real-IP", "X-Forwarded-For"},
			headerName:  "X-Forwarded-For",
			headerValue: "1.0.0.1,1.1.1.1",
			expectedIP:  "1.0.0.1",
		},
		{
			limit:       2,
			trusted:     []string{"X-Real-IP", "X-Forwarded-For"},
			headerName:  "X-Real-IP",
			headerValue: "1.0.0.1",
			expectedIP:  "1.0.0.1",
		},
		{
			limit:       2,
			trusted:     []string{"CF-Connecting-IP"},
			headerName:  "X-Real-IP",
			headerValue: "1.0.0.1",
			expectedIP:  "<nil>",
		},
		{
			limit:       2,
			trusted:     []string{"CF-Connecting-IP"},
			headerName:  "CF-Connecting-IP",
			headerValue: "1.0.0.1",
			expectedIP:  "1.0.0.1",
		},
		{
			limit:       2,
			trusted:     []string{"X-Real-IP"},
			headerName:  "X-Real-IP",
			headerValue: "1.0.0.1, 1.1.1.1",
			expectedIP:  "<nil>",
		},
	} {
		t.Run(fmt.Sprintf("%s: %s limit=%d", test.headerName, test.headerValue, test.limit), func(t *testing.T) {
			a := azugo.NewTestApp()
			defer a.Stop()

			a.RouterOptions().Proxy.Clear()
			a.RouterOptions().Proxy.Add("*")
			a.RouterOptions().Proxy.TrustedHeaders = test.trusted
			a.RouterOptions().Proxy.ForwardLimit = test.limit
			a.Use(RealIP)

			a.Get("/", func(ctx *azugo.Context) {
				qt.Check(t, qt.Equals(ctx.IP().String(), test.expectedIP))
			})

			a.Start(t)
			defer a.Stop()

			c := a.TestClient()
			resp, err := c.Get("/", c.WithHeader(test.headerName, test.headerValue))
			qt.Assert(t, qt.IsNil(err))
			defer fasthttp.ReleaseResponse(resp)
		})
	}
}
