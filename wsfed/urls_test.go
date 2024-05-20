package wsfed

import (
	"context"
	"net/url"
	"testing"
	"time"

	"azugo.io/azugo"

	"github.com/go-quicktest/qt"
	"github.com/jonboulle/clockwork"
)

func TestSigninURL(t *testing.T) {
	a := azugo.NewTestApp()
	a.Start(t)
	defer a.Stop()

	ws, err := New(a.App, "")
	ws.IDPEndpoint = &url.URL{Scheme: "https", Host: "idp.example.local", Path: "/wsfed"}
	ws.clock = clockwork.NewFakeClockAt(time.Date(2022, time.January, 2, 14, 32, 15, 0, time.UTC))
	qt.Assert(t, qt.IsNil(err))

	signinURL, err := ws.SigninURL(context.TODO(), "urn:test", WithRequestParam("lang", "en"))
	qt.Assert(t, qt.IsNil(err))

	// Wait for cache to sync
	time.Sleep(50 * time.Millisecond)

	u, err := url.Parse(signinURL)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(u.Scheme, "https"))
	qt.Check(t, qt.Equals(u.Host, "idp.example.local"))
	qt.Check(t, qt.Equals(u.Path, "/wsfed"))

	v := u.Query()
	qt.Assert(t, qt.Equals(v.Get("wa"), "wsignin1.0"), qt.Commentf("invalid wa value"))
	qt.Assert(t, qt.Equals(v.Get("wtrealm"), "urn:test"), qt.Commentf("invalid wtrealm value"))
	qt.Assert(t, qt.Equals(v.Get("wct"), "2022-01-02T14:32:15Z"), qt.Commentf("invalid wct value"))
	qt.Assert(t, qt.Not(qt.HasLen(v.Get("wctx"), 0)), qt.Commentf("wctx is empty"))
	qt.Assert(t, qt.Equals(v.Get("lang"), "en"), qt.Commentf("invalid lang value"))

	valid, err := ws.NonceStore.Verify(context.TODO(), v.Get("wctx"))
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(valid), qt.Commentf("wctx is not valid"))
}

func TestSignoutURL(t *testing.T) {
	a := azugo.NewTestApp()

	ws, err := New(a.App, "")
	ws.IDPEndpoint = &url.URL{Scheme: "https", Host: "idp.example.local", Path: "/wsfed"}
	qt.Assert(t, qt.IsNil(err))

	signoutURL, err := ws.SignoutURL("urn:test", WithRequestWreply("http://test.local/callback"))
	qt.Assert(t, qt.IsNil(err))

	u, err := url.Parse(signoutURL)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(u.Scheme, "https"))
	qt.Check(t, qt.Equals(u.Host, "idp.example.local"))
	qt.Check(t, qt.Equals(u.Path, "/wsfed"))

	v := u.Query()
	qt.Assert(t, qt.Equals(v.Get("wa"), "wsignout1.0"), qt.Commentf("invalid wa value"))
	qt.Assert(t, qt.Equals(v.Get("wtrealm"), "urn:test"), qt.Commentf("invalid wtrealm value"))
	qt.Assert(t, qt.Equals(v.Get("wreply"), "http://test.local/callback"), qt.Commentf("invalid wreply value"))
}
