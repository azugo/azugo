package azugo

import (
	"crypto/rand"
	"errors"
	"iter"
	"maps"
	"time"

	"azugo.io/azugo/config"

	"azugo.io/core/cache"
	"github.com/goccy/go-json"
	"go.uber.org/zap"
)

// flashData is everything one request hands to the next.
type flashData struct {
	// Messages holds the notices by kind, in insertion order.
	Messages map[string][]string `json:"messages,omitempty"`
	// Errors maps a form field name to its validation message.
	Errors map[string]string `json:"errors,omitempty"`
	// Values holds typed payloads (form models, view state) by key.
	Values map[string]json.RawMessage `json:"values,omitempty"`
}

func (d *flashData) empty() bool {
	return d == nil || (len(d.Messages) == 0 && len(d.Errors) == 0 && len(d.Values) == 0)
}

func flashTTL(conf *config.Configuration) time.Duration {
	if conf.Flash.TTL > 0 {
		return conf.Flash.TTL
	}

	if idle := conf.Server.IdleTimeout; idle > 0 {
		return idle + idle/10
	}

	return time.Minute
}

// FlashCtx carries one-shot state from this request to the next one.
type FlashCtx struct {
	noCopy noCopy

	ctx *Context

	// loaded is the record the previous request left, popped on first read; hasLoaded marks it
	// present and cookieSeen that the request carried a flash cookie at all.
	loaded     flashData
	hasLoaded  bool
	loadTried  bool
	cookieSeen bool
	// pending is what this request queues for the next one.
	pending flashData
	keep    bool
	now     bool
}

func (f *FlashCtx) reset() {
	f.loaded = flashData{}
	f.hasLoaded = false
	f.loadTried = false
	f.cookieSeen = false
	f.pending = flashData{}
	f.keep = false
	f.now = false
}

// Info message for the next request.
func (f *FlashCtx) Info(text string) {
	f.add("info", text)
}

// Success message for the next request.
func (f *FlashCtx) Success(text string) {
	f.add("success", text)
}

// Warning message for the next request.
func (f *FlashCtx) Warning(text string) {
	f.add("warning", text)
}

// Error message for the next request.
func (f *FlashCtx) Error(text string) {
	f.add("error", text)
}

func (f *FlashCtx) add(kind, text string) {
	if f.pending.Messages == nil {
		f.pending.Messages = make(map[string][]string)
	}

	f.pending.Messages[kind] = append(f.pending.Messages[kind], text)
}

// FieldError attaches a validation message to a form field for the next request.
func (f *FlashCtx) FieldError(field, text string) {
	if f.pending.Errors == nil {
		f.pending.Errors = make(map[string]string)
	}

	f.pending.Errors[field] = text
}

// Set stores a JSON-serialisable value (a form model, view state) under key for the next
// request.
func (f *FlashCtx) Set(key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if f.pending.Values == nil {
		f.pending.Values = make(map[string]json.RawMessage)
	}

	f.pending.Values[key] = raw

	return nil
}

// Keep re-flashes what the previous request for one more request.
func (f *FlashCtx) Keep() {
	f.load()
	f.keep = true
}

// Now makes the values queued in this request readable in this request too.
func (f *FlashCtx) Now() {
	f.now = true
}

// Has reports whether there is anything to read.
func (f *FlashCtx) Has() bool {
	f.load()

	return (f.hasLoaded && !f.loaded.empty()) || (f.now && !f.pending.empty())
}

// Infos returns the informational messages for this request.
func (f *FlashCtx) Infos() []string {
	return f.messages("info")
}

// Successes returns the success messages for this request.
func (f *FlashCtx) Successes() []string {
	return f.messages("success")
}

// Warnings returns the warning messages for this request.
func (f *FlashCtx) Warnings() []string {
	return f.messages("warning")
}

// Errors returns the error messages for this request.
func (f *FlashCtx) Errors() []string {
	return f.messages("error")
}

func (f *FlashCtx) messages(kind string) []string {
	f.load()

	var out []string

	for d := range f.sources() {
		out = append(out, d.Messages[kind]...)
	}

	return out
}

// FieldErrors returns every field error for this request.
func (f *FlashCtx) FieldErrors() map[string]string {
	f.load()

	out := make(map[string]string)

	for d := range f.sources() {
		for k, v := range d.Errors {
			out[k] = v
		}
	}

	return out
}

// FieldErrorFor returns the error for field.
func (f *FlashCtx) FieldErrorFor(field string) string {
	f.load()

	for d := range f.sources() {
		if v, ok := d.Errors[field]; ok {
			return v
		}
	}

	return ""
}

// Get decodes the value left under key into v if it was present.
func (f *FlashCtx) Get(key string, v any) (bool, error) {
	f.load()

	for d := range f.sources() {
		raw, ok := d.Values[key]
		if !ok {
			continue
		}

		return true, json.Unmarshal(raw, v)
	}

	return false, nil
}

func (f *FlashCtx) sources() iter.Seq[*flashData] {
	return func(yield func(*flashData) bool) {
		if f.hasLoaded && !yield(&f.loaded) {
			return
		}

		if f.now {
			yield(&f.pending)
		}
	}
}

func (f *FlashCtx) load() {
	if f.loadTried {
		return
	}

	f.loadTried = true

	id := f.ctx.Cookie.Get(f.ctx.app.Config().Flash.CookieName)
	if id == "" {
		return
	}

	f.cookieSeen = true

	store := f.ctx.app.flashCache
	if store == nil {
		f.ctx.Log().Error("flash used before App.Start")

		return
	}

	d, err := store.Pop(f.ctx, id)
	if err != nil {
		var knf cache.KeyNotFoundError
		if !errors.As(err, &knf) {
			f.ctx.Log().Error("failed to load flash data", zap.Error(err))
		}

		return
	}

	f.loaded = d
	f.hasLoaded = true
}

func (f *FlashCtx) commit() {
	if f.keep && f.hasLoaded {
		f.pending.merge(&f.loaded)
	}

	if f.pending.empty() && !f.cookieSeen {
		return
	}

	conf := f.ctx.app.Config()
	name := conf.Flash.CookieName

	if f.pending.empty() {
		f.ctx.Cookie.Clear(name, CookieDefaultSecurity())

		return
	}

	store := f.ctx.app.flashCache
	if store == nil {
		f.ctx.Log().Error("flash used before App.Start")

		return
	}

	id := rand.Text()
	ttl := flashTTL(conf)

	if err := store.Set(f.ctx, id, f.pending, cache.TTL[flashData](ttl)); err != nil {
		f.ctx.Log().Error("failed to save flash data", zap.Error(err))

		return
	}

	if err := store.Sync(f.ctx); err != nil {
		f.ctx.Log().Warn("failed to sync flash data", zap.Error(err))

		return
	}

	f.ctx.Cookie.Set(name, id, CookieDefaultSecurity(), CookieMaxAge(ttl))
}

func (d *flashData) merge(kept *flashData) {
	if d.Messages == nil {
		d.Messages = kept.Messages
	} else {
		for k, v := range kept.Messages {
			d.Messages[k] = append(v, d.Messages[k]...)
		}
	}

	if len(kept.Errors) > 0 {
		maps.Copy(kept.Errors, d.Errors)
		d.Errors = kept.Errors
	}

	if len(kept.Values) > 0 {
		maps.Copy(kept.Values, d.Values)
		d.Values = kept.Values
	}
}
