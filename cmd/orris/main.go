package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/orris-inc/orris/internal/interfaces/cli/migrate"
	"github.com/orris-inc/orris/internal/interfaces/cli/server"
)

// version is set via ldflags at build time.
var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "orris",
		Short:   "Orris - A modern Go application",
		Long:    `Orris is a production-ready Go application with built-in server, migration tools, and administrative commands.`,
		Version: version,
	}

	rootCmd.AddCommand(
		server.NewCommand(),
		migrate.NewCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
