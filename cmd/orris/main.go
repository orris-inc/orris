package main

import (
	"os"

	"github.com/spf13/cobra"

	"orris/internal/interfaces/cli/migrate"
	"orris/internal/interfaces/cli/server"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "orris",
		Short: "Orris - A modern Go application",
		Long:  `Orris is a production-ready Go application with built-in server, migration tools, and administrative commands.`,
	}

	rootCmd.AddCommand(
		server.NewCommand(),
		migrate.NewCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
