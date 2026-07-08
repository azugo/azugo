package azugo

import (
	"go.uber.org/zap"
)

// baseLoggerFields appends default request fields to the given slice.
func (c *Context) baseLoggerFields(fields []zap.Field) []zap.Field {
	path := c.Path()
	cleanedPath := path

	basePath := c.BasePath()
	if len(basePath) > 0 && len(basePath) < len(path) && basePath == path[:len(basePath)] {
		cleanedPath = path[len(basePath):]
	}

	return append(fields,
		// Basic request fields
		zap.String("http.request.id", c.ID()),
		zap.String("http.request.method", c.Method().String()),
		zap.String("url.path", cleanedPath),
		// Source
		zap.String("source.ip", c.IP().String()),
	)
}

// LogFields returns the fields added to the context logger.
func (c *Context) LogFields() []zap.Field {
	return c.loggerFields
}

// AddLogFields add fields to context logger.
func (c *Context) AddLogFields(fields ...zap.Field) error {
	if len(fields) == 0 {
		return nil
	}

	for _, field := range fields {
		found := false

		for i := range c.loggerFields {
			if c.loggerFields[i].Key == field.Key {
				c.loggerFields[i] = field
				found = true

				break
			}
		}

		if !found {
			c.loggerFields = append(c.loggerFields, field)
		}
	}

	// Invalidate the materialized logger so that it is rebuilt on next use.
	c.logger = nil

	return nil
}

// ReplaceLogger replaces current context logger with custom.
func (c *Context) ReplaceLogger(logger *zap.Logger) error {
	if logger == nil {
		return nil
	}

	c.loggerCore = logger
	c.logger = nil

	return nil
}

// Log returns the request logger, building it lazily on first use.
func (c *Context) Log() *zap.Logger {
	if c.logger != nil {
		return c.logger
	}

	if c.loggerCore == nil {
		c.loggerCore = c.app.Log()
	}

	fields := make([]zap.Field, 0, len(c.loggerFields)+4)
	if c.context != nil {
		fields = c.baseLoggerFields(fields)
	}

	for _, field := range c.loggerFields {
		found := false

		for i := range fields {
			if fields[i].Key == field.Key {
				fields[i] = field
				found = true

				break
			}
		}

		if !found {
			fields = append(fields, field)
		}
	}

	c.logger = c.loggerCore.With(fields...)

	return c.logger
}

// SkipRequestLog sets to skip request log entry and tracing for current request.
func (c *Context) SkipRequestLog() {
	c.SetUserValue("__log_request", false)
}

// IsSkipRequestLog reports whether request logging and tracing are disabled for
// the current request.
func (c *Context) IsSkipRequestLog() bool {
	val, ok := c.UserValue("__log_request").(bool)

	return !ok || !val
}

// SkipMetrics sets to skip metrics recording for current request.
func (c *Context) SkipMetrics() {
	c.SetUserValue("__skip_metrics", true)
}
