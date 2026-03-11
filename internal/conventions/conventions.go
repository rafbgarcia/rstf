// Package conventions defines the file conventions and rules that map the
// project's directory structure to HTTP routes.
package conventions

import (
	"fmt"
	"strings"
)

// FolderToURLPattern converts a route folder name to a Go 1.22+ ServeMux
// URL pattern. Dots become path separators, _param becomes {param}, and
// the folder name "index" maps to "/".
//
// Examples:
//
//	"index"           → "/"
//	"dashboard"       → "/dashboard"
//	"users._id"       → "/users/{id}"
//	"users._id.edit"  → "/users/{id}/edit"
func FolderToURLPattern(folderName string) string {
	if folderName == "index" {
		return "/"
	}

	segments := strings.Split(folderName, ".")
	for i, seg := range segments {
		if isDynamicSegment(seg) {
			segments[i] = "{" + seg[1:] + "}"
		}
	}
	return "/" + strings.Join(segments, "/")
}

// IsRouteDir reports whether a path is a valid route directory according to
// rstf's file-based routing rules. Route directories live directly under
// routes/ and use dotted names for nesting semantics (for example,
// routes/admin.users rather than routes/admin/users).
func IsRouteDir(path string) bool {
	if path == "routes" {
		return true
	}
	if !isWithinRoutes(path) {
		return false
	}
	name := strings.TrimPrefix(path, "routes/")
	return name != "" && !strings.Contains(name, "/")
}

// ValidateRouteDir reports a clear error when a path violates rstf's route
// directory convention.
func ValidateRouteDir(path string) error {
	if !isWithinRoutes(path) || path == "routes" {
		return nil
	}
	name := strings.TrimPrefix(path, "routes/")
	if strings.Contains(name, "/") {
		return fmt.Errorf(
			"invalid route directory %q: nested route directories are not supported; use dotted names like routes/admin.users",
			path,
		)
	}
	return nil
}

func isWithinRoutes(path string) bool {
	return path == "routes" || strings.HasPrefix(path, "routes/")
}

func isDynamicSegment(seg string) bool {
	return len(seg) > 1 && strings.HasPrefix(seg, "_")
}
