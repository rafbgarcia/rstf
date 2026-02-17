package codegen

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
)

// importRe matches ES module imports with relative paths:
//
//	import { Foo } from "./bar"
//	import { Foo } from "../shared/ui/thing"
//
// It captures the path specifier (group 1). Bare specifiers like "react" or
// "@rstf/dashboard" don't start with ./ or ../ and are naturally excluded.
var importRe = regexp.MustCompile(`from\s+['"](\.\.?/[^'"]+)['"]`)

// fsCache provides thread-safe caching for filesystem operations used during
// dependency analysis. Shared across all AnalyzeDeps calls to avoid redundant
// reads when multiple routes import the same TSX files or share directories.
type fsCache struct {
	mu    sync.Mutex
	files map[string][]byte // absPath → file content
	hasGo map[string]bool   // dir → has .go files
}

// newFSCache creates an empty filesystem cache.
func newFSCache() *fsCache {
	return &fsCache{
		files: make(map[string][]byte),
		hasGo: make(map[string]bool),
	}
}

// readFile returns cached file content, reading from disk on first access.
func (c *fsCache) readFile(absPath string) ([]byte, error) {
	c.mu.Lock()
	if content, ok := c.files[absPath]; ok {
		c.mu.Unlock()
		return content, nil
	}
	c.mu.Unlock()

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.files[absPath] = content
	c.mu.Unlock()
	return content, nil
}

// dirHasGoFile reports whether dir contains at least one .go file, with caching.
func (c *fsCache) dirHasGoFile(dir string) bool {
	c.mu.Lock()
	if result, ok := c.hasGo[dir]; ok {
		c.mu.Unlock()
		return result
	}
	c.mu.Unlock()

	matches, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	result := len(matches) > 0

	c.mu.Lock()
	c.hasGo[dir] = result
	c.mu.Unlock()
	return result
}

// AnalyzeDeps discovers which directories contain .go files (server data) for
// a given TSX entry file. It recursively follows local relative imports in .tsx
// files, checking each resolved file's directory for .go files.
//
// projectRoot is the absolute path to the project root.
// entryPath is the path to the TSX entry file, relative to projectRoot
// (e.g. "routes/dashboard/index.tsx").
// cache is an optional *fsCache for sharing reads across multiple calls. Pass
// nil to use no cache (each call reads from disk independently).
//
// Returns directory paths relative to projectRoot, sorted alphabetically.
// The entry file's own directory is included if it contains a .go file.
// The layout ("main") is NOT included — the caller adds it.
func AnalyzeDeps(projectRoot string, entryPath string, cache *fsCache) ([]string, error) {
	absEntry := filepath.Join(projectRoot, entryPath)

	visited := map[string]bool{}
	goDirs := map[string]bool{}

	if err := walkImports(projectRoot, absEntry, visited, goDirs, cache); err != nil {
		return nil, err
	}

	result := make([]string, 0, len(goDirs))
	for dir := range goDirs {
		result = append(result, dir)
	}
	sort.Strings(result)
	return result, nil
}

// walkImports reads a .tsx file, extracts local imports, checks for .go files,
// and recurses into imported .tsx files.
func walkImports(projectRoot, absFilePath string, visited map[string]bool, goDirs map[string]bool, cache *fsCache) error {
	if visited[absFilePath] {
		return nil
	}
	visited[absFilePath] = true

	var content []byte
	var err error
	if cache != nil {
		content, err = cache.readFile(absFilePath)
	} else {
		content, err = os.ReadFile(absFilePath)
	}
	if err != nil {
		return err
	}

	// Check if this file's directory has .go files.
	dir := filepath.Dir(absFilePath)
	relDir, err := filepath.Rel(projectRoot, dir)
	if err != nil {
		return err
	}
	var hasGo bool
	if cache != nil {
		hasGo = cache.dirHasGoFile(dir)
	} else {
		hasGo = dirHasGoFile(dir)
	}
	if hasGo {
		goDirs[filepath.ToSlash(relDir)] = true
	}

	// Extract and follow local imports.
	specifiers := extractLocalImports(content)
	for _, spec := range specifiers {
		resolved := resolveImportPath(dir, spec)
		if resolved == "" {
			continue
		}
		if err := walkImports(projectRoot, resolved, visited, goDirs, cache); err != nil {
			return err
		}
	}
	return nil
}

// extractLocalImports returns relative import specifiers from TSX/TS content.
// Only imports starting with "./" or "../" are returned.
func extractLocalImports(content []byte) []string {
	matches := importRe.FindAllSubmatch(content, -1)
	specifiers := make([]string, 0, len(matches))
	for _, m := range matches {
		specifiers = append(specifiers, string(m[1]))
	}
	return specifiers
}

// resolveImportPath resolves a relative import specifier to an absolute .tsx
// file path. Returns "" if no matching file is found.
//
// Resolution order:
//  1. {specifier}.tsx
//  2. {specifier}/index.tsx
func resolveImportPath(baseDir, specifier string) string {
	abs := filepath.Join(baseDir, specifier)

	candidates := []string{
		abs + ".tsx",
		filepath.Join(abs, "index.tsx"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

// dirHasGoFile reports whether dir contains at least one .go file.
func dirHasGoFile(dir string) bool {
	matches, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	return len(matches) > 0
}

