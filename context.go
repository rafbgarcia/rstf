package rstf

import "net/http"

// Context is the request-scoped framework context passed to route handlers.
// It provides access to logging, wide events, and other framework utilities.
type Context struct {
	Log     *Logger
	Request *http.Request
}

// NewContext creates a new Context for the given HTTP request.
func NewContext(r *http.Request) *Context {
	return &Context{
		Log:     NewLogger(),
		Request: r,
	}
}
