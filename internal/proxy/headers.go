package proxy

import (
	"azugo.io/core/http"
)

var hopHeaders = []string{
	http.HeaderConnection,
	http.HeaderProxyConnection, // non-standard but still sent by libcurl and rejected by e.g. google
	http.HeaderKeepAlive,
	http.HeaderProxyAuthenticate,
	http.HeaderProxyAuthorization,
	http.HeaderTE,
	http.HeaderTrailer, // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	http.HeaderTransferEncoding,
	http.HeaderUpgrade,
}

// HeaderDel is an interface for deleting HTTP headers.
type HeaderDel interface {
	Del(key string)
}

// StripHeaders removes hop-by-hop headers from the response or request.
func StripHeaders(headers HeaderDel) {
	for _, h := range hopHeaders {
		headers.Del(h)
	}
}
