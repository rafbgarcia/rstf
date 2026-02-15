package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rafbgarcia/rstf/internal/conventions"
)

// Generate runs the full codegen pipeline for a project. It removes any
// existing .rstf/ output, parses Go route files, analyzes TSX dependencies,
// and writes all generated files (.d.ts, runtime modules, server_gen.go).
//
// Returns the number of routes generated.
func Generate(projectRoot string) (int, error) {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return 0, fmt.Errorf("resolving project root: %w", err)
	}

	// 1. Clean slate — remove .rstf/ since everything in it is generated.
	rstfDir := filepath.Join(absRoot, ".rstf")
	if err := os.RemoveAll(rstfDir); err != nil {
		return 0, fmt.Errorf("removing .rstf/: %w", err)
	}

	// 2. Read module path from go.mod.
	goModContent, err := os.ReadFile(filepath.Join(absRoot, "go.mod"))
	if err != nil {
		return 0, fmt.Errorf("reading go.mod: %w", err)
	}
	modulePath := ParseModulePath(goModContent)
	if modulePath == "" {
		return 0, fmt.Errorf("no module directive found in go.mod")
	}

	// 3. Parse all Go route files.
	files, err := ParseDir(absRoot)
	if err != nil {
		return 0, fmt.Errorf("parsing project: %w", err)
	}

	// 4. Analyze TSX dependencies for each route directory.
	deps := map[string][]string{}
	for _, f := range files {
		if !conventions.IsRouteDir(f.Dir) {
			continue
		}
		entryPath := filepath.Join(f.Dir, "index.tsx")
		absEntry := filepath.Join(absRoot, entryPath)
		if _, err := os.Stat(absEntry); os.IsNotExist(err) {
			continue
		}
		d, err := AnalyzeDeps(absRoot, entryPath)
		if err != nil {
			return 0, fmt.Errorf("analyzing deps for %s: %w", f.Dir, err)
		}
		deps[f.Dir] = d
	}

	// 5. Create .rstf/ directory structure.
	for _, dir := range []string{
		filepath.Join(rstfDir, "types"),
		filepath.Join(rstfDir, "generated"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return 0, fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	// 6. Generate DTS and runtime modules for each RouteFile.
	for _, rf := range files {
		// Write .d.ts file.
		dtsName := dtsFileName(rf.Dir)
		dtsPath := filepath.Join(rstfDir, "types", dtsName)
		dts := GenerateDTS(rf)
		if err := os.WriteFile(dtsPath, []byte(dts), 0644); err != nil {
			return 0, fmt.Errorf("writing %s: %w", dtsPath, err)
		}

		// Write runtime module.
		rtMod := GenerateRuntimeModule(rf)
		if rtMod != "" {
			rtPath := filepath.Join(rstfDir, "generated", runtimeModulePath(rf.Dir))
			if err := os.MkdirAll(filepath.Dir(rtPath), 0755); err != nil {
				return 0, fmt.Errorf("creating dir for %s: %w", rtPath, err)
			}
			if err := os.WriteFile(rtPath, []byte(rtMod), 0644); err != nil {
				return 0, fmt.Errorf("writing %s: %w", rtPath, err)
			}
		}
	}

	// 7. Generate server_gen.go.
	serverCode, err := GenerateServer(modulePath, files, deps)
	if err != nil {
		return 0, fmt.Errorf("generating server: %w", err)
	}
	serverPath := filepath.Join(rstfDir, "server_gen.go")
	if err := os.WriteFile(serverPath, []byte(serverCode), 0644); err != nil {
		return 0, fmt.Errorf("writing server_gen.go: %w", err)
	}

	// Count routes.
	routeCount := 0
	for _, f := range files {
		if conventions.IsRouteDir(f.Dir) {
			routeCount++
		}
	}

	return routeCount, nil
}

// dtsFileName returns the .d.ts filename for a given directory path.
//
//	"."                       → "main.d.ts"
//	"routes/dashboard"        → "dashboard.d.ts"
//	"routes/users.$id.edit"   → "users-id-edit.d.ts"
//	"shared/ui/user-avatar"   → "shared-ui-user-avatar.d.ts"
func dtsFileName(dir string) string {
	if dir == "." {
		return "main.d.ts"
	}
	name := strings.TrimPrefix(dir, "routes/")
	name = strings.ReplaceAll(name, "$", "")
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, "/", "-")
	return name + ".d.ts"
}

// runtimeModulePath returns the runtime module path for a given directory.
//
//	"."                       → "main.ts"
//	"routes/dashboard"        → "routes/dashboard.ts"
//	"shared/ui/user-avatar"   → "shared/ui/user-avatar.ts"
func runtimeModulePath(dir string) string {
	if dir == "." {
		return "main.ts"
	}
	return dir + ".ts"
}
