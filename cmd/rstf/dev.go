package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rafbgarcia/rstf/internal/bundler"
	"github.com/rafbgarcia/rstf/internal/codegen"
	"github.com/rafbgarcia/rstf/internal/watcher"
	"github.com/spf13/cobra"
)

func newDevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Start the development server",
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetString("port")
			return runDev(port)
		},
	}

	cmd.Flags().String("port", "3000", "HTTP server port")
	return cmd
}

func runDev(port string) error {
	// Step 1: Create generator and run initial codegen.
	gen, err := codegen.NewGenerator(".")
	if err != nil {
		return fmt.Errorf("codegen init error: %w", err)
	}

	fmt.Print("  Codegen ......... ")
	t := time.Now()
	result, err := gen.Generate()
	if err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("codegen error: %w", err)
	}
	fmt.Printf("done (%d routes) [%s]\n", result.RouteCount, fmtDuration(time.Since(t)))

	// Step 2: Bundle client JS for each route.
	fmt.Print("  Client bundles .. ")
	t = time.Now()
	if err := buildClientBundles(result); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("bundling error: %w", err)
	}
	fmt.Printf("done [%s]\n", fmtDuration(time.Since(t)))

	fmt.Print("  SSR bundles ..... ")
	t = time.Now()
	if err := buildSSRBundles(result); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("SSR bundling error: %w", err)
	}
	fmt.Printf("done [%s]\n", fmtDuration(time.Since(t)))

	// Step 3: Build CSS (if main.css exists).
	if _, err := os.Stat("main.css"); err == nil {
		fmt.Print("  CSS ............. ")
		t = time.Now()
		if err := buildCSS(); err != nil {
			fmt.Println("FAILED")
			return fmt.Errorf("css error: %w", err)
		}
		fmt.Printf("done [%s]\n", fmtDuration(time.Since(t)))
	}

	// Step 4: Start the Go HTTP server.
	fmt.Printf("  HTTP server ..... starting on :%s\n", port)
	server := startServer(port)

	// Step 5: Start file watcher.
	fmt.Println("\n  Watching for changes...")

	eventCh := make(chan []watcher.Event, 100)
	w := watcher.New(".", func(batch []watcher.Event) { eventCh <- batch })
	if err := w.Start(); err != nil {
		return fmt.Errorf("watcher error: %w", err)
	}

	// Step 6: Event loop — react to file changes and SIGINT.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case batch := <-eventCh:
			// Classify batch into change kinds.
			var hasGo, hasTsx, hasCss bool
			for _, ev := range batch {
				switch ev.Kind {
				case "go":
					hasGo = true
				case "tsx":
					hasTsx = true
				case "css":
					hasCss = true
				}
			}

			// Print changed files.
			for _, ev := range batch {
				fmt.Printf("\n  [change] %s\n", ev.Path)
			}

			if hasGo || hasTsx {
				server = handleCodeChange(gen, server, &result, port, batch, hasGo)
			}
			if hasCss {
				handleCssChange()
			}

		case <-sigCh:
			w.Stop()
			stopServer(server)
			return nil
		}
	}
}

// handleCodeChange runs incremental codegen, re-bundles, and restarts the
// server if Go files changed or the server_gen.go content changed.
func handleCodeChange(gen *codegen.Generator, server *exec.Cmd, result *codegen.GenerateResult, port string, batch []watcher.Event, hasGo bool) *exec.Cmd {
	if hasGo {
		stopServer(server)
	}

	// Convert watcher events to codegen change events.
	var events []codegen.ChangeEvent
	for _, ev := range batch {
		if ev.Kind == "go" || ev.Kind == "tsx" {
			events = append(events, codegen.ChangeEvent{Path: ev.Path, Kind: ev.Kind})
		}
	}

	fmt.Print("  Codegen ......... ")
	t := time.Now()
	regenResult, err := gen.Regenerate(events)
	if err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "  codegen error: %s\n", err)
		if hasGo {
			fmt.Printf("  HTTP server ..... restarting on :%s\n", port)
			return startServer(port)
		}
		return server
	}
	fmt.Printf("done (%d routes) [%s]\n", regenResult.RouteCount, fmtDuration(time.Since(t)))

	fmt.Print("  Client bundles .. ")
	t = time.Now()
	if err := buildClientBundles(regenResult.GenerateResult); err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "  bundling error: %s\n", err)
	} else {
		fmt.Printf("done [%s]\n", fmtDuration(time.Since(t)))
	}

	fmt.Print("  SSR bundles ..... ")
	t = time.Now()
	if err := buildSSRBundles(regenResult.GenerateResult); err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "  SSR bundling error: %s\n", err)
	} else {
		fmt.Printf("done [%s]\n", fmtDuration(time.Since(t)))
	}

	if err := buildCSS(); err != nil {
		fmt.Fprintf(os.Stderr, "  css error: %s\n", err)
	}

	*result = regenResult.GenerateResult

	if hasGo || regenResult.ServerChanged {
		fmt.Printf("  HTTP server ..... restarting on :%s\n", port)
		return startServer(port)
	}

	return server
}

// handleCssChange rebuilds CSS. No JS rebundle or sidecar invalidation needed
// since CSS is served statically via FileServer.
func handleCssChange() {
	fmt.Print("  CSS ............. ")
	t := time.Now()
	if err := buildCSS(); err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "  css error: %s\n", err)
		return
	}
	fmt.Printf("done [%s]\n", fmtDuration(time.Since(t)))
}

// startServer launches the generated Go server as a child process.
// The process is placed in its own process group so stopServer can kill
// both `go run` and the child binary it spawns.
func startServer(port string) *exec.Cmd {
	cmd := exec.Command("go", "run", "./rstf/server_gen.go", "--port", port)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start server: %s\n", err)
		os.Exit(1)
	}
	return cmd
}

func buildClientBundles(result codegen.GenerateResult) error {
	return bundler.BundleEntries(".", result.Entries)
}

func buildSSRBundles(result codegen.GenerateResult) error {
	return bundler.BundleSSREntries(".", result.SSREntries)
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

// fmtDuration formats a duration as a human-friendly string (e.g. "12ms", "1.3s").
func fmtDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// buildCSS processes main.css if it exists. If a postcss.config.mjs is present,
// it runs PostCSS via a generated build script. Otherwise, it copies main.css
// directly to the static output directory.
func buildCSS() error {
	if _, err := os.Stat("main.css"); os.IsNotExist(err) {
		return nil // no CSS to build
	}

	outDir := filepath.Join("rstf", "static")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating %s: %w", outDir, err)
	}

	outFile := filepath.Join(outDir, "main.css")

	// If a PostCSS config exists, run PostCSS via a build script.
	if _, err := os.Stat("postcss.config.mjs"); err == nil {
		return buildCSSWithPostCSS(outFile)
	}

	// No PostCSS config — copy main.css as-is.
	src, err := os.ReadFile("main.css")
	if err != nil {
		return fmt.Errorf("reading main.css: %w", err)
	}
	if err := os.WriteFile(outFile, src, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outFile, err)
	}
	return nil
}

func currentAppName() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}
	name := filepath.Base(cwd)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "", fmt.Errorf("could not derive app name from %s", cwd)
	}
	return name, nil
}

// buildCSSWithPostCSS writes a small build script to rstf/ and runs it with
// node. The script loads the user's postcss.config.mjs and processes main.css.
func buildCSSWithPostCSS(outFile string) error {
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
	scriptPath := filepath.Join("rstf", "build-css.mjs")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return fmt.Errorf("writing build-css.mjs: %w", err)
	}

	cmd := exec.Command("node", scriptPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("postcss processing: %w", err)
	}
	return nil
}
