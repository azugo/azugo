package wsfed

import (
	"context"
	"net/url"
	"testing"
	"time"

	"azugo.io/azugo"

	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSigninURL(t *testing.T) {
	a := azugo.NewTestApp()
	a.Start(t)
	defer a.Stop()

	ws, err := New(a.App, "")
	ws.IDPEndpoint = &url.URL{Scheme: "https", Host: "idp.example.local", Path: "/wsfed"}
	ws.clock = clockwork.NewFakeClockAt(time.Date(2022, time.January, 2, 14, 32, 15, 0, time.UTC))
	require.NoError(t, err)

	signinURL, err := ws.SigninURL(context.TODO(), "urn:test", WithRequestParam("lang", "en"))
	require.NoError(t, err)

	u, err := url.Parse(signinURL)
	require.NoError(t, err)

	assert.Equal(t, "https", u.Scheme)
	assert.Equal(t, "idp.example.local", u.Host)
	assert.Equal(t, "/wsfed", u.Path)
	v := u.Query()
	assert.Equal(t, "wsignin1.0", v.Get("wa"), "invalid wa value")
	assert.Equal(t, "urn:test", v.Get("wtrealm"), "invalid wtrealm value")
	assert.Equal(t, "2022-01-02T14:32:15Z", v.Get("wct"), "invalid wct value")
	require.NotEmpty(t, v.Get("wctx"), "wctx is empty")
	assert.Equal(t, "en", v.Get("lang"), "invalid lang value")

	valid, err := ws.NonceStore.Verify(context.TODO(), v.Get("wctx"))
	require.NoError(t, err)
	require.True(t, valid, "wctx is not valid")
}

func TestSignoutURL(t *testing.T) {
	a := azugo.NewTestApp()

	ws, err := New(a.App, "")
	ws.IDPEndpoint = &url.URL{Scheme: "https", Host: "idp.example.local", Path: "/wsfed"}
	require.NoError(t, err)

	signoutURL, err := ws.SignoutURL("urn:test", WithRequestWreply("http://test.local/callback"))
	require.NoError(t, err)

	u, err := url.Parse(signoutURL)
	require.NoError(t, err)

	assert.Equal(t, "https", u.Scheme)
	assert.Equal(t, "idp.example.local", u.Host)
	assert.Equal(t, "/wsfed", u.Path)
	v := u.Query()
	assert.Equal(t, "wsignout1.0", v.Get("wa"), "invalid wa value")
	assert.Equal(t, "urn:test", v.Get("wtrealm"), "invalid wtrealm value")
	assert.Equal(t, "http://test.local/callback", v.Get("wreply"), "invalid wreply value")
}
