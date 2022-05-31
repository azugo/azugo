package proxy

import (
	"bytes"

	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
)

var (
	contentEncodingGzip    = []byte("gzip")
	contentEncodingDeflate = []byte("deflate")
	contentEncodingBr      = []byte("br")
)

var (
	contentTypePlain  = []byte("text/plain")
	contentTypeHTML   = []byte("text/html")
	contentTypeCSS    = []byte("text/css")
	contentTypeJS     = []byte("text/javascript")
	contentTypeJSAlt  = []byte("application/javascript")
	contentTypeJSXAlt = []byte("application/x-javascript")
	contentTypeJSON   = []byte("application/json")
	contentTypeXML    = []byte("application/xml")
	contentTypeXHTML  = []byte("application/xhtml")
)

type replacePair struct {
	from, to []byte
}

type BodyRewriter struct {
	contentTypes [][]byte
	replaceRules []*replacePair

	BasePath       string
	RewriteBaseURL bool
}

func NewBodyRewriter() *BodyRewriter {
	return &BodyRewriter{
		contentTypes: [][]byte{
			contentTypePlain,
			contentTypeHTML,
			contentTypeCSS,
			contentTypeJS,
			contentTypeJSAlt,
			contentTypeJSXAlt,
			contentTypeJSON,
			contentTypeXML,
			contentTypeXHTML,
		},
		RewriteBaseURL: true,
		replaceRules:   make([]*replacePair, 0),
	}
}

var responseBodyPool bytebufferpool.Pool

// AddReplace adds a replacement in response body from upstream.
func (r *BodyRewriter) AddReplace(from, to []byte) {
	if r.replaceRules == nil {
		r.replaceRules = make([]*replacePair, 0, 1)
	}
	r.replaceRules = append(r.replaceRules, &replacePair{from, to})
}

func getDecodedBody(resp *fasthttp.Response) []byte {
	enc := resp.Header.Peek("Content-Encoding")
	if bytes.Equal(enc, contentEncodingGzip) {
		b, err := resp.BodyGunzip()
		if err != nil {
			return nil
		}
		return b
	}
	if bytes.Equal(enc, contentEncodingDeflate) {
		b, err := resp.BodyInflate()
		if err != nil {
			return nil
		}
		return b
	}
	if bytes.Equal(enc, contentEncodingBr) {
		b, err := resp.BodyUnbrotli()
		if err != nil {
			return nil
		}
		return b
	}

	return resp.Body()
}

func setEncodedBody(resp *fasthttp.Response, body []byte) {
	if len(body) == 0 {
		resp.ResetBody()
		return
	}
	enc := resp.Header.Peek("Content-Encoding")
	if bytes.Equal(enc, contentEncodingGzip) {
		w := responseBodyPool.Get()
		body = fasthttp.AppendGzipBytes(w.B, body)
		resp.SetBody(body)
		responseBodyPool.Put(w)
		return
	}
	if bytes.Equal(enc, contentEncodingDeflate) {
		w := responseBodyPool.Get()
		body = fasthttp.AppendDeflateBytes(w.B, body)
		resp.SetBody(body)
		responseBodyPool.Put(w)
		return
	}
	if bytes.Equal(enc, contentEncodingBr) {
		w := responseBodyPool.Get()
		body = fasthttp.AppendBrotliBytes(w.B, body)
		resp.SetBody(body)
		responseBodyPool.Put(w)
		return
	}
	resp.SetBody(body)
}

func trimScheme(url []byte) []byte {
	i := bytes.IndexByte(url, ':')
	if i == -1 {
		return url
	}
	return url[i+1:]
}

// Enabled checks if the rewriter is enabled.
func (r *BodyRewriter) Enabled() bool {
	return len(r.replaceRules) > 0 || r.RewriteBaseURL
}

// RewriteResponse rewrites response body.
func (r *BodyRewriter) RewriteResponse(baseURL, upstream []byte, resp *fasthttp.Response) {
	if !r.Enabled() {
		return
	}

	ct := resp.Header.ContentType()
	isRewritable := false
	for _, ctb := range r.contentTypes {
		if bytes.HasPrefix(ct, ctb) {
			isRewritable = true
			break
		}
	}
	if !isRewritable {
		return
	}

	body := getDecodedBody(resp)
	if len(body) == 0 {
		return
	}
	for _, v := range r.replaceRules {
		body = bytes.ReplaceAll(body, v.from, v.to)
	}
	if r.RewriteBaseURL {
		baseURL = bytes.TrimRight(baseURL, "/")
		upstream = bytes.TrimRight(upstream, "/")
		body = bytes.ReplaceAll(body, upstream, baseURL)
		body = bytes.ReplaceAll(body, trimScheme(upstream), trimScheme(baseURL))
	}
	setEncodedBody(resp, body)
}
