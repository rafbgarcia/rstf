package basic_test

import (
	"fmt"
	"net"
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
	"github.com/rafbgarcia/rstf/internal/codegen"
)

func TestHydration(t *testing.T) {
	root := projectRoot()

	// Step 1: Run codegen.
	result, err := codegen.Generate(root)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Join(root, ".rstf")) })

	// Step 2: Bundle client JS for each entry.
	for _, entryPath := range result.Entries {
		base := filepath.Base(entryPath)
		name := base[:len(base)-len(".entry.tsx")]
		outDir := filepath.Join(root, ".rstf", "static", name)

		if err := os.MkdirAll(outDir, 0755); err != nil {
			t.Fatalf("creating %s: %v", outDir, err)
		}

		outFile := filepath.Join(outDir, "bundle.js")
		cmd := exec.Command("bun", "build", entryPath, "--outfile", outFile)
		cmd.Dir = root
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("bundling %s: %v", entryPath, err)
		}
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

// freePort finds an available TCP port by binding to :0 then closing.
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("finding free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return fmt.Sprintf("%d", port)
}

// waitForServer polls a URL until it returns a 200 response or the timeout expires.
func waitForServer(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("server at %s not ready after %s", url, timeout)
}

