package middleware

import (
	"testing"

	"azugo.io/azugo"

	"azugo.io/core/http"
	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestCrossOriginProtection(t *testing.T) {
	a := azugo.NewTestApp()
	a.Use(CrossOriginProtection(TrustedOrigins{"https://Partner.example"}))

	handler := func(ctx *azugo.Context) {
		ctx.Text("ok")
	}

	a.Get("/form", handler)
	a.Post("/form", handler)

	a.Start(t)
	defer a.Stop()

	c := a.TestClient()

	for name, tc := range map[string]struct {
		method  http.Method
		headers map[string]string
		want    int
	}{
		"safe method cross-site":    {http.MethodGet, map[string]string{http.HeaderSecFetchSite: "cross-site"}, http.StatusOK},
		"no headers":                {http.MethodPost, nil, http.StatusOK},
		"fetch same-origin":         {http.MethodPost, map[string]string{http.HeaderSecFetchSite: "same-origin"}, http.StatusOK},
		"fetch same-site":           {http.MethodPost, map[string]string{http.HeaderSecFetchSite: "same-site"}, http.StatusForbidden},
		"fetch cross-site":          {http.MethodPost, map[string]string{http.HeaderSecFetchSite: "cross-site"}, http.StatusForbidden},
		"origin matches":            {http.MethodPost, map[string]string{http.HeaderOrigin: "http://test"}, http.StatusOK},
		"origin other host":         {http.MethodPost, map[string]string{http.HeaderOrigin: "http://evil.example"}, http.StatusForbidden},
		"trusted origin":            {http.MethodPost, map[string]string{http.HeaderSecFetchSite: "cross-site", http.HeaderOrigin: "https://partner.example"}, http.StatusOK},
		"trusted origin other port": {http.MethodPost, map[string]string{http.HeaderSecFetchSite: "cross-site", http.HeaderOrigin: "https://partner.example:8443"}, http.StatusForbidden},
	} {
		t.Run(name, func(t *testing.T) {
			opts := make([]azugo.TestClientOption, 0, len(tc.headers))
			for k, v := range tc.headers {
				opts = append(opts, c.WithHeader(k, v))
			}

			resp, err := c.Call(tc.method, "/form", nil, opts...)
			defer fasthttp.ReleaseResponse(resp)
			qt.Assert(t, qt.IsNil(err))
			qt.Check(t, qt.Equals(resp.StatusCode(), tc.want))
		})
	}
}

func TestTrustedOriginsRejectsInvalid(t *testing.T) {
	for _, origin := range []string{"partner.example", "https://partner.example/", "https://partner.example/path", "https://user@partner.example", "https://"} {
		qt.Check(t, qt.PanicMatches(func() { CrossOriginProtection(TrustedOrigins{origin}) }, "invalid trusted origin .*"), qt.Commentf("%s", origin))
	}
}
