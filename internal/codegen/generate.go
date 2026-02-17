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

// ChangeEvent describes a single file change for incremental codegen.
type ChangeEvent struct {
	Path string // absolute path
	Kind string // "go" or "tsx"
}

// RegenerateResult extends GenerateResult with information about what changed.
type RegenerateResult struct {
	GenerateResult
	ServerChanged bool
}

// Generator holds persisted state between codegen runs, enabling incremental
// rebuilds via Regenerate.
type Generator struct {
	root       string // absolute project root
	rstfDir    string
	modulePath string

	files      []RouteFile
	filesByDir map[string]RouteFile
	deps       map[string][]string
	cache      *fsCache
	entries    map[string]string // routeDir -> absolute entry file path

	prevServerCode string
}

// NewGenerator creates a Generator for the given project root. It reads go.mod
// to resolve the module path but does not run codegen.
func NewGenerator(projectRoot string) (*Generator, error) {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolving project root: %w", err)
	}

	goModContent, err := os.ReadFile(filepath.Join(absRoot, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("reading go.mod: %w", err)
	}
	modulePath := ParseModulePath(goModContent)
	if modulePath == "" {
		return nil, fmt.Errorf("no module directive found in go.mod")
	}

	return &Generator{
		root:       absRoot,
		rstfDir:    filepath.Join(absRoot, ".rstf"),
		modulePath: modulePath,
		filesByDir: make(map[string]RouteFile),
		deps:       make(map[string][]string),
		entries:    make(map[string]string),
		cache:      newFSCache(),
	}, nil
}

// Generate runs the full codegen pipeline — clean slate rebuild. It populates
// the Generator's internal state so subsequent Regenerate calls can be
// incremental.
func (g *Generator) Generate() (GenerateResult, error) {
	// --- Phase 1: sequential setup ---

	// 1. Clean slate — remove .rstf/ since everything in it is generated.
	if err := os.RemoveAll(g.rstfDir); err != nil {
		return GenerateResult{}, fmt.Errorf("removing .rstf/: %w", err)
	}

	// 2. Parse all Go route files.
	files, err := ParseDir(g.root)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("parsing project: %w", err)
	}

	// Create .rstf/ directory structure before any parallel writes.
	for _, dir := range []string{
		filepath.Join(g.rstfDir, "types"),
		filepath.Join(g.rstfDir, "generated"),
		filepath.Join(g.rstfDir, "entries"),
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
		absEntry := filepath.Join(g.root, entryPath)
		if _, err := os.Stat(absEntry); os.IsNotExist(err) {
			continue
		}
		depJobs = append(depJobs, depJob{f.Dir, entryPath})
		seenDirs[f.Dir] = true
	}

	tsxRouteDirs, err := discoverTSXRouteDirs(g.root)
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
	g.cache = newFSCache()

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

	// Parallel AnalyzeDeps for each route.
	for _, job := range depJobs {
		wg.Add(1)
		go func(dir, entryPath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			d, err := AnalyzeDeps(g.root, entryPath, g.cache)
			if err != nil {
				setErr(fmt.Errorf("analyzing deps for %s: %w", dir, err))
				return
			}
			mu.Lock()
			deps[dir] = d
			mu.Unlock()
		}(job.dir, job.entryPath)
	}

	// Parallel DTS and runtime module writes for each RouteFile.
	for _, rf := range files {
		wg.Add(1)
		go func(rf RouteFile) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := writeDTSAndRuntime(g.rstfDir, rf); err != nil {
				setErr(err)
			}
		}(rf)
	}

	// Create symlinks for directories with $ (dynamic segments).
	for _, f := range files {
		if !strings.Contains(f.Dir, "$") || f.Dir == "." {
			continue
		}
		sanitized := strings.ReplaceAll(f.Dir, "$", "")
		linkPath := filepath.Join(g.rstfDir, "pkgs", sanitized)
		if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
			return GenerateResult{}, fmt.Errorf("creating symlink parent for %s: %w", f.Dir, err)
		}
		if err := os.Symlink(filepath.Join(g.root, f.Dir), linkPath); err != nil {
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
			entryPath := filepath.Join(g.rstfDir, "entries", entryFileName(routeDir))
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

	serverCode, err := GenerateServer(g.modulePath, files, deps)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("generating server: %w", err)
	}
	serverPath := filepath.Join(g.rstfDir, "server_gen.go")
	if err := os.WriteFile(serverPath, []byte(serverCode), 0644); err != nil {
		return GenerateResult{}, fmt.Errorf("writing server_gen.go: %w", err)
	}

	if err := ensureDeps(g.root, g.modulePath); err != nil {
		return GenerateResult{}, err
	}

	// Persist state for incremental rebuilds.
	g.files = files
	g.filesByDir = make(map[string]RouteFile, len(files))
	for _, f := range files {
		g.filesByDir[f.Dir] = f
	}
	g.deps = deps
	g.entries = entries
	g.prevServerCode = serverCode

	return GenerateResult{
		RouteCount: countRoutes(files, deps),
		Entries:    entries,
	}, nil
}

// Regenerate performs an incremental codegen based on file change events. It
// re-parses only changed Go directories, re-analyzes deps (with a warm cache),
// and only writes files that actually changed. Returns which outputs changed so
// the caller can decide whether to restart the server.
func (g *Generator) Regenerate(events []ChangeEvent) (RegenerateResult, error) {
	// 1. Classify events.
	goChangedDirs := map[string]bool{} // relative dir -> true
	var changedPaths []string

	for _, ev := range events {
		changedPaths = append(changedPaths, ev.Path)
		if ev.Kind == "go" {
			relDir, err := filepath.Rel(g.root, filepath.Dir(ev.Path))
			if err != nil {
				continue
			}
			goChangedDirs[filepath.ToSlash(relDir)] = true
		}
	}

	// 2. Invalidate cache entries for changed paths.
	g.cache.invalidatePaths(changedPaths)

	// 3. For each Go-changed dir: re-parse and update filesByDir, write DTS + runtime.
	for relDir := range goChangedDirs {
		absDir := filepath.Join(g.root, relDir)
		rf, err := ParseSingleDir(g.root, absDir)
		if err != nil {
			return RegenerateResult{}, fmt.Errorf("parsing %s: %w", relDir, err)
		}

		if rf != nil {
			g.filesByDir[rf.Dir] = *rf
			if err := writeDTSAndRuntime(g.rstfDir, *rf); err != nil {
				return RegenerateResult{}, err
			}
		} else {
			// Directory no longer has route functions — remove it.
			delete(g.filesByDir, relDir)
		}
	}

	// 4. Rebuild files slice from filesByDir.
	g.files = make([]RouteFile, 0, len(g.filesByDir))
	for _, rf := range g.filesByDir {
		g.files = append(g.files, rf)
	}

	// 5. Handle $ symlinks for changed dirs.
	for relDir := range goChangedDirs {
		if !strings.Contains(relDir, "$") || relDir == "." {
			continue
		}
		sanitized := strings.ReplaceAll(relDir, "$", "")
		linkPath := filepath.Join(g.rstfDir, "pkgs", sanitized)
		// Remove stale symlink, re-create.
		os.Remove(linkPath)
		if _, exists := g.filesByDir[relDir]; exists {
			if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
				return RegenerateResult{}, fmt.Errorf("creating symlink parent for %s: %w", relDir, err)
			}
			if err := os.Symlink(filepath.Join(g.root, relDir), linkPath); err != nil {
				return RegenerateResult{}, fmt.Errorf("creating symlink for %s: %w", relDir, err)
			}
		}
	}

	// 6. Re-discover TSX-only routes.
	tsxRouteDirs, err := discoverTSXRouteDirs(g.root)
	if err != nil {
		return RegenerateResult{}, fmt.Errorf("discovering TSX routes: %w", err)
	}

	// 7. Re-run AnalyzeDeps for all routes (parallel, warm cache).
	type depJob struct {
		dir       string
		entryPath string
	}
	var depJobs []depJob
	seenDirs := map[string]bool{}

	for _, f := range g.files {
		if !conventions.IsRouteDir(f.Dir) {
			continue
		}
		entryPath := filepath.Join(f.Dir, "index.tsx")
		absEntry := filepath.Join(g.root, entryPath)
		if _, err := os.Stat(absEntry); os.IsNotExist(err) {
			continue
		}
		depJobs = append(depJobs, depJob{f.Dir, entryPath})
		seenDirs[f.Dir] = true
	}
	for _, routeDir := range tsxRouteDirs {
		if seenDirs[routeDir] {
			continue
		}
		depJobs = append(depJobs, depJob{routeDir, filepath.Join(routeDir, "index.tsx")})
	}

	var mu sync.Mutex
	newDeps := map[string][]string{}
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

	for _, job := range depJobs {
		wg.Add(1)
		go func(dir, entryPath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			d, err := AnalyzeDeps(g.root, entryPath, g.cache)
			if err != nil {
				setErr(fmt.Errorf("analyzing deps for %s: %w", dir, err))
				return
			}
			mu.Lock()
			newDeps[dir] = d
			mu.Unlock()
		}(job.dir, job.entryPath)
	}
	wg.Wait()
	if firstErr != nil {
		return RegenerateResult{}, firstErr
	}

	// 8. Diff old vs new deps → only write hydration entries that changed.
	newEntries := make(map[string]string, len(g.entries))
	for routeDir, routeDeps := range newDeps {
		if !conventions.IsRouteDir(routeDir) {
			continue
		}
		oldDeps := g.deps[routeDir]
		if !depsEqual(oldDeps, routeDeps) || g.entries[routeDir] == "" {
			entryContent := GenerateHydrationEntry(routeDir, routeDeps)
			entryPath := filepath.Join(g.rstfDir, "entries", entryFileName(routeDir))
			if err := os.WriteFile(entryPath, []byte(entryContent), 0644); err != nil {
				return RegenerateResult{}, fmt.Errorf("writing entry %s: %w", entryPath, err)
			}
			newEntries[routeDir] = entryPath
		} else {
			newEntries[routeDir] = g.entries[routeDir]
		}
	}

	// 9. Generate server_gen.go, compare with previous.
	serverCode, err := GenerateServer(g.modulePath, g.files, newDeps)
	if err != nil {
		return RegenerateResult{}, fmt.Errorf("generating server: %w", err)
	}
	serverChanged := serverCode != g.prevServerCode
	if serverChanged {
		serverPath := filepath.Join(g.rstfDir, "server_gen.go")
		if err := os.WriteFile(serverPath, []byte(serverCode), 0644); err != nil {
			return RegenerateResult{}, fmt.Errorf("writing server_gen.go: %w", err)
		}
	}

	// 10. Update cached state.
	g.deps = newDeps
	g.entries = newEntries
	g.prevServerCode = serverCode

	return RegenerateResult{
		GenerateResult: GenerateResult{
			RouteCount: countRoutes(g.files, newDeps),
			Entries:    newEntries,
		},
		ServerChanged: serverChanged,
	}, nil
}

// Generate is a standalone wrapper that creates a throwaway Generator and runs
// the full pipeline. Existing tests and one-shot callers can use this without
// change.
func Generate(projectRoot string) (GenerateResult, error) {
	gen, err := NewGenerator(projectRoot)
	if err != nil {
		return GenerateResult{}, err
	}
	return gen.Generate()
}

// --- helpers ---

// writeDTSAndRuntime writes the .d.ts and runtime module for a single RouteFile.
func writeDTSAndRuntime(rstfDir string, rf RouteFile) error {
	// Write .d.ts file.
	dtsPath := filepath.Join(rstfDir, "types", dtsFileName(rf.Dir))
	dts := GenerateDTS(rf)
	if err := os.WriteFile(dtsPath, []byte(dts), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", dtsPath, err)
	}

	// Write runtime module.
	rtMod := GenerateRuntimeModule(rf, componentPathForDir(rf.Dir))
	if rtMod != "" {
		rtPath := filepath.Join(rstfDir, "generated", runtimeModulePath(rf.Dir))
		if err := os.MkdirAll(filepath.Dir(rtPath), 0755); err != nil {
			return fmt.Errorf("creating dir for %s: %w", rtPath, err)
		}
		if err := os.WriteFile(rtPath, []byte(rtMod), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", rtPath, err)
		}
	}
	return nil
}

// countRoutes counts unique route directories across parsed files and deps.
func countRoutes(files []RouteFile, deps map[string][]string) int {
	routeSet := map[string]bool{}
	for _, f := range files {
		if conventions.IsRouteDir(f.Dir) {
			routeSet[f.Dir] = true
		}
	}
	for routeDir := range deps {
		if conventions.IsRouteDir(routeDir) {
			routeSet[routeDir] = true
		}
	}
	return len(routeSet)
}

// depsEqual reports whether two sorted dep slices are identical.
func depsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
