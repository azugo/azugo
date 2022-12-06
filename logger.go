package azugo

import (
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

func (ctx *Context) initLoggerFields() {
	method := ctx.Method()
	path := ctx.Path()
	cleanedPath := path
	basePath := ctx.BasePath()
	if len(basePath) > 0 && len(basePath) < len(path) && basePath == path[:len(basePath)] {
		cleanedPath = path[len(basePath):]
	}

	fields := make([]zap.Field, 0, 8)

	fields = append(fields,
		// Basic request fields
		zap.String("http.request.id", ctx.ID()),
		zap.String("http.request.method", method),
		zap.String("url.path", cleanedPath),
		// Source
		zap.String("source.ip", ctx.IP().String()),
	)

	_ = ctx.AddLogFields(fields...)
}

// AddLogFields add fields to context logger.
func (ctx *Context) AddLogFields(fields ...zap.Field) error {
	for _, field := range fields {
		ctx.loggerFields[field.Key] = field
	}
	return ctx.ReplaceLogger(ctx.loggerCore)
}

// ReplaceLogger replaces current context logger with custom.
func (ctx *Context) ReplaceLogger(logger *zap.Logger) error {
	if logger == nil {
		return nil
	}
	ctx.loggerCore = logger
	ctx.logger = logger.With(maps.Values(ctx.loggerFields)...)
	return nil
}

// Log returns the logger.
func (ctx *Context) Log() *zap.Logger {
	return ctx.logger
}

// SkipRequestLog sets to skip request log entry for current request.
func (ctx *Context) SkipRequestLog() {
	ctx.SetUserValue("log_request", false)
}
