package azugo

import (
	"embed"
	"io/fs"
	"mime"
	"path/filepath"
	"strings"
)

type staticOptions struct {
	// TrimPrefix is the prefix to trim from the FS path.
	TrimPrefix string
}

type StaticOption interface {
	apply(opts *staticOptions)
}

// StaticDirTrimPrefix sets the prefix to trim from the FS path.
type StaticDirTrimPrefix string

func (p StaticDirTrimPrefix) apply(o *staticOptions) {
	o.TrimPrefix = string(p)
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

		a.Get(fpath, func(ctx *Context) {
			ctx.Header.Set("Content-Type", mime.TypeByExtension(filepath.Ext(path)))
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
