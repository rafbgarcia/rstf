package rstf

import (
	"database/sql"
	"net/http"
)

// Context is the request-scoped framework context passed to route handlers.
// It provides access to logging, the database connection pool, and other framework utilities.
type Context struct {
	Log                   *Logger
	Writer                http.ResponseWriter
	Request               *http.Request
	DB                    *sql.DB
	requestBodyLimitBytes int64
}

// NewContext creates a new Context for the given HTTP request.
func NewContext(r *http.Request) *Context {
	return &Context{
		Log:                   NewLogger(),
		Request:               r,
		requestBodyLimitBytes: DefaultBodyLimit,
	}
}

// Param returns a path parameter value from the current request.
func (c *Context) Param(name string) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return c.Request.PathValue(name)
}

// SetRequestBodyLimitBytes sets the maximum request body size accepted by BindJSON.
func (c *Context) SetRequestBodyLimitBytes(limit int64) error {
	if limit <= 0 {
		return &RequestError{
			Code:    ErrorCodeInternal,
			Message: "request body limit must be greater than zero bytes",
			Status:  http.StatusInternalServerError,
		}
	}
	c.requestBodyLimitBytes = limit
	return nil
}

// RequestBodyLimitBytes returns the configured request body limit.
func (c *Context) RequestBodyLimitBytes() int64 {
	if c == nil || c.requestBodyLimitBytes <= 0 {
		return DefaultBodyLimit
	}
	return c.requestBodyLimitBytes
}
