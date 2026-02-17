package bundler

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// BundleEntries bundles all hydration entry files into client-side JS bundles
// using esbuild's Go API (single in-process call, no child processes).
// Each entry produces .rstf/static/{name}/bundle.js.
//
// projectRoot is the path to the project directory (resolved to absolute).
// entries maps routeDir -> absolute path to .entry.tsx file.
func BundleEntries(projectRoot string, entries map[string]string) error {
	if len(entries) == 0 {
		return nil
	}

	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("resolving project root: %w", err)
	}

	var entryPoints []api.EntryPoint
	for _, entryPath := range entries {
		base := filepath.Base(entryPath)
		name := base[:len(base)-len(".entry.tsx")]
		entryPoints = append(entryPoints, api.EntryPoint{
			InputPath:  entryPath,
			OutputPath: name + "/bundle",
		})
	}

	result := api.Build(api.BuildOptions{
		EntryPointsAdvanced: entryPoints,
		Bundle:              true,
		Outdir:              filepath.Join(absRoot, ".rstf", "static"),
		Platform:            api.PlatformBrowser,
		JSX:                 api.JSXAutomatic,
		AbsWorkingDir:       absRoot,
		Write:               true,
	})

	if len(result.Errors) > 0 {
		var msgs []string
		for _, msg := range result.Errors {
			text := msg.Text
			if msg.Location != nil {
				text = fmt.Sprintf("%s:%d:%d: %s", msg.Location.File, msg.Location.Line, msg.Location.Column, msg.Text)
			}
			msgs = append(msgs, text)
		}
		return fmt.Errorf("esbuild errors:\n%s", strings.Join(msgs, "\n"))
	}

	return nil
}
