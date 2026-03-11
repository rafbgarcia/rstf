package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rafbgarcia/rstf/internal/codegen"
	"github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build a deployable dist directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild()
		},
	}
}

func runBuild() error {
	appName, err := currentAppName()
	if err != nil {
		return err
	}

	gen, err := codegen.NewGenerator(".")
	if err != nil {
		return fmt.Errorf("codegen init error: %w", err)
	}

	fmt.Print("  Codegen ......... ")
	result, err := gen.Generate()
	if err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("codegen error: %w", err)
	}
	fmt.Printf("done (%d routes)\n", result.RouteCount)

	fmt.Print("  Client bundles .. ")
	if err := buildClientBundles(result); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("bundling error: %w", err)
	}
	fmt.Println("done")

	if _, err := os.Stat("main.css"); err == nil {
		fmt.Print("  CSS ............. ")
		if err := buildCSS(); err != nil {
			fmt.Println("FAILED")
			return fmt.Errorf("css error: %w", err)
		}
		fmt.Println("done")
	}

	distDir := "dist"
	if err := os.RemoveAll(distDir); err != nil {
		return fmt.Errorf("removing dist: %w", err)
	}
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return fmt.Errorf("creating dist: %w", err)
	}

	fmt.Print("  Dist layout ..... ")
	if err := copyProjectToDist(".", distDir); err != nil {
		fmt.Println("FAILED")
		return err
	}
	if err := copyDir("rstf", filepath.Join(distDir, "rstf")); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("copying generated assets: %w", err)
	}
	if err := copyDir("node_modules", filepath.Join(distDir, "node_modules")); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("copying node_modules: %w", err)
	}
	fmt.Println("done")

	fmt.Print("  Go binary ....... ")
	outputPath := filepath.Join(distDir, appName)
	build := exec.Command("go", "build", "-o", outputPath, "./rstf/server_gen.go")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("building server binary: %w", err)
	}
	fmt.Printf("done (%s)\n", outputPath)

	fmt.Println("\n  Build complete. Run `cd dist && ./" + appName + "`.")
	return nil
}
