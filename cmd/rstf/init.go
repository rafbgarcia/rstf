package main

import (
	"fmt"

	"github.com/rafbgarcia/rstf/internal/scaffold"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var module string
	var skipInstall bool

	cmd := &cobra.Command{
		Use:   "init <name>",
		Short: "Create a new rstf app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := scaffold.DeriveConfig(args[0], module)
			if err != nil {
				return err
			}

			fmt.Printf("  Creating app ..... %s\n", cfg.TargetDir)
			fmt.Printf("  Module ........... %s\n", cfg.Module)
			fmt.Printf("  Package .......... %s\n", cfg.PackageName)

			if err := scaffold.Create(cfg, scaffold.Options{
				InstallDependencies: !skipInstall,
			}); err != nil {
				return err
			}

			if skipInstall {
				fmt.Println("\n  App scaffolded. Run `npm install` and then `npm run dev` inside the app directory.")
			} else {
				fmt.Println("\n  App ready. Run `cd " + cfg.Name + " && npm run dev`.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&module, "module", "", "Go module path for the generated app")
	cmd.Flags().BoolVar(&skipInstall, "skip-install", false, "Write scaffold files without running npm install or go mod tidy")

	return cmd
}
