// Package conventions defines the file conventions and rules that map the
// project's directory structure to HTTP routes.
package conventions

import (
	"strings"
)

// FolderToURLPattern converts a route folder name to a Go 1.22+ ServeMux
// URL pattern. Dots become path separators, $param becomes {param}, and
// the folder name "index" maps to "/".
//
// Examples:
//
//	"index"           → "/"
//	"dashboard"       → "/dashboard"
//	"users.$id"       → "/users/{id}"
//	"users.$id.edit"  → "/users/{id}/edit"
func FolderToURLPattern(folderName string) string {
	if folderName == "index" {
		return "/"
	}

	segments := strings.Split(folderName, ".")
	for i, seg := range segments {
		if strings.HasPrefix(seg, "$") {
			segments[i] = "{" + seg[1:] + "}"
		}
	}
	return "/" + strings.Join(segments, "/")
}

// IsRouteDir reports whether a path (relative to the project root) is
// inside the routes/ directory.
func IsRouteDir(path string) bool {
	return path == "routes" || strings.HasPrefix(path, "routes/")
}
