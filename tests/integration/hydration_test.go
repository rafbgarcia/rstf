package integration_test

import (
	"bytes"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHydration(t *testing.T) {
	root := testProjectRoot()

	// Step 1: Run codegen.
	result, err := codegen.Generate(root)
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(filepath.Join(root, "rstf")) })
	require.NoError(t, tidyGoModule(root))

	// Step 2: Bundle client JS for each entry.
	require.NoError(t, bundler.BundleEntries(root, result.Entries))
	require.NoError(t, bundler.BundleSSREntries(root, result.SSREntries))

	// Step 3: Pick a free port.
	port := freePort(t)

	// Step 4: Build the generated server first to catch compilation errors.
	build := exec.Command("go", "build", "-o", filepath.Join(root, "rstf", "server"), "./rstf/server_gen.go")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		require.FailNowf(t, "compiling server", "compiling server: %v\n%s", err, out)
	}

	// Step 5: Start the compiled server.
	server := exec.Command(filepath.Join(root, "rstf", "server"), "--port", port)
	server.Dir = root
	server.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, server.Start())
	t.Cleanup(func() {
		stopProcessGroup(t, server, 1*time.Second)
	})

	// Step 6: Wait for the server to be ready.
	baseURL := fmt.Sprintf("http://localhost:%s", port)
	waitForServer(t, baseURL+"/get-vs-ssr", 10*time.Second)

	// Step 7: Launch headless browser.
	u := launcher.New().Headless(true).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	t.Cleanup(func() { browser.MustClose() })

	page := browser.MustPage("about:blank")
	page.MustSetExtraHeaders("Accept", "text/html")
	page.MustNavigate(baseURL + "/get-vs-ssr")
	page = page.Timeout(15 * time.Second)
	page.MustWaitStable()

	// Step 8: Verify SSR content is present.
	body := page.MustElement("body").MustText()
	for _, expected := range []string{
		"Welcome to the dashboard!",
		"First Post",
		"Count: 0",
	} {
		assert.Containsf(t, body, expected, "page missing SSR content %q\n\nbody text:\n%s", expected, body)
	}

	// Step 9: Click the counter button and verify hydration.
	btn := page.MustElement("[data-testid=counter]")
	btn.MustClick()
	require.Eventually(t, func() bool {
		el, err := page.Element("[data-testid=counter]")
		if err != nil {
			return false
		}
		text, err := el.Text()
		if err != nil {
			return false
		}
		return text == "Count: 1"
	}, 5*time.Second, 100*time.Millisecond)
}

func TestLiveQueryUpdatesAcrossClients(t *testing.T) {
	root := testProjectRoot()

	result, err := codegen.Generate(root)
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(filepath.Join(root, "rstf")) })
	require.NoError(t, tidyGoModule(root))

	require.NoError(t, bundler.BundleEntries(root, result.Entries))
	require.NoError(t, bundler.BundleSSREntries(root, result.SSREntries))

	port := freePort(t)
	build := exec.Command("go", "build", "-o", filepath.Join(root, "rstf", "server"), "./rstf/server_gen.go")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		require.FailNowf(t, "compiling server", "compiling server: %v\n%s", err, out)
	}

	server := exec.Command(filepath.Join(root, "rstf", "server"), "--port", port)
	server.Dir = root
	server.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, server.Start())
	t.Cleanup(func() {
		stopProcessGroup(t, server, 1*time.Second)
	})

	baseURL := fmt.Sprintf("http://localhost:%s", port)
	waitForServer(t, baseURL+"/live-chat/room-1", 10*time.Second)

	u := launcher.New().Headless(true).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	t.Cleanup(func() { browser.MustClose() })

	pageA := browser.MustPage("about:blank")
	pageA.MustSetExtraHeaders("Accept", "text/html")
	pageA.MustNavigate(baseURL + "/live-chat/room-1")
	pageA = pageA.Timeout(15 * time.Second)
	pageA.MustWaitStable()

	pageB := browser.MustPage("about:blank")
	pageB.MustSetExtraHeaders("Accept", "text/html")
	pageB.MustNavigate(baseURL + "/live-chat/room-1")
	pageB = pageB.Timeout(15 * time.Second)
	pageB.MustWaitStable()

	require.Eventually(t, func() bool {
		el, err := pageA.Element("[data-testid=messages-list]")
		if err != nil {
			return false
		}
		text, err := el.Text()
		return err == nil && strings.Contains(text, "Hello from the server")
	}, 5*time.Second, 100*time.Millisecond)
	require.Eventually(t, func() bool {
		el, err := pageB.Element("[data-testid=messages-list]")
		if err != nil {
			return false
		}
		text, err := el.Text()
		return err == nil && strings.Contains(text, "Hello from the server")
	}, 5*time.Second, 100*time.Millisecond)

	reqBody := bytes.NewBufferString(`{"kind":"mutation","route":"live-chat._id","name":"SendMessage","params":{"id":"room-1"},"input":{"body":"Second message"}}`)
	resp, err := http.Post(baseURL+"/__rstf/rpc", "application/json", reqBody)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.Eventually(t, func() bool {
		el, err := pageB.Element("[data-testid=messages-list]")
		if err != nil {
			return false
		}
		text, err := el.Text()
		return err == nil && strings.Contains(text, "Second message")
	}, 10*time.Second, 100*time.Millisecond)
}

// TestCSS verifies the full CSS pipeline: PostCSS/Tailwind processing, static
// serving, link tag injection, and computed styles in the browser.
func TestCSS(t *testing.T) {
	root := testProjectRoot()

	// Step 1: Run codegen.
	result, err := codegen.Generate(root)
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(filepath.Join(root, "rstf")) })
	require.NoError(t, tidyGoModule(root))

	// Step 2: Bundle client JS.
	require.NoError(t, bundler.BundleEntries(root, result.Entries))
	require.NoError(t, bundler.BundleSSREntries(root, result.SSREntries))

	// Step 3: Build CSS via PostCSS (same approach as dev.go's buildCSSWithPostCSS).
	outFile := filepath.Join("rstf", "static", "main.css")
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

mkdirSync(resolve("rstf/static"), { recursive: true });
writeFileSync(resolve("` + outFile + `"), result.css);
`
	scriptPath := filepath.Join(root, "rstf", "build-css.mjs")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	cmd := exec.Command("node", scriptPath)
	cmd.Dir = root
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())

	// Step 4: Verify the built CSS file exists and contains expected output.
	cssOutput, err := os.ReadFile(filepath.Join(root, "rstf", "static", "main.css"))
	require.NoError(t, err)
	cssStr := string(cssOutput)

	// Tailwind should produce the text-blue-500 utility (used in dashboard TSX).
	assert.Containsf(t, cssStr, "text-blue-500", "built CSS missing Tailwind utility 'text-blue-500'\n\nCSS output (first 500 chars):\n%s", cssStr[:min(500, len(cssStr))])
	// Custom rule from main.css should survive PostCSS processing.
	assert.Containsf(t, cssStr, ".custom-red", "built CSS missing custom rule '.custom-red'\n\nCSS output (first 500 chars):\n%s", cssStr[:min(500, len(cssStr))])

	// Step 5: Build and start the generated server.
	port := freePort(t)
	build := exec.Command("go", "build", "-o", filepath.Join(root, "rstf", "server"), "./rstf/server_gen.go")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		require.FailNowf(t, "compiling server", "compiling server: %v\n%s", err, out)
	}

	server := exec.Command(filepath.Join(root, "rstf", "server"), "--port", port)
	server.Dir = root
	server.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, server.Start())
	t.Cleanup(func() {
		stopProcessGroup(t, server, 1*time.Second)
	})

	baseURL := fmt.Sprintf("http://localhost:%s", port)
	waitForServer(t, baseURL+"/get-vs-ssr", 10*time.Second)

	// Step 6: Verify the HTML response contains the CSS link tag.
	resp, err := http.Get(baseURL + "/get-vs-ssr")
	require.NoError(t, err)
	defer resp.Body.Close()
	htmlBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	htmlStr := string(htmlBytes)

	assert.Contains(t, htmlStr, `<link rel="stylesheet" href="/rstf/static/main.css">`)

	// Step 7: Verify the CSS file is served by the static handler.
	cssResp, err := http.Get(baseURL + "/rstf/static/main.css")
	require.NoError(t, err)
	defer cssResp.Body.Close()
	require.Equal(t, 200, cssResp.StatusCode)

	// Step 8: Launch headless browser and verify computed styles.
	u := launcher.New().Headless(true).MustLaunch()
	browser := rod.New().ControlURL(u).MustConnect()
	t.Cleanup(func() { browser.MustClose() })

	page := browser.MustPage("about:blank")
	page.MustSetExtraHeaders("Accept", "text/html")
	page.MustNavigate(baseURL + "/get-vs-ssr")
	page = page.Timeout(5 * time.Second)
	page.MustWaitStable()

	// Tailwind's text-blue-500 should set the color on the h2.
	h2 := page.MustElement("h2.text-blue-500")
	color := h2.MustEval(`() => getComputedStyle(this).color`).String()

	// Tailwind v4 text-blue-500 produces oklch, which browsers compute to rgb.
	// Accept any non-black, non-inherited color as proof the stylesheet loaded.
	assert.NotEmpty(t, color)
	assert.NotEqual(t, "rgb(0, 0, 0)", color)
}
