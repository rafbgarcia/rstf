package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "rstf",
	Short: "rstf framework CLI",
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start the development server",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		runDev(port)
	},
}

func init() {
	devCmd.Flags().String("port", "3000", "HTTP server port")
	rootCmd.AddCommand(devCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
