package azugo

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/valyala/fasthttp"
)

type staticHandler struct {
	// TrimPrefix is the prefix to trim from the FS path.
	TrimPrefix string

	// ReplacerFunc will be used to replace values in the content of all files.
	ReplacerFunc StaticContentReplacer

	// NotFoundPath is the path to use for unknown paths, ex. for SPA routing.
	NotFoundPath string

	fs         *embed.FS
	gzip       bool
	altcontent map[string][]byte
	extcache   map[string]string
}

// StaticOption is an interface for static file serving options.
type StaticOption interface {
	apply(opts *staticHandler)
}

// StaticDirTrimPrefix sets the prefix to trim from the FS path.
type StaticDirTrimPrefix string

func (p StaticDirTrimPrefix) apply(h *staticHandler) {
	h.TrimPrefix = string(p)
}

// StaticContentReplacer is a function that will be called to replace values in the content of all files.
// First return value is the hash to use for caching, second return value is the replacer. If empty string
// is returned for the hash, the content will not be cached.
type StaticContentReplacer func(ctx *Context) (string, *strings.Replacer)

func (r StaticContentReplacer) apply(h *staticHandler) {
	h.ReplacerFunc = r
}

// StaticSPARouterPath sets the path to use for all unknown static content paths for SPA routing.
type StaticSPARouterPath string

func (p StaticSPARouterPath) apply(h *staticHandler) {
	h.NotFoundPath = string(p)
}

// StaticGzipContent enables pre-compressed gzip content that is served directly when the client accepts gzip encoding.
func StaticGzipContent() StaticOption {
	return staticGzipContent(true)
}

type staticGzipContent bool

func (g staticGzipContent) apply(h *staticHandler) {
	h.gzip = bool(g)
}

func (h *staticHandler) replaceContent(file string, replacer *strings.Replacer) ([]byte, bool, error) {
	s, err := h.fs.Open(file)
	if err != nil {
		return nil, false, err
	}
	defer s.Close() //nolint:errcheck

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, s); err != nil {
		return nil, false, err
	}

	content := []byte(replacer.Replace(buf.String()))
	if !bytes.Equal(content, buf.Bytes()) {
		return content, true, nil
	}

	return content, false, nil
}

func (h *staticHandler) requestHandler(fpath, path string) RequestHandler {
	return func(ctx *Context) {
		ctx.Header.Set("Content-Type", h.extcache[fpath])

		useGzip := h.gzip && ctx.Request().Header.HasAcceptEncoding("gzip")
		if h.gzip {
			ctx.Header.Set("Vary", "Accept-Encoding")
		}

		// Modify content on the fly and replace values in content
		if h.ReplacerFunc != nil {
			hash, replacer := h.ReplacerFunc(ctx)
			if replacer != nil {
				if len(hash) > 0 {
					prefix := ""
					if useGzip {
						prefix = "gz:"
					}
					// Check non-hash-specific cache (content didn't change)
					if content, ok := h.altcontent[prefix+fpath]; ok {
						if useGzip {
							ctx.Header.Set("Content-Encoding", "gzip")
						}

						ctx.Raw(content)

						return
					}
					// Check hash-specific cache
					if content, ok := h.altcontent[prefix+hash+fpath]; ok {
						if useGzip {
							ctx.Header.Set("Content-Encoding", "gzip")
						}

						ctx.Raw(content)

						return
					}
				}

				content, changed, err := h.replaceContent(path, replacer)
				if err != nil {
					ctx.Error(err)

					return
				}

				if len(hash) > 0 {
					cacheKey := fpath
					if changed {
						cacheKey = hash + fpath
					}

					h.altcontent[cacheKey] = content

					if h.gzip {
						h.altcontent["gz:"+cacheKey] = fasthttp.AppendGzipBytesLevel(nil, content, fasthttp.CompressBestCompression)
					}

					if useGzip {
						if gz, ok := h.altcontent["gz:"+cacheKey]; ok {
							ctx.Header.Set("Content-Encoding", "gzip")
							ctx.Raw(gz)

							return
						}
					}
				}

				ctx.Raw(content)

				return
			}
		}

		// Serve pre-compressed content if available and client accepts gzip
		if useGzip {
			if content, ok := h.altcontent["gz:"+fpath]; ok {
				ctx.Header.Set("Content-Encoding", "gzip")
				ctx.Raw(content)

				return
			}
		}

		s, err := h.fs.Open(path)
		if err != nil {
			ctx.Error(err)

			return
		}

		ctx.Stream(s)
	}
}

// StaticEmbedded serves files from an embedded filesystem at the given path.
func (a *App) StaticEmbedded(path string, f *embed.FS, opts ...StaticOption) error {
	h := &staticHandler{
		fs:         f,
		altcontent: make(map[string][]byte, 10),
		extcache:   make(map[string]string, 10),
	}
	for _, o := range opts {
		o.apply(h)
	}

	base := path
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}

	if !strings.HasSuffix(base, "/") {
		base += "/"
	}

	if err := fs.WalkDir(f, ".", func(file string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		fpath := base
		if len(h.TrimPrefix) > 0 {
			fpath += strings.TrimPrefix(file, h.TrimPrefix)
		} else {
			fpath += file
		}

		h.extcache[fpath] = mime.TypeByExtension(filepath.Ext(fpath))

		if h.gzip {
			if fh, ferr := f.Open(file); ferr == nil {
				var buf bytes.Buffer

				_, _ = io.Copy(&buf, fh)
				_ = fh.Close()
				h.altcontent["gz:"+fpath] = fasthttp.AppendGzipBytesLevel(nil, buf.Bytes(), fasthttp.CompressBestCompression)
			}
		}

		a.Get(fpath, h.requestHandler(fpath, file))

		return nil
	}); err != nil {
		return err
	}

	if len(h.NotFoundPath) > 0 {
		fpath := h.NotFoundPath
		if !strings.HasPrefix(fpath, "/") {
			fpath = "/" + fpath
		}

		file, err := url.JoinPath(h.TrimPrefix, strings.TrimPrefix(h.NotFoundPath, "/"))
		if err != nil {
			return err
		}

		// Validate that the file exists and pre-compress if needed.
		ff, err := h.fs.Open(file)
		if err != nil {
			return fmt.Errorf("static SPA route handler file not found: %w", err)
		}

		h.extcache[fpath] = mime.TypeByExtension(filepath.Ext(fpath))

		if h.gzip {
			var buf bytes.Buffer

			_, _ = io.Copy(&buf, ff)
			h.altcontent["gz:"+fpath] = fasthttp.AppendGzipBytesLevel(nil, buf.Bytes(), fasthttp.CompressBestCompression)
		}

		_ = ff.Close()

		a.Get(base+"{path:*}", h.requestHandler(fpath, file))
	}

	return nil
}
