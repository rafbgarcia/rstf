package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rafbgarcia/rstf/internal/codegen"
)

func TestCodegen(t *testing.T) {
	root := testProjectRoot()

	result, err := codegen.Generate(root)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Join(root, ".rstf")) })

	if result.RouteCount != 6 {
		t.Errorf("expected 6 routes, got %d", result.RouteCount)
	}

	// Verify generated files exist.
	expectedFiles := []string{
		".rstf/server_gen.go",
		".rstf/types/main.d.ts",
		".rstf/types/get-vs-ssr.d.ts",
		".rstf/generated/main.ts",
		".rstf/generated/routes/get-vs-ssr.ts",
		".rstf/entries/get-vs-ssr.entry.tsx",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(root, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Verify DTS content.
	mainDTS, _ := os.ReadFile(filepath.Join(root, ".rstf/types/main.d.ts"))
	if !strings.Contains(string(mainDTS), "declare namespace Main") {
		t.Errorf("main.d.ts missing Main namespace:\n%s", mainDTS)
	}
	if !strings.Contains(string(mainDTS), "appName: string") {
		t.Errorf("main.d.ts missing appName field:\n%s", mainDTS)
	}

	dashDTS, _ := os.ReadFile(filepath.Join(root, ".rstf/types/get-vs-ssr.d.ts"))
	if !strings.Contains(string(dashDTS), "declare namespace RoutesGetVsSsr") {
		t.Errorf("get-vs-ssr.d.ts missing RoutesGetVsSsr namespace:\n%s", dashDTS)
	}
	if !strings.Contains(string(dashDTS), "posts: Post[]") {
		t.Errorf("get-vs-ssr.d.ts missing posts field:\n%s", dashDTS)
	}

	// Verify runtime module has dual-mode initialization.
	dashMod, _ := os.ReadFile(filepath.Join(root, ".rstf/generated/routes/get-vs-ssr.ts"))
	dashModStr := string(dashMod)
	if !strings.Contains(dashModStr, `typeof window !== "undefined"`) {
		t.Errorf("get-vs-ssr.ts missing dual-mode init:\n%s", dashModStr)
	}
	if !strings.Contains(dashModStr, `__RSTF_SERVER_DATA__["routes/get-vs-ssr"]`) {
		t.Errorf("get-vs-ssr.ts missing server data key:\n%s", dashModStr)
	}

	// Verify server_gen.go content.
	serverCode, _ := os.ReadFile(filepath.Join(root, ".rstf/server_gen.go"))
	serverStr := string(serverCode)
	for _, expected := range []string{
		"package main",
		`app "github.com/rafbgarcia/rstf/tests/integration/test_project"`,
		`dashboard "github.com/rafbgarcia/rstf/tests/integration/test_project/routes/get-vs-ssr"`,
		`Component: "routes/get-vs-ssr"`,
		`Layout: "main"`,
		"structToMap(app.SSR(ctx))",
		"structToMap(dashboard.SSR(ctx))",
		"func assemblePage(",
		`rt.Handle("/.rstf/static/*"`,
		"assemblePage(html, sd,",
	} {
		if !strings.Contains(serverStr, expected) {
			t.Errorf("server_gen.go missing %q", expected)
		}
	}

	// Verify hydration entry content.
	entryContent, _ := os.ReadFile(filepath.Join(root, ".rstf/entries/get-vs-ssr.entry.tsx"))
	entryStr := string(entryContent)
	for _, expected := range []string{
		`import { hydrateRoot } from "react-dom/client"`,
		`import { View as Layout } from "../../main"`,
		`import { View as Route } from "../../routes/get-vs-ssr"`,
		`import "@rstf/main"`,
		`import "@rstf/routes/get-vs-ssr"`,
		"hydrateRoot(document,",
	} {
		if !strings.Contains(entryStr, expected) {
			t.Errorf("get-vs-ssr.entry.tsx missing %q\n\nFull content:\n%s", expected, entryStr)
		}
	}

	// Verify Entries map is populated.
	if _, ok := result.Entries["routes/get-vs-ssr"]; !ok {
		t.Error("Entries map missing routes/get-vs-ssr key")
	}
}
