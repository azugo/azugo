package azugo

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"mime"
	"path/filepath"
	"strings"
)

type staticOptions struct {
	// TrimPrefix is the prefix to trim from the FS path.
	TrimPrefix string

	// ReplacerFunc will be used to replace values in the content of all files.
	ReplacerFunc StaticContentReplacer
}

type StaticOption interface {
	apply(opts *staticOptions)
}

// StaticDirTrimPrefix sets the prefix to trim from the FS path.
type StaticDirTrimPrefix string

func (p StaticDirTrimPrefix) apply(o *staticOptions) {
	o.TrimPrefix = string(p)
}

// StaticContentReplacer is a function that will be called to replace values in the content of all files.
// First return value is the hash to use for caching, second return value is the replacer. If empty string
// is returned for the hash, the content will not be cached.
type StaticContentReplacer func(ctx *Context) (string, *strings.Replacer)

func (r StaticContentReplacer) apply(o *staticOptions) {
	o.ReplacerFunc = r
}

func replaceContent(f embed.FS, file string, replacer *strings.Replacer) ([]byte, bool, error) {
	s, err := f.Open(file)
	if err != nil {
		return nil, false, err
	}
	defer s.Close()

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

func (a *App) StaticEmbedded(path string, f embed.FS, opts ...StaticOption) error {
	opt := &staticOptions{}
	for _, o := range opts {
		o.apply(opt)
	}

	base := path
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}

	if !strings.HasSuffix(base, "/") {
		base += "/"
	}

	altcontent := make(map[string][]byte, 5)
	extcache := make(map[string]string, 10)

	if err := fs.WalkDir(f, ".", func(file string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		fpath := base
		if len(opt.TrimPrefix) > 0 {
			fpath += strings.TrimPrefix(file, opt.TrimPrefix)
		} else {
			fpath += file
		}

		extcache[fpath] = mime.TypeByExtension(filepath.Ext(fpath))

		a.Get(fpath, func(ctx *Context) {
			ctx.Header.Set("Content-Type", extcache[fpath])

			// Modify content on the fly and replace values in content
			if opt.ReplacerFunc != nil {
				hash, replacer := opt.ReplacerFunc(ctx)
				if replacer != nil {
					// Check if already cached
					if content, ok := altcontent[fpath]; len(hash) > 0 && ok {
						ctx.Raw(content)

						return
					}
					if content, ok := altcontent[hash+fpath]; len(hash) > 0 && ok {
						ctx.Raw(content)

						return
					}

					content, changed, err := replaceContent(f, file, replacer)
					if err != nil {
						ctx.Error(err)

						return
					}

					// Update cache
					if len(hash) > 0 {
						if !changed {
							altcontent[fpath] = content
						} else {
							altcontent[hash+fpath] = content
						}
					}

					ctx.Raw(content)

					return
				}
			}

			s, err := f.Open(file)
			if err != nil {
				ctx.Error(err)

				return
			}
			ctx.Stream(s)
		})

		return nil
	}); err != nil {
		return err
	}

	return nil
}
