package azugo

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

const otelParentSpanContextKey = "__otelParentSpanContext"

// Deadline returns the time when work done on behalf of this context
// should be canceled. Deadline returns ok==false when no deadline is
// set. Successive calls to Deadline return the same results.
func (c *Context) Deadline() (time.Time, bool) {
	if c == nil || c.context == nil {
		return time.Time{}, false
	}
	if p := c.OtelBridge(); p != nil {
		return p.Deadline()
	}

	return c.context.Deadline()
}

func (c *Context) OtelBridge() context.Context {
	if c == nil {
		return nil
	}

	ctxVal := c.UserValue(otelParentSpanContextKey)
	if ctxVal == nil {
		return nil
	}

	if self, ok := ctxVal.(*Context); ok && self == c {
		return nil
	}

	parentCtx, ok := ctxVal.(context.Context)
	if !ok {
		return nil
	}

	spanCtx := trace.SpanContextFromContext(parentCtx)
	if !spanCtx.IsValid() {
		return context.Background()
	}

	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)
	if span := trace.SpanFromContext(parentCtx); span != nil {
		return trace.ContextWithSpan(ctx, span)
	}

	return ctx
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled. Done may return nil if this context can
// never be canceled. Successive calls to Done return the same value.
// The close of the Done channel may happen asynchronously,
// after the cancel function returns.
//
// WithCancel arranges for Done to be closed when cancel is called;
// WithDeadline arranges for Done to be closed when the deadline
// expires; WithTimeout arranges for Done to be closed when the timeout
// elapses.
//
// Done is provided for use in select statements:
//
//	// Stream generates values with DoSomething and sends them to out
//	// until DoSomething returns an error or ctx.Done is closed.
//	func Stream(ctx context.Context, out chan<- Value) error {
//		for {
//			v, err := DoSomething(ctx)
//			if err != nil {
//				return err
//			}
//			select {
//			case <-ctx.Done():
//				return ctx.Err()
//			case out <- v:
//			}
//		}
//	}
//
// See https://blog.golang.org/pipelines for more examples of how to use
// a Done channel for cancellation.
func (c *Context) Done() <-chan struct{} {
	if c == nil || c.context == nil {
		return nil
	}

	if p := c.OtelBridge(); p != nil {
		return p.Done()
	}

	return c.context.Done()
}

// If Done is not yet closed, Err returns nil.
// If Done is closed, Err returns a non-nil error explaining why:
// Canceled if the context was canceled
// or DeadlineExceeded if the context's deadline passed.
// After Err returns a non-nil error, successive calls to Err return the same error.
func (c *Context) Err() error {
	if c == nil || c.context == nil {
		return nil
	}

	if p := c.OtelBridge(); p != nil {
		return p.Err()
	}

	return c.context.Err()
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
//
// Use context values only for request-scoped data that transits
// processes and API boundaries, not for passing optional parameters to
// functions.
//
// A key identifies a specific value in a Context. Functions that wish
// to store values in Context typically allocate a key in a global
// variable then use that key as the argument to context.WithValue and
// Context.Value. A key can be any type that supports equality;
// packages should define keys as an unexported type to avoid
// collisions.
//
// Packages that define a Context key should provide type-safe accessors
// for the values stored using that key:
//
//	// Package user defines a User type that's stored in Contexts.
//	package user
//
//	import "context"
//
//	// User is the type of value stored in the Contexts.
//	type User struct {...}
//
//	// key is an unexported type for keys defined in this package.
//	// This prevents collisions with keys defined in other packages.
//	type key int
//
//	// userKey is the key for user.User values in Contexts. It is
//	// unexported; clients use user.NewContext and user.FromContext
//	// instead of using this key directly.
//	var userKey key
//
//	// NewContext returns a new Context that carries value u.
//	func NewContext(ctx context.Context, u *User) context.Context {
//		return context.WithValue(ctx, userKey, u)
//	}
//
//	// FromContext returns the User value stored in ctx, if any.
//	func FromContext(ctx context.Context) (*User, bool) {
//		u, ok := ctx.Value(userKey).(*User)
//		return u, ok
//	}
func (c *Context) Value(key any) any {
	if c == nil || c.context == nil {
		return nil
	}

	if p := c.OtelBridge(); p != nil {
		if v := p.Value(key); v != nil {
			return v
		}
	}

	return c.context.Value(key)
}

// Contexter is an interface that checks if type has a request context.
type Contexter interface {
	// RequestContext returns request context.
	RequestContext() context.Context
}

// RequestContext returns request context from context.
//
//nolint:contextcheck
func RequestContext(ctx context.Context) *Context {
	if ctx == nil {
		return nil
	}

	if c, ok := ctx.(Contexter); ok {
		ctx = c.RequestContext()
	}

	rctx, ok := ctx.(*Context)
	if !ok {
		return nil
	}

	return rctx
}
