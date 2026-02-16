package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rafbgarcia/rstf/internal/conventions"
)

// frameworkModule is the import path of the rstf framework itself.
const frameworkModule = "github.com/rafbgarcia/rstf"

// serverImport tracks a user-package import for the generated server file.
type serverImport struct {
	Alias      string // Go import alias (e.g. "app", "dashboard")
	ImportPath string // full import path
	Dir        string // project-relative dir (e.g. ".", "routes/dashboard")
	HasContext bool   // whether SSR() takes *rstf.Context
}

// routeEntry pairs a route directory with its computed URL pattern.
type routeEntry struct {
	dir        string // e.g. "routes/dashboard"
	folderName string // e.g. "dashboard"
	urlPattern string // e.g. "/dashboard"
}

// GenerateServer produces the content of .rstf/server_gen.go — the Go entry
// point that wires routes to handlers, calls SSR functions, and renders via
// the Bun sidecar.
//
// Parameters:
//   - modulePath: the user's Go module path (from go.mod), e.g. "github.com/user/myapp"
//   - files: all RouteFile results from ParseDir(projectRoot), with Dir relative
//     to the project root (".", "routes/dashboard", "shared/ui/user-avatar")
//   - deps: maps route dir → dep dirs from AnalyzeDeps. The layout dir "." is
//     NOT expected in deps — GenerateServer always adds it.
func GenerateServer(modulePath string, files []RouteFile, deps map[string][]string) (string, error) {
	// Build dir → RouteFile lookup.
	fileMap := map[string]RouteFile{}
	for _, f := range files {
		fileMap[f.Dir] = f
	}

	// Find the layout (root package).
	layout, hasLayout := fileMap["."]

	// Validate: main.go must not use "package main" — it needs to be importable
	// by the generated server_gen.go which itself declares package main.
	if hasLayout && layout.Package == "main" {
		return "", fmt.Errorf(
			"main.go: package main is reserved for rstf, please use a different package name (e.g. your app name)",
		)
	}

	// Identify route dirs and compute URL patterns.
	var routes []routeEntry
	for _, f := range files {
		if !conventions.IsRouteDir(f.Dir) {
			continue
		}
		folder := strings.TrimPrefix(f.Dir, "routes/")
		routes = append(routes, routeEntry{
			dir:        f.Dir,
			folderName: folder,
			urlPattern: conventions.FolderToURLPattern(folder),
		})
	}

	// Also add routes that appear in deps but don't have a .go file (TSX-only routes).
	routeSet := map[string]bool{}
	for _, r := range routes {
		routeSet[r.dir] = true
	}
	for routeDir := range deps {
		if !routeSet[routeDir] && conventions.IsRouteDir(routeDir) {
			folder := strings.TrimPrefix(routeDir, "routes/")
			routes = append(routes, routeEntry{
				dir:        routeDir,
				folderName: folder,
				urlPattern: conventions.FolderToURLPattern(folder),
			})
		}
	}

	// Sort routes by URL pattern for deterministic output.
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].urlPattern < routes[j].urlPattern
	})

	// Collect all user-package imports needed across all routes.
	imports := collectImports(modulePath, layout, hasLayout, routes, deps, fileMap)

	// Build alias lookup: dir → serverImport.
	aliasMap := map[string]serverImport{}
	for _, imp := range imports {
		aliasMap[imp.Dir] = imp
	}

	// Generate the Go source.
	var b strings.Builder

	writeHeader(&b)
	writeImports(&b, imports)
	writeStructToMap(&b)
	writeAssemblePage(&b)
	writeMain(&b, routes, layout, hasLayout, aliasMap, deps)

	return b.String(), nil
}

// collectImports gathers all unique user-package imports across the layout and
// all routes, assigning collision-free aliases.
func collectImports(
	modulePath string,
	layout RouteFile,
	hasLayout bool,
	routes []routeEntry,
	deps map[string][]string,
	fileMap map[string]RouteFile,
) []serverImport {
	seen := map[string]bool{}       // dir → already added
	usedAliases := map[string]int{} // alias → count (for collision detection)
	var imports []serverImport

	addImport := func(dir string) {
		if seen[dir] {
			return
		}
		seen[dir] = true

		rf, ok := fileMap[dir]
		if !ok {
			return // no .go file for this dir
		}

		var importPath string
		var baseAlias string
		if dir == "." {
			importPath = modulePath
			baseAlias = "app"
		} else if strings.Contains(dir, "$") {
			// $ is not valid in Go import paths. The codegen pipeline creates
			// symlinks under .rstf/pkgs/ with $ stripped so Go can import them.
			importPath = modulePath + "/.rstf/pkgs/" + strings.ReplaceAll(dir, "$", "")
			baseAlias = rf.Package
		} else {
			importPath = modulePath + "/" + dir
			baseAlias = rf.Package
		}

		// Resolve alias collisions.
		alias := baseAlias
		if count, exists := usedAliases[baseAlias]; exists {
			alias = fmt.Sprintf("%s%d", baseAlias, count+1)
		}
		usedAliases[baseAlias]++

		hasCtx := false
		if len(rf.Funcs) > 0 {
			hasCtx = rf.Funcs[0].HasContext
		}

		imports = append(imports, serverImport{
			Alias:      alias,
			ImportPath: importPath,
			Dir:        dir,
			HasContext: hasCtx,
		})
	}

	// Layout first.
	if hasLayout {
		addImport(".")
	}

	// Route packages and their deps.
	for _, r := range routes {
		addImport(r.dir)
		for _, depDir := range deps[r.dir] {
			addImport(depDir)
		}
	}

	return imports
}

func writeHeader(b *strings.Builder) {
	b.WriteString("// Code generated by rstf. DO NOT EDIT.\n")
	b.WriteString("package main\n\n")
}

func writeImports(b *strings.Builder, imports []serverImport) {
	b.WriteString("import (\n")
	// Standard library.
	b.WriteString("\t\"encoding/json\"\n")
	b.WriteString("\t\"flag\"\n")
	b.WriteString("\t\"fmt\"\n")
	b.WriteString("\t\"net/http\"\n")
	b.WriteString("\t\"os\"\n")
	b.WriteString("\t\"os/signal\"\n")
	b.WriteString("\t\"strings\"\n")
	b.WriteString("\t\"syscall\"\n")
	b.WriteString("\n")
	// Framework.
	fmt.Fprintf(b, "\trstf %q\n", frameworkModule)
	fmt.Fprintf(b, "\t%q\n", frameworkModule+"/renderer")
	b.WriteString("\n")
	// User packages.
	for _, imp := range imports {
		fmt.Fprintf(b, "\t%s %q\n", imp.Alias, imp.ImportPath)
	}
	b.WriteString(")\n\n")
}

func writeStructToMap(b *strings.Builder) {
	b.WriteString(`func structToMap(v any) map[string]any {
	b, _ := json.Marshal(v)
	var m map[string]any
	json.Unmarshal(b, &m)
	return m
}`)
	b.WriteString("\n\n")
}

func writeAssemblePage(b *strings.Builder) {
	b.WriteString(`func assemblePage(html string, serverData map[string]map[string]any, bundlePath string) string {
	sdJSON, _ := json.Marshal(serverData)
	dataScript := "<script>window.__RSTF_SERVER_DATA__ = " + string(sdJSON) + "</script>"
	bundleScript := "<script src=\"" + bundlePath + "\"></script>"
	page := "<!DOCTYPE html>" + strings.Replace(html, "</body>", dataScript+bundleScript+"</body>", 1)
	return page
}`)
	b.WriteString("\n\n")
}

func writeMain(
	b *strings.Builder,
	routes []routeEntry,
	layout RouteFile,
	hasLayout bool,
	aliasMap map[string]serverImport,
	deps map[string][]string,
) {
	b.WriteString(`func main() {
	port := flag.String("port", "3000", "HTTP server port")
	flag.Parse()

	r := renderer.New()
	if err := r.Start("."); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start renderer: %s\n", err)
		os.Exit(1)
	}
	defer r.Stop()

	// Stop the sidecar on interrupt/terminate signals.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		r.Stop()
		os.Exit(0)
	}()

	http.Handle("GET /.rstf/static/", http.StripPrefix("/.rstf/static/", http.FileServer(http.Dir(".rstf/static"))))
`)

	for _, route := range routes {
		// Build ServerData entries.
		type sdEntry struct {
			key  string // e.g. "main", "routes/dashboard"
			call string // e.g. "structToMap(app.SSR(ctx))"
		}
		var entries []sdEntry

		// Layout always first.
		if hasLayout && len(layout.Funcs) > 0 {
			imp := aliasMap["."]
			entries = append(entries, sdEntry{
				key:  "main",
				call: ssrCall(imp.Alias, imp.HasContext),
			})
		}

		// Route's own deps (which includes itself if it has a .go file).
		for _, depDir := range deps[route.dir] {
			if depDir == "." {
				continue // layout already handled
			}
			imp, ok := aliasMap[depDir]
			if !ok {
				continue
			}
			entries = append(entries, sdEntry{
				key:  depDir,
				call: ssrCall(imp.Alias, imp.HasContext),
			})
		}

		fmt.Fprintf(b, `
	http.HandleFunc("GET %s", func(w http.ResponseWriter, req *http.Request) {
		ctx := rstf.NewContext(req)

		sd := map[string]map[string]any{
`, route.urlPattern)

		for _, e := range entries {
			fmt.Fprintf(b, "\t\t\t%q: %s,\n", e.key, e.call)
		}

		fmt.Fprintf(b, "\t\t}\n\n")
		fmt.Fprintf(b, "\t\thtml, err := r.Render(renderer.RenderRequest{\n")
		fmt.Fprintf(b, "\t\t\tComponent: %q,\n", route.dir)
		fmt.Fprintf(b, "\t\t\tLayout:    \"main\",\n")

		if len(entries) > 0 {
			b.WriteString("\t\t\tServerData: sd,\n")
		}

		fmt.Fprintf(b, `		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Fprint(w, assemblePage(html, sd, %q))
	})
`, bundlePath(route.dir))
	}

	b.WriteString(`
	http.ListenAndServe(":"+*port, nil)
}
`)
}

// ssrCall returns the Go expression for calling an SSR function.
func ssrCall(alias string, hasContext bool) string {
	if hasContext {
		return fmt.Sprintf("structToMap(%s.SSR(ctx))", alias)
	}
	return fmt.Sprintf("structToMap(%s.SSR())", alias)
}

// ParseModulePath extracts the module path from go.mod content.
// Returns an empty string if no module directive is found.
func ParseModulePath(goModContent []byte) string {
	for _, line := range strings.Split(string(goModContent), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
