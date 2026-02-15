package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/rafbgarcia/rstf/internal/codegen"
)

func runDev() {
	// Step 1: Run codegen.
	fmt.Print("  Codegen ......... ")
	routeCount, err := codegen.Generate(".")
	if err != nil {
		fmt.Println("FAILED")
		fmt.Fprintf(os.Stderr, "codegen error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("done (%d routes)\n", routeCount)

	// Step 2: Compile and run the generated server.
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
