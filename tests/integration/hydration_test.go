package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/rafbgarcia/rstf/internal/bundler"
	"github.com/rafbgarcia/rstf/internal/codegen"
)

func TestHydration(t *testing.T) {
	root := testProjectRoot()

	// Step 1: Run codegen.
	result, err := codegen.Generate(root)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Join(root, ".rstf")) })

	// Step 2: Bundle client JS for each entry.
	if err := bundler.BundleEntries(root, result.Entries); err != nil {
		t.Fatalf("bundling: %v", err)
	}

	// Step 3: Pick a free port.
	port := freePort(t)

	// Step 4: Build the generated server first to catch compilation errors.
	build := exec.Command("go", "build", "-o", filepath.Join(root, ".rstf", "server"), "./.rstf/server_gen.go")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compiling server: %v\n%s", err, out)
	}

	// Step 5: Start the compiled server. The generated server handles SIGINT
	// gracefully — it stops the Bun sidecar before exiting.
	server := exec.Command(filepath.Join(root, ".rstf", "server"), "--port", port)
	server.Dir = root
	server.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := server.Start(); err != nil {
		t.Fatalf("starting server: %v", err)
	}
	t.Cleanup(func() {
		server.Process.Signal(syscall.SIGINT)
		server.Wait()
	})

	// Step 6: Wait for the server to be ready.
	baseURL := fmt.Sprintf("http://localhost:%s", port)
	waitForServer(t, baseURL+"/dashboard", 30*time.Second)

	// Step 7: Launch headless browser.
	u := launcher.New().Headless(true).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	t.Cleanup(func() { browser.MustClose() })

	page := browser.MustPage(baseURL + "/dashboard")
	page.MustWaitStable()

	// Step 8: Verify SSR content is present.
	body := page.MustElement("body").MustText()
	for _, expected := range []string{
		"Welcome to the dashboard!",
		"First Post",
		"Count: 0",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("page missing SSR content %q\n\nbody text:\n%s", expected, body)
		}
	}

	// Step 9: Click the counter button and verify hydration.
	btn := page.MustElement("[data-testid=counter]")
	btn.MustClick()
	page.MustWaitStable()

	btnText := btn.MustText()
	if btnText != "Count: 1" {
		t.Errorf("after click, expected button text %q, got %q", "Count: 1", btnText)
	}
}

// TestCSS verifies the full CSS pipeline: PostCSS/Tailwind processing, static
// serving, link tag injection, and computed styles in the browser.
func TestCSS(t *testing.T) {
	root := testProjectRoot()

	// Step 1: Run codegen.
	result, err := codegen.Generate(root)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Join(root, ".rstf")) })

	// Step 2: Bundle client JS.
	if err := bundler.BundleEntries(root, result.Entries); err != nil {
		t.Fatalf("bundling: %v", err)
	}

	// Step 3: Build CSS via PostCSS (same approach as dev.go's buildCSSWithPostCSS).
	outFile := filepath.Join(".rstf", "static", "main.css")
	script := `import { readFileSync, writeFileSync, mkdirSync } from "fs";
import { resolve } from "path";
import { pathToFileURL } from "url";
import postcss from "postcss";

const configPath = resolve("postcss.config.mjs");
const { default: config } = await import(pathToFileURL(configPath).href);

const plugins = await Promise.all(
  Object.entries(config.plugins || {}).map(async ([name, opts]) => {
    const mod = await import(name);
    return (mod.default || mod)(typeof opts === "object" ? opts : {});
  })
);

const css = readFileSync(resolve("main.css"), "utf8");
const result = await postcss(plugins).process(css, {
  from: resolve("main.css"),
  to: resolve("` + outFile + `"),
});

mkdirSync(resolve(".rstf/static"), { recursive: true });
writeFileSync(resolve("` + outFile + `"), result.css);
`
	scriptPath := filepath.Join(root, ".rstf", "build-css.mjs")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("writing build-css.mjs: %v", err)
	}

	cmd := exec.Command("bun", "run", scriptPath)
	cmd.Dir = root
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("PostCSS build: %v", err)
	}

	// Step 4: Verify the built CSS file exists and contains expected output.
	cssOutput, err := os.ReadFile(filepath.Join(root, ".rstf", "static", "main.css"))
	if err != nil {
		t.Fatalf("reading built CSS: %v", err)
	}
	cssStr := string(cssOutput)

	// Tailwind should produce the text-blue-500 utility (used in dashboard TSX).
	if !strings.Contains(cssStr, "text-blue-500") {
		t.Errorf("built CSS missing Tailwind utility 'text-blue-500'\n\nCSS output (first 500 chars):\n%s", cssStr[:min(500, len(cssStr))])
	}
	// Custom rule from main.css should survive PostCSS processing.
	if !strings.Contains(cssStr, ".custom-red") {
		t.Errorf("built CSS missing custom rule '.custom-red'\n\nCSS output (first 500 chars):\n%s", cssStr[:min(500, len(cssStr))])
	}

	// Step 5: Build and start the generated server.
	port := freePort(t)
	build := exec.Command("go", "build", "-o", filepath.Join(root, ".rstf", "server"), "./.rstf/server_gen.go")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("compiling server: %v\n%s", err, out)
	}

	server := exec.Command(filepath.Join(root, ".rstf", "server"), "--port", port)
	server.Dir = root
	server.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := server.Start(); err != nil {
		t.Fatalf("starting server: %v", err)
	}
	t.Cleanup(func() {
		server.Process.Signal(syscall.SIGINT)
		server.Wait()
	})

	baseURL := fmt.Sprintf("http://localhost:%s", port)
	waitForServer(t, baseURL+"/dashboard", 30*time.Second)

	// Step 6: Verify the HTML response contains the CSS link tag.
	resp, err := http.Get(baseURL + "/dashboard")
	if err != nil {
		t.Fatalf("GET /dashboard: %v", err)
	}
	defer resp.Body.Close()
	htmlBytes, _ := io.ReadAll(resp.Body)
	htmlStr := string(htmlBytes)

	if !strings.Contains(htmlStr, `<link rel="stylesheet" href="/.rstf/static/main.css">`) {
		t.Errorf("HTML missing CSS link tag\n\nHTML:\n%s", htmlStr)
	}

	// Step 7: Verify the CSS file is served by the static handler.
	cssResp, err := http.Get(baseURL + "/.rstf/static/main.css")
	if err != nil {
		t.Fatalf("GET /.rstf/static/main.css: %v", err)
	}
	defer cssResp.Body.Close()
	if cssResp.StatusCode != 200 {
		t.Fatalf("CSS static file returned status %d", cssResp.StatusCode)
	}

	// Step 8: Launch headless browser and verify computed styles.
	u := launcher.New().Headless(true).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	t.Cleanup(func() { browser.MustClose() })

	page := browser.MustPage(baseURL + "/dashboard")
	page.MustWaitStable()

	// Tailwind's text-blue-500 should set the color on the h2.
	h2 := page.MustElement("h2.text-blue-500")
	color := h2.MustEval(`() => getComputedStyle(this).color`).String()

	// Tailwind v4 text-blue-500 produces oklch, which browsers compute to rgb.
	// Accept any non-black, non-inherited color as proof the stylesheet loaded.
	if color == "" || color == "rgb(0, 0, 0)" {
		t.Errorf("expected Tailwind text-blue-500 to apply a color, got %q", color)
	}
}
