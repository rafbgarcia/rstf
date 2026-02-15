package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "dev":
		// Parse --port flag from remaining args.
		devFlags := flag.NewFlagSet("dev", flag.ExitOnError)
		port := devFlags.String("port", "3000", "HTTP server port")
		devFlags.Parse(os.Args[2:])
		runDev(*port)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: rstf <command>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  dev    Start the development server")
}
