// Package codegen parses Go route files and generates TypeScript type
// declarations, runtime modules, and the Go server entry point.
//
// Files in this package:
//   - parse.go: Go AST parsing, shared types
//   - typescript.go: TypeScript output (.d.ts, runtime modules)
//   - server.go: Go server entry point generation
//   - imports.go: TypeScript import analysis for dependency discovery
package codegen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// RouteFunc represents a parsed route handler function (e.g. SSR).
type RouteFunc struct {
	Name       string // Function name: "SSR"
	ReturnType string // Name of the return struct (e.g. "ServerData")
	HasContext  bool   // Whether the function accepts a *rstf.Context parameter
}

// StructDef represents a parsed Go struct and its fields.
type StructDef struct {
	Name   string
	Fields []StructField
}

// StructField represents a single field in a Go struct.
type StructField struct {
	Name     string // Go field name
	JSONName string // Name from json tag (used in TS output)
	Type     string // Mapped TypeScript type
}

// RouteFile is the result of parsing a single route directory.
type RouteFile struct {
	Dir     string      // Relative directory path (e.g. "dashboard")
	Package string      // Go package name
	Funcs   []RouteFunc // Route handler functions found
	Structs []StructDef // Struct types referenced by route functions
}

// routeFuncNames are the exported function names the framework recognizes.
var routeFuncNames = map[string]bool{
	"SSR": true,
}

// ParseDir walks rootDir and parses all Go route files.
// It returns a RouteFile for each directory that contains route handler functions.
func ParseDir(rootDir string) ([]RouteFile, error) {
	dirFiles := map[string][]string{}

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".rstf", ".git", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		dir := filepath.Dir(path)
		dirFiles[dir] = append(dirFiles[dir], path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", rootDir, err)
	}

	var results []RouteFile
	for dir, files := range dirFiles {
		rf, err := parseRouteDir(rootDir, dir, files)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", dir, err)
		}
		if rf != nil {
			results = append(results, *rf)
		}
	}
	return results, nil
}

// ParseSingleDir parses a single directory's Go files and returns a RouteFile.
// Returns nil if the directory doesn't exist or has no .go files with route functions.
// absDir must be an absolute path; rootDir is the project root (also absolute).
func ParseSingleDir(rootDir, absDir string) (*RouteFile, error) {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var goFiles []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".go" {
			goFiles = append(goFiles, filepath.Join(absDir, e.Name()))
		}
	}
	if len(goFiles) == 0 {
		return nil, nil
	}

	return parseRouteDir(rootDir, absDir, goFiles)
}

// parseRouteDir parses all Go files in a single route directory.
func parseRouteDir(rootDir, dir string, files []string) (*RouteFile, error) {
	fset := token.NewFileSet()
	var allFiles []*ast.File

	for _, path := range files {
		f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		allFiles = append(allFiles, f)
	}

	if len(allFiles) == 0 {
		return nil, nil
	}

	// Collect all struct definitions from the package.
	structDefs := map[string]StructDef{}
	for _, f := range allFiles {
		for name, def := range extractStructs(f) {
			structDefs[name] = def
		}
	}

	// Find route handler functions.
	var funcs []RouteFunc
	referencedStructs := map[string]bool{}

	for _, f := range allFiles {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			if !routeFuncNames[fn.Name.Name] {
				continue
			}
			rf, refs := parseRouteFunc(fn)
			if rf != nil {
				funcs = append(funcs, *rf)
				for _, r := range refs {
					referencedStructs[r] = true
				}
			}
		}
	}

	if len(funcs) == 0 {
		return nil, nil
	}

	// Resolve transitive struct references (e.g. ServerData -> Post, Author).
	allRefs := resolveTransitiveStructs(referencedStructs, structDefs)
	var structs []StructDef
	for name := range allRefs {
		if sd, ok := structDefs[name]; ok {
			structs = append(structs, sd)
		}
	}

	relDir, _ := filepath.Rel(rootDir, dir)

	return &RouteFile{
		Dir:     relDir,
		Package: allFiles[0].Name.Name,
		Funcs:   funcs,
		Structs: structs,
	}, nil
}

// parseRouteFunc extracts the return type from a route function.
// SSR must return a single struct type. Returns nil if the function doesn't match.
// Detects if the first input parameter is a *rstf.Context (regardless of import alias).
func parseRouteFunc(fn *ast.FuncDecl) (*RouteFunc, []string) {
	results := fn.Type.Results
	if results == nil || len(results.List) != 1 {
		return nil, nil // Must have exactly one return value
	}

	field := results.List[0]
	typeName, isSlice := resolveType(field.Type)
	if typeName == "" || isSlice || isPrimitiveGoType(typeName) {
		return nil, nil // Must be a named struct (not primitive, not slice)
	}

	hasContext := false
	if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
		hasContext = isContextParam(fn.Type.Params.List[0].Type)
	}

	return &RouteFunc{
		Name:       fn.Name.Name,
		ReturnType: typeName,
		HasContext:  hasContext,
	}, []string{typeName}
}

// isContextParam checks if a type expression is *<pkg>.Context.
// Matches any import alias (e.g. *rstf.Context, *fw.Context).
func isContextParam(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return sel.Sel.Name == "Context"
}

// resolveType returns the type name and whether it's a slice.
func resolveType(expr ast.Expr) (string, bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, false
	case *ast.ArrayType:
		name, _ := resolveType(t.Elt)
		return name, true
	case *ast.StarExpr:
		return resolveType(t.X)
	default:
		return "", false
	}
}

// extractStructs finds all type Foo struct{} declarations in a file.
func extractStructs(f *ast.File) map[string]StructDef {
	structs := map[string]StructDef{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			sd := StructDef{Name: ts.Name.Name}
			for _, field := range st.Fields.List {
				if len(field.Names) == 0 {
					continue // Skip embedded fields
				}
				fieldName := field.Names[0].Name
				if !ast.IsExported(fieldName) {
					continue
				}
				jsonName := jsonTagName(field)
				if jsonName == "" {
					jsonName = lcFirst(fieldName)
				}
				if jsonName == "-" {
					continue
				}
				typeName, isSlice := resolveType(field.Type)
				tsType := goTypeToTS(typeName, isSlice)

				sd.Fields = append(sd.Fields, StructField{
					Name:     fieldName,
					JSONName: jsonName,
					Type:     tsType,
				})
			}
			structs[ts.Name.Name] = sd
		}
	}
	return structs
}

// jsonTagName extracts the field name from a `json:"name"` tag.
func jsonTagName(field *ast.Field) string {
	if field.Tag == nil {
		return ""
	}
	tag := strings.Trim(field.Tag.Value, "`")
	for _, part := range strings.Split(tag, " ") {
		if strings.HasPrefix(part, "json:\"") {
			val := strings.TrimPrefix(part, "json:\"")
			val = strings.TrimSuffix(val, "\"")
			name, _, _ := strings.Cut(val, ",")
			return name
		}
	}
	return ""
}

// goTypeToTS maps a Go type name to its TypeScript equivalent.
func goTypeToTS(goType string, isSlice bool) string {
	var tsType string
	switch goType {
	case "string":
		tsType = "string"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		tsType = "number"
	case "bool":
		tsType = "boolean"
	default:
		tsType = goType // Struct name used as-is
	}
	if isSlice {
		tsType += "[]"
	}
	return tsType
}

func isPrimitiveGoType(name string) bool {
	switch name {
	case "string", "bool",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return true
	}
	return false
}

func lcFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// ucFirst uppercases the first character of a string.
func ucFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// resolveTransitiveStructs walks struct field types to find all transitively
// referenced structs. For example, ServerData{Posts []Post, Author Author}
// references both Post and Author.
func resolveTransitiveStructs(roots map[string]bool, allStructs map[string]StructDef) map[string]bool {
	result := map[string]bool{}
	queue := make([]string, 0, len(roots))
	for name := range roots {
		queue = append(queue, name)
	}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if result[name] {
			continue
		}
		result[name] = true
		sd, ok := allStructs[name]
		if !ok {
			continue
		}
		for _, f := range sd.Fields {
			typeName := strings.TrimSuffix(f.Type, "[]")
			if _, exists := allStructs[typeName]; exists && !result[typeName] {
				queue = append(queue, typeName)
			}
		}
	}
	return result
}
