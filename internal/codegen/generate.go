package codegen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/rafbgarcia/rstf/internal/conventions"
)

// GenerateResult holds the output of a codegen run.
type GenerateResult struct {
	RouteCount int
	Entries    map[string]string // routeDir -> absolute path to .entry.tsx
}

// Generate runs the full codegen pipeline for a project. It removes any
// existing .rstf/ output, parses Go route files, analyzes TSX dependencies,
// and writes all generated files (.d.ts, runtime modules, hydration entries,
// server_gen.go).
//
// The pipeline parallelizes independent work:
//   - Phase 1 (sequential): clean, read go.mod, parse Go files, create dirs
//   - Phase 2 (parallel): AnalyzeDeps per route, DTS + runtime writes per file, symlinks
//   - Phase 3 (parallel): hydration entries per route (needs deps from Phase 2)
//   - Phase 4 (sequential): server_gen.go, ensureDeps
func Generate(projectRoot string) (GenerateResult, error) {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("resolving project root: %w", err)
	}

	// --- Phase 1: sequential setup ---

	// 1. Clean slate — remove .rstf/ since everything in it is generated.
	rstfDir := filepath.Join(absRoot, ".rstf")
	if err := os.RemoveAll(rstfDir); err != nil {
		return GenerateResult{}, fmt.Errorf("removing .rstf/: %w", err)
	}

	// 2. Read module path from go.mod.
	goModContent, err := os.ReadFile(filepath.Join(absRoot, "go.mod"))
	if err != nil {
		return GenerateResult{}, fmt.Errorf("reading go.mod: %w", err)
	}
	modulePath := ParseModulePath(goModContent)
	if modulePath == "" {
		return GenerateResult{}, fmt.Errorf("no module directive found in go.mod")
	}

	// 3. Parse all Go route files.
	files, err := ParseDir(absRoot)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("parsing project: %w", err)
	}

	// Create .rstf/ directory structure before any parallel writes.
	for _, dir := range []string{
		filepath.Join(rstfDir, "types"),
		filepath.Join(rstfDir, "generated"),
		filepath.Join(rstfDir, "entries"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return GenerateResult{}, fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	// Collect all route entries that need dependency analysis.
	// Includes both Go+TSX routes and TSX-only routes.
	type depJob struct {
		dir       string
		entryPath string
	}
	var depJobs []depJob
	seenDirs := map[string]bool{}

	for _, f := range files {
		if !conventions.IsRouteDir(f.Dir) {
			continue
		}
		entryPath := filepath.Join(f.Dir, "index.tsx")
		absEntry := filepath.Join(absRoot, entryPath)
		if _, err := os.Stat(absEntry); os.IsNotExist(err) {
			continue
		}
		depJobs = append(depJobs, depJob{f.Dir, entryPath})
		seenDirs[f.Dir] = true
	}

	tsxRouteDirs, err := discoverTSXRouteDirs(absRoot)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("discovering TSX routes: %w", err)
	}
	for _, routeDir := range tsxRouteDirs {
		if seenDirs[routeDir] {
			continue
		}
		depJobs = append(depJobs, depJob{routeDir, filepath.Join(routeDir, "index.tsx")})
	}

	// --- Phase 2: parallel AnalyzeDeps + DTS/runtime writes + symlinks ---

	var mu sync.Mutex
	deps := map[string][]string{}
	cache := newFSCache()

	sem := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup
	var firstErr error

	setErr := func(err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
	}

	// 4. Parallel AnalyzeDeps for each route.
	for _, job := range depJobs {
		wg.Add(1)
		go func(dir, entryPath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			d, err := AnalyzeDeps(absRoot, entryPath, cache)
			if err != nil {
				setErr(fmt.Errorf("analyzing deps for %s: %w", dir, err))
				return
			}
			mu.Lock()
			deps[dir] = d
			mu.Unlock()
		}(job.dir, job.entryPath)
	}

	// 7. Parallel DTS and runtime module writes for each RouteFile.
	for _, rf := range files {
		wg.Add(1)
		go func(rf RouteFile) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Write .d.ts file.
			dtsPath := filepath.Join(rstfDir, "types", dtsFileName(rf.Dir))
			dts := GenerateDTS(rf)
			if err := os.WriteFile(dtsPath, []byte(dts), 0644); err != nil {
				setErr(fmt.Errorf("writing %s: %w", dtsPath, err))
				return
			}

			// Write runtime module.
			rtMod := GenerateRuntimeModule(rf, componentPathForDir(rf.Dir))
			if rtMod != "" {
				rtPath := filepath.Join(rstfDir, "generated", runtimeModulePath(rf.Dir))
				if err := os.MkdirAll(filepath.Dir(rtPath), 0755); err != nil {
					setErr(fmt.Errorf("creating dir for %s: %w", rtPath, err))
					return
				}
				if err := os.WriteFile(rtPath, []byte(rtMod), 0644); err != nil {
					setErr(fmt.Errorf("writing %s: %w", rtPath, err))
					return
				}
			}
		}(rf)
	}

	// 6. Create symlinks for directories with $ (dynamic segments).
	// Sequential — few entries, and must complete before server_gen.go.
	for _, f := range files {
		if !strings.Contains(f.Dir, "$") || f.Dir == "." {
			continue
		}
		sanitized := strings.ReplaceAll(f.Dir, "$", "")
		linkPath := filepath.Join(rstfDir, "pkgs", sanitized)
		if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
			return GenerateResult{}, fmt.Errorf("creating symlink parent for %s: %w", f.Dir, err)
		}
		if err := os.Symlink(filepath.Join(absRoot, f.Dir), linkPath); err != nil {
			return GenerateResult{}, fmt.Errorf("creating symlink for %s: %w", f.Dir, err)
		}
	}

	wg.Wait()
	if firstErr != nil {
		return GenerateResult{}, firstErr
	}

	// --- Phase 3: parallel hydration entries (needs deps from Phase 2) ---

	entries := map[string]string{}
	for routeDir, routeDeps := range deps {
		if !conventions.IsRouteDir(routeDir) {
			continue
		}
		wg.Add(1)
		go func(routeDir string, routeDeps []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			entryContent := GenerateHydrationEntry(routeDir, routeDeps)
			entryPath := filepath.Join(rstfDir, "entries", entryFileName(routeDir))
			if err := os.WriteFile(entryPath, []byte(entryContent), 0644); err != nil {
				setErr(fmt.Errorf("writing entry %s: %w", entryPath, err))
				return
			}
			mu.Lock()
			entries[routeDir] = entryPath
			mu.Unlock()
		}(routeDir, routeDeps)
	}

	wg.Wait()
	if firstErr != nil {
		return GenerateResult{}, firstErr
	}

	// --- Phase 4: sequential finalization ---

	// 9. Generate server_gen.go.
	serverCode, err := GenerateServer(modulePath, files, deps)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("generating server: %w", err)
	}
	serverPath := filepath.Join(rstfDir, "server_gen.go")
	if err := os.WriteFile(serverPath, []byte(serverCode), 0644); err != nil {
		return GenerateResult{}, fmt.Errorf("writing server_gen.go: %w", err)
	}

	// 10. Ensure framework dependencies are in go.sum. The generated
	// server_gen.go lives in .rstf/ which Go tools skip (dot-prefixed dir),
	// so go mod tidy won't discover its imports. We use go get to explicitly
	// resolve the framework packages and their transitive deps (e.g. chi).
	// Skipped when developing the framework itself or when already resolved.
	if err := ensureDeps(absRoot, modulePath); err != nil {
		return GenerateResult{}, err
	}

	// Count routes.
	routeCount := 0
	routeSet := map[string]bool{}
	for _, f := range files {
		if conventions.IsRouteDir(f.Dir) {
			routeSet[f.Dir] = true
			routeCount++
		}
	}
	for routeDir := range deps {
		if !routeSet[routeDir] && conventions.IsRouteDir(routeDir) {
			routeCount++
		}
	}

	return GenerateResult{
		RouteCount: routeCount,
		Entries:    entries,
	}, nil
}

// discoverTSXRouteDirs finds route directories that have index.tsx but might
// not have been discovered by ParseDir (because they lack .go files).
func discoverTSXRouteDirs(absRoot string) ([]string, error) {
	routesDir := filepath.Join(absRoot, "routes")
	if _, err := os.Stat(routesDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(routesDir)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		tsxPath := filepath.Join(routesDir, e.Name(), "index.tsx")
		if _, err := os.Stat(tsxPath); err == nil {
			dirs = append(dirs, filepath.ToSlash(filepath.Join("routes", e.Name())))
		}
	}
	return dirs, nil
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

// componentPathForDir returns the key used in __RSTF_SERVER_DATA__.
//
//	"."                       → "main"
//	"routes/dashboard"        → "routes/dashboard"
//	"shared/ui/user-avatar"   → "shared/ui/user-avatar"
func componentPathForDir(dir string) string {
	if dir == "." {
		return "main"
	}
	return dir
}

// ensureDeps runs `go get` for framework packages that the generated
// server_gen.go imports. Since server_gen.go lives in .rstf/ (a dot-prefixed
// directory invisible to go mod tidy), its transitive dependencies (e.g. chi
// via rstf/router) won't appear in go.sum otherwise.
//
// Skipped when the current module IS the framework (sub-packages are local)
// or when the framework module is already recorded in go.sum.
func ensureDeps(projectRoot, modulePath string) error {
	// Developing the framework itself — sub-packages are local.
	if modulePath == frameworkModule {
		return nil
	}

	// Framework module already in go.sum — deps are resolved.
	if sumContent, err := os.ReadFile(filepath.Join(projectRoot, "go.sum")); err == nil {
		if strings.Contains(string(sumContent), frameworkModule+" ") {
			return nil
		}
	}

	cmd := exec.Command("go", "get",
		frameworkModule+"/renderer",
		frameworkModule+"/router",
	)
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("resolving framework deps: %s\n%s", err, out)
	}
	return nil
}
