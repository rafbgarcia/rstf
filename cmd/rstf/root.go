package main

import (
	"fmt"

	"github.com/rafbgarcia/rstf/internal/release"
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "rstf",
		Short:   "rstf framework CLI",
		Version: release.Version,
	}

	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newDevCmd())
	rootCmd.AddCommand(newBuildCmd())
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the rstf release version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), release.Version)
		},
	})

	return rootCmd
}
