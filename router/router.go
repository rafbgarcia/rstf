// Package router provides the HTTP router used by rstf's generated server.
// It wraps chi and bridges chi URL params to Go's Request.PathValue() so
// user handlers can call ctx.Request.PathValue("id") transparently.
package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Router is the HTTP router for rstf applications.
type Router struct {
	mux chi.Router
}

// New creates a Router with the PathValue bridge middleware applied.
func New() *Router {
	mux := chi.NewRouter()

	// Bridge chi URL params to Go's Request.PathValue() so user handlers
	// can call ctx.Request.PathValue("id") regardless of the router.
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			rctx := chi.RouteContext(req.Context())
			for i, key := range rctx.URLParams.Keys {
				req.SetPathValue(key, rctx.URLParams.Values[i])
			}
			next.ServeHTTP(w, req)
		})
	})

	return &Router{mux: mux}
}

// Get registers a handler for GET requests at the given pattern.
func (r *Router) Get(pattern string, handler http.HandlerFunc) {
	r.mux.Get(pattern, handler)
}

// Handle registers an http.Handler at the given pattern.
func (r *Router) Handle(pattern string, handler http.Handler) {
	r.mux.Handle(pattern, handler)
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}
