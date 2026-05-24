package main

import (
	"fmt"
	"os"

	"github.com/supuwoerc/gapi-server/cmd/cli/commands/system"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gapi-cli",
	Short: "GAPI Server CLI tools",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func init() {
	system.Register(rootCmd, WireCli)
}
