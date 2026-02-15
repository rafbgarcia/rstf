package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rafbgarcia/rstf/internal/codegen"
)

func runDev() {
	// Step 1: Run codegen.
	fmt.Print("  Codegen ......... ")
	result, err := codegen.Generate(".")
	if err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "codegen error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("done (%d routes)\n", result.RouteCount)

	// Step 2: Bundle client JS for each route.
	fmt.Print("  Client bundles .. ")
	if err := bundleEntries(result.Entries); err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "bundling error: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("done")

	// Step 3: Compile and run the generated server.
	fmt.Println("  HTTP server ..... starting on :3000")

	cmd := exec.Command("go", "run", "./.rstf/server_gen.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start server: %s\n", err)
		os.Exit(1)
	}

	// Wait for interrupt signal, then forward it to the child process.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	cmd.Process.Signal(syscall.SIGINT)
	cmd.Wait()
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
