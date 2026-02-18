package rstf

import "net/http"

// Middleware is a standard Go HTTP middleware.
// It is a type alias so any func(http.Handler) http.Handler is compatible
// without casting.
type Middleware = func(http.Handler) http.Handler
