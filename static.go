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

	// Replacer will be used to replace values in the content of all files.
	Replacer *strings.Replacer
}

type StaticOption interface {
	apply(opts *staticOptions)
}

// StaticDirTrimPrefix sets the prefix to trim from the FS path.
type StaticDirTrimPrefix string

func (p StaticDirTrimPrefix) apply(o *staticOptions) {
	o.TrimPrefix = string(p)
}

type valueReplacer []string

func (r valueReplacer) apply(o *staticOptions) {
	o.Replacer = strings.NewReplacer(r...)
}

// StaticContentReplacer sets the replacer for the static file content.
func StaticContentReplacer(oldnew ...string) StaticOption {
	return valueReplacer(oldnew)
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

		// Check if the content needs to be replaced
		if opt.Replacer != nil {
			go func(file string, f embed.FS) {
				s, err := f.Open(file)
				if err != nil {
					return
				}
				defer s.Close()

				buf := new(bytes.Buffer)
				if _, err := io.Copy(buf, s); err != nil {
					return
				}

				content := []byte(opt.Replacer.Replace(buf.String()))
				if !bytes.Equal(content, buf.Bytes()) {
					altcontent[fpath] = content
				}
			}(file, f)
		}

		extcache[fpath] = mime.TypeByExtension(filepath.Ext(fpath))

		a.Get(fpath, func(ctx *Context) {
			ctx.Header.Set("Content-Type", extcache[fpath])
			if content, ok := altcontent[fpath]; ok {
				ctx.Raw(content)
				return
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
