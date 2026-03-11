package rstf

import (
	"database/sql"
	"net/http"
)

// QueryContext is the request-scoped context for deterministic read functions.
type QueryContext struct {
	*Context
}

// MutationContext is the request-scoped context for deterministic write functions.
type MutationContext struct {
	*Context
	invalidate func(...SubscriptionKey)
}

// ActionContext is the request-scoped context for side-effectful functions.
type ActionContext struct {
	*Context
}

// NewQueryContext creates a new QueryContext for the given request.
func NewQueryContext(r *http.Request, db *sql.DB, requestBodyLimit int64) *QueryContext {
	ctx := NewContext(r)
	ctx.DB = db
	_ = ctx.SetRequestBodyLimitBytes(requestBodyLimit)
	return &QueryContext{Context: ctx}
}

// NewMutationContext creates a new MutationContext for the given request.
func NewMutationContext(
	r *http.Request,
	db *sql.DB,
	requestBodyLimit int64,
	invalidate func(...SubscriptionKey),
) *MutationContext {
	ctx := NewContext(r)
	ctx.DB = db
	_ = ctx.SetRequestBodyLimitBytes(requestBodyLimit)
	return &MutationContext{
		Context:    ctx,
		invalidate: invalidate,
	}
}

// NewActionContext creates a new ActionContext for the given request.
func NewActionContext(r *http.Request, requestBodyLimit int64) *ActionContext {
	ctx := NewContext(r)
	_ = ctx.SetRequestBodyLimitBytes(requestBodyLimit)
	return &ActionContext{Context: ctx}
}

// Invalidate reruns all live queries subscribed to the given keys.
func (c *MutationContext) Invalidate(keys ...SubscriptionKey) {
	if c == nil || c.invalidate == nil || len(keys) == 0 {
		return
	}
	c.invalidate(keys...)
}
