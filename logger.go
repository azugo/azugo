package azugo

import (
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

func (c *Context) initLoggerFields() {
	method := c.Method()
	path := c.Path()
	cleanedPath := path

	basePath := c.BasePath()
	if len(basePath) > 0 && len(basePath) < len(path) && basePath == path[:len(basePath)] {
		cleanedPath = path[len(basePath):]
	}

	fields := make([]zap.Field, 0, 8)

	fields = append(fields,
		// Basic request fields
		zap.String("http.request.id", c.ID()),
		zap.String("http.request.method", method),
		zap.String("url.path", cleanedPath),
		// Source
		zap.String("source.ip", c.IP().String()),
	)

	_ = c.AddLogFields(fields...)
}

// AddLogFields add fields to context logger.
func (c *Context) AddLogFields(fields ...zap.Field) error {
	if len(fields) == 0 {
		return nil
	}

	for _, field := range fields {
		c.loggerFields[field.Key] = field
	}

	return c.ReplaceLogger(c.loggerCore)
}

// ReplaceLogger replaces current context logger with custom.
func (c *Context) ReplaceLogger(logger *zap.Logger) error {
	if logger == nil {
		return nil
	}

	c.loggerCore = logger
	c.logger = logger.With(maps.Values(c.loggerFields)...)

	return nil
}

// Log returns the logger.
func (c *Context) Log() *zap.Logger {
	return c.logger
}

// SkipRequestLog sets to skip request log entry for current request.
func (c *Context) SkipRequestLog() {
	c.SetUserValue("log_request", false)
}
