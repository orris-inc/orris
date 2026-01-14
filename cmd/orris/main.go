package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/orris-inc/orris/internal/interfaces/cli/migrate"
	"github.com/orris-inc/orris/internal/interfaces/cli/server"
	"github.com/orris-inc/orris/internal/shared/version"
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "orris",
		Short:   "Orris - A modern Go application",
		Long:    `Orris is a production-ready Go application with built-in server, migration tools, and administrative commands.`,
		Version: version.Current,
	}

	// Enable -v as short flag for --version
	rootCmd.Flags().BoolP("version", "v", false, "version for orris")

	rootCmd.AddCommand(
		server.NewCommand(),
		migrate.NewCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
