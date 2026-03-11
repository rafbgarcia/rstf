package main

import "github.com/spf13/cobra"

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "rstf",
		Short: "rstf framework CLI",
	}

	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newDevCmd())
	rootCmd.AddCommand(newBuildCmd())

	return rootCmd
}
