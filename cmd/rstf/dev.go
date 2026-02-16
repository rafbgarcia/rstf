package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/rafbgarcia/rstf/internal/codegen"
	"github.com/rafbgarcia/rstf/internal/watcher"
)

func runDev(port string) {
	// Step 1: Run codegen.
	fmt.Print("  Codegen ......... ")
	t := time.Now()
	result, err := codegen.Generate(".")
	if err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "codegen error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("done (%d routes) [%s]\n", result.RouteCount, fmtDuration(time.Since(t)))

	// Step 2: Bundle client JS for each route.
	fmt.Print("  Client bundles .. ")
	t = time.Now()
	if err := bundleEntries(result.Entries); err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "bundling error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("done [%s]\n", fmtDuration(time.Since(t)))

	// Step 3: Start the Go HTTP server.
	fmt.Printf("  HTTP server ..... starting on :%s\n", port)
	server := startServer(port)

	// Step 4: Start file watcher.
	fmt.Println("\n  Watching for changes...")

	eventCh := make(chan watcher.Event, 100)
	w := watcher.New(".", func(e watcher.Event) { eventCh <- e })
	if err := w.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "watcher error: %s\n", err)
		os.Exit(1)
	}

	// Step 5: Event loop — react to file changes and SIGINT.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case ev := <-eventCh:
			switch ev.Kind {
			case "go":
				fmt.Printf("\n  [change] %s\n", ev.Path)
				server = handleGoChange(server, &result, port)
			case "tsx":
				fmt.Printf("\n  [change] %s\n", ev.Path)
				handleTsxChange(result.Entries)
			}

		case <-sigCh:
			w.Stop()
			stopServer(server)
			return
		}
	}
}

// handleGoChange re-runs codegen, re-bundles, kills the server, and restarts.
func handleGoChange(server *exec.Cmd, result *codegen.GenerateResult, port string) *exec.Cmd {
	stopServer(server)

	fmt.Print("  Codegen ......... ")
	t := time.Now()
	newResult, err := codegen.Generate(".")
	if err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "  codegen error: %s\n", err)
		return startServer(port) // restart with old code
	}
	fmt.Printf("done (%d routes) [%s]\n", newResult.RouteCount, fmtDuration(time.Since(t)))

	fmt.Print("  Client bundles .. ")
	t = time.Now()
	if err := bundleEntries(newResult.Entries); err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "  bundling error: %s\n", err)
	} else {
		fmt.Printf("done [%s]\n", fmtDuration(time.Since(t)))
	}

	*result = newResult
	fmt.Printf("  HTTP server ..... restarting on :%s\n", port)
	return startServer(port)
}

// handleTsxChange re-bundles client JS and invalidates the sidecar module cache.
func handleTsxChange(entries map[string]string) {
	fmt.Print("  Client bundles .. ")
	t := time.Now()
	if err := bundleEntries(entries); err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "  bundling error: %s\n", err)
		return
	}
	fmt.Printf("done [%s]\n", fmtDuration(time.Since(t)))

	invalidateSidecar()
}

// startServer launches the generated Go server as a child process.
// The process is placed in its own process group so stopServer can kill
// both `go run` and the child binary it spawns.
func startServer(port string) *exec.Cmd {
	cmd := exec.Command("go", "run", "./.rstf/server_gen.go", "--port", port)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start server: %s\n", err)
		os.Exit(1)
	}
	return cmd
}

// stopServer kills the server's entire process group (go run + child binary),
// then waits for the process to exit.
func stopServer(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	// Kill the entire process group: negative PID targets the group.
	syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	cmd.Wait()
}

// invalidateSidecar reads the sidecar port from .rstf/sidecar.port and POSTs
// to /invalidate to clear the module cache.
func invalidateSidecar() {
	data, err := os.ReadFile(".rstf/sidecar.port")
	if err != nil {
		return // sidecar may not have written port yet
	}
	port := strings.TrimSpace(string(data))
	http.Post("http://localhost:"+port+"/invalidate", "application/json", nil)
}

// fmtDuration formats a duration as a human-friendly string (e.g. "12ms", "1.3s").
func fmtDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// bundleEntries runs bun build for each hydration entry file, producing
// .rstf/static/{name}/bundle.js for each route.
func bundleEntries(entries map[string]string) error {
	for _, entryPath := range entries {
		// Derive the output directory from the entry filename.
		// e.g. .rstf/entries/dashboard.entry.tsx -> .rstf/static/dashboard/
		base := filepath.Base(entryPath)
		name := base[:len(base)-len(".entry.tsx")]
		outDir := filepath.Join(".rstf", "static", name)

		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("creating %s: %w", outDir, err)
		}

		outFile := filepath.Join(outDir, "bundle.js")
		cmd := exec.Command("bun", "build", entryPath, "--outfile", outFile)
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("bundling %s: %w", entryPath, err)
		}
	}
	return nil
}
