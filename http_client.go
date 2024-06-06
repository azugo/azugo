package azugo

import (
	"fmt"

	"azugo.io/core/http"
)

// HTTPClient returns HTTP client instance.
func (a *App) HTTPClient() http.Client {
	a.httpSync.RLock()
	defer a.httpSync.RUnlock()

	c := a.http
	if c == nil {
		a.httpSync.RUnlock()
		a.httpSync.Lock()

		c = http.NewClient(
			append([]http.Option{
				a.Config().HTTPClient,
				http.Instrumenter(a.Instrumenter()),
				http.UserAgent(fmt.Sprintf("%s/%s", a.AppName, a.AppVer)),
			}, a.httpOpts...)...,
		)
		a.http = c

		a.httpSync.Unlock()
		a.httpSync.RLock()
	}

	return c
}

// AddHTTPClientOption adds a additional option to HTTP client.
func (a *App) AddHTTPClientOption(opt http.Option) {
	a.httpSync.Lock()
	defer a.httpSync.Unlock()

	a.httpOpts = append(a.httpOpts, opt)
	a.http = nil
}

// HTTPClient returns the HTTP client with the current context.
func (c *Context) HTTPClient() http.Client {
	return c.app.HTTPClient().WithContext(c)
}
